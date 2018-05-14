package server

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/usb"
	"github.com/trezor/trezord-go/wire"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const version = "2.0.13"

type session struct {
	path string
	id   string
	dev  usb.Device
	call int32 // atomic
}

type Server struct {
	https *http.Server
	bus   *usb.USB

	sessions      map[string]*session
	sessionsMutex sync.Mutex // for atomic access to sessions

	callInProgress bool       // we cannot make calls and enumeration at the same time
	callMutex      sync.Mutex // for atomic access to callInProgress, plus prevent enumeration
	lastInfos      []usb.Info // when call is in progress, use saved info for enumerating

	mw, dmw         *memorywriter.MemoryWriter
	logger, dlogger *log.Logger
}

func New(bus *usb.USB, logWriter io.Writer, mw, dmw *memorywriter.MemoryWriter, logger, dlogger *log.Logger) (*Server, error) {
	dlogger.Println("http - starting")
	https := &http.Server{
		Addr: "127.0.0.1:21325",
	}
	s := &Server{
		bus:      bus,
		https:    https,
		sessions: make(map[string]*session),

		mw:      mw,
		dmw:     dmw,
		logger:  logger,
		dlogger: dlogger,
	}
	r := mux.NewRouter()

	sr := r.Methods("POST").Subrouter()

	sr.HandleFunc("/", s.Info)
	sr.HandleFunc("/configure", s.Info)
	sr.HandleFunc("/listen", s.Listen)
	sr.HandleFunc("/enumerate", s.Enumerate)
	sr.HandleFunc("/acquire/{path}", s.Acquire)
	sr.HandleFunc("/acquire/{path}/{session}", s.Acquire)
	sr.HandleFunc("/release/{session}", s.Release)
	sr.HandleFunc("/call/{session}", s.Call)
	sr.HandleFunc("/post/{session}", s.Post)

	getsr := r.Methods("GET").Subrouter()
	getsr.HandleFunc("/", s.StatusPage)

	dlogger.Println("http - creating cors validator")
	v, err := corsValidator()
	if err != nil {
		return nil, err
	}

	var h http.Handler = r
	// Restrict cross-origin access.
	h = CORS(v)(h)
	// Log after the request is done, in the Apache format.
	h = handlers.LoggingHandler(logWriter, h)
	// Log when the request is received.
	h = s.logRequest(h)

	https.Handler = h

	dlogger.Println("http - server created")
	return s, nil
}

func (s *Server) logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Printf("%s %s", r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func corsValidator() (OriginValidator, error) {
	tregex, err := regexp.Compile(`^https://([[:alnum:]\-_]+\.)*trezor\.io$`)
	if err != nil {
		return nil, err
	}
	// `localhost:8xxx` and `5xxx` are added for easing local development.
	lregex, err := regexp.Compile(`^https?://localhost:[58][[:digit:]]{3}$`)
	if err != nil {
		return nil, err
	}
	v := func(origin string) bool {
		if lregex.MatchString(origin) {
			return true
		}

		// `null` is for electron apps or chrome extensions.
		// commented out for now
		// if origin == "null" {
		//	return true
		// }

		if tregex.MatchString(origin) {
			return true
		}

		return false
	}

	return v, nil
}

func (s *Server) Run() error {
	return s.https.ListenAndServe()
}

func (s *Server) Close() error {
	return s.https.Close()
}

func makeStatusTemplateDevice(dev entry) statusTemplateDevice {
	var devType statusTemplateDevType
	if dev.Vendor == usb.VendorT1 {
		devType = typeT1
	}
	if dev.Vendor == usb.VendorT2 {
		if dev.Product == usb.ProductT2Firmware {
			devType = typeT2
		} else {
			devType = typeT2Boot
		}
	}
	var session string
	if dev.Session != nil {
		session = *dev.Session
	}
	tdev := statusTemplateDevice{
		Path:    dev.Path,
		Type:    devType,
		Used:    dev.Session != nil,
		Session: session,
	}
	return tdev
}

func (s *Server) StatusPage(w http.ResponseWriter, r *http.Request) {
	s.dlogger.Println("http - building status page")

	var templateErr error

	e, err := s.enumerate()
	if err != nil {
		s.dlogger.Printf("http - status - enumerate err %s", err.Error())
		templateErr = err
	}

	tdevs := make([]statusTemplateDevice, 0)

	for _, dev := range e {
		tdevs = append(tdevs, makeStatusTemplateDevice(dev))
	}

	s.dlogger.Println("http - asking devcon")

	devconLog, err := devconInfo(s.dlogger)
	if err != nil {
		s.dlogger.Printf("http - status - devcon err %s", err.Error())
		templateErr = err
	}

	start := version + "\n" + devconLog

	log, err := s.mw.String(start)
	if err != nil {
		s.respondError(w, err)
		return
	}

	gziplog, err := s.dmw.GzipJsArray(start)

	if err != nil {
		s.respondError(w, err)
		return
	}

	s.dlogger.Println("http - actually building status data")

	isErr := templateErr != nil
	strErr := ""
	if templateErr != nil {
		strErr = templateErr.Error()
	}

	data := &statusTemplateData{
		Version:        version,
		Devices:        tdevs,
		DeviceCount:    len(tdevs),
		Log:            log,
		DLogGzipJSData: gziplog,
		IsError:        isErr,
		Error:          strErr,
	}

	err = statusTemplate.Execute(w, data)
	s.checkJSONError(w, err)
}

func (s *Server) Info(w http.ResponseWriter, r *http.Request) {
	s.dlogger.Printf("http - version %s", version)
	type info struct {
		Version string `json:"version"`
	}
	err := json.NewEncoder(w).Encode(info{
		Version: version,
	})
	s.checkJSONError(w, err)
}

type entry struct {
	Path    string  `json:"path"`
	Vendor  int     `json:"vendor"`
	Product int     `json:"product"`
	Session *string `json:"session"`
}

func sortEntries(entries []entry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
}

func (s *Server) Listen(w http.ResponseWriter, r *http.Request) {
	s.dlogger.Println("http - listen starting")
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}
	cnn := cn.CloseNotify()

	const (
		iterMax   = 600
		iterDelay = 500 // ms
	)
	var entries []entry

	s.dlogger.Println("http - listen decoding entries")

	err := json.NewDecoder(r.Body).Decode(&entries)
	defer func() {
		errClose := r.Body.Close()
		if errClose != nil {
			// just log
			s.logger.Printf("Error on request close: %s", errClose.Error())
		}
	}()

	if err != nil {
		s.respondError(w, err)
		return
	}

	sortEntries(entries)

	for i := 0; i < iterMax; i++ {
		s.dlogger.Println("http - listen before enumerating")
		e, enumErr := s.enumerate()
		if enumErr != nil {
			s.respondError(w, enumErr)
			return
		}
		if reflect.DeepEqual(entries, e) {
			s.dlogger.Println("http - listen equal, waiting")
			select {
			case <-cnn:
				s.dlogger.Println("http - listen request closed")
				return
			default:
				time.Sleep(iterDelay * time.Millisecond)
			}
		} else {
			s.dlogger.Println("http - listen different")
			entries = e
			break
		}
	}
	s.dlogger.Println("http - listen encoding and exiting")
	err = json.NewEncoder(w).Encode(entries)
	s.checkJSONError(w, err)
}

func (s *Server) Enumerate(w http.ResponseWriter, r *http.Request) {
	s.dlogger.Println("http - Enumerate start")
	e, err := s.enumerate()
	if err != nil {
		s.respondError(w, err)
		return
	}
	s.dlogger.Println("http - Enumerate encoding and exiting")
	err = json.NewEncoder(w).Encode(e)
	s.checkJSONError(w, err)
}

func (s *Server) enumerate() ([]entry, error) {
	// Lock for atomic access to s.sessions.
	s.dlogger.Println("http - enumerate locking sessionsMutex")
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	s.dlogger.Println("http - enumerate locking callMutex")
	// Lock for atomic access to s.callInProgress.  It needs to be over
	// whole function, so that call does not actually start while
	// enumerating.
	s.callMutex.Lock()
	defer s.callMutex.Unlock()

	// Use saved info if call is in progress, otherwise enumerate.
	infos := s.lastInfos

	s.dlogger.Printf("http - enumerate callInProgress %t", s.callInProgress)
	if !s.callInProgress {
		s.dlogger.Println("http - enumerate bus")
		busInfos, err := s.bus.Enumerate()
		if err != nil {
			return nil, err
		}
		infos = busInfos
		s.lastInfos = infos
	}

	entries := s.createEnumerateEntries(infos)
	s.dlogger.Println("http - enumerate release disconnected")
	s.releaseDisconnected(infos)
	return entries, nil
}

func (s *Server) createEnumerateEntries(infos []usb.Info) []entry {
	entries := make([]entry, 0, len(infos))
	for _, info := range infos {
		e := entry{
			Path:    info.Path,
			Vendor:  info.VendorID,
			Product: info.ProductID,
		}
		for _, ss := range s.sessions {
			if ss.path == info.Path {
				// Copying to prevent overwriting on Acquire and
				// wrong comparison in Listen.
				ssidCopy := ss.id
				e.Session = &ssidCopy
			}
		}
		entries = append(entries, e)
	}
	sortEntries(entries)
	return entries
}

func (s *Server) releaseDisconnected(infos []usb.Info) {
	for ssid, ss := range s.sessions {
		connected := false
		for _, info := range infos {
			if ss.path == info.Path {
				connected = true
			}
		}
		if !connected {
			s.dlogger.Printf("http - releasing disconnected device %s", ssid)
			err := s.release(ssid)
			// just log if there is an error
			// they are disconnected anyway
			if err != nil {
				s.logger.Printf("Error on releasing disconnected device: %s", err)
			}
		}
	}
}

var (
	ErrWrongPrevSession = errors.New("wrong previous session")
	ErrSessionNotFound  = errors.New("session not found")
	ErrMalformedData    = errors.New("malformed data")
)

func (s *Server) Acquire(w http.ResponseWriter, r *http.Request) {
	s.dlogger.Println("http - acquire - locking sessionsMutex")
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	vars := mux.Vars(r)
	path := vars["path"]
	prev := vars["session"]
	if prev == "null" {
		prev = ""
	}

	s.dlogger.Printf("http - acquire - input path %s prev %s", path, prev)

	var acquired *session
	for _, ss := range s.sessions {
		if ss.path == path {
			acquired = ss
			break
		}
	}

	if acquired == nil {
		acquired = &session{path: path, call: 0}
	}

	s.dlogger.Printf("http - acquire - actually previous %s", acquired.id)

	if acquired.id != prev {
		s.respondError(w, ErrWrongPrevSession)
		return
	}

	if prev != "" {
		s.dlogger.Printf("http - acquire - releasing previous")
		err := s.release(prev)
		if err != nil {
			s.respondError(w, err)
			return
		}
	}

	s.dlogger.Println("http - acquire - trying to connect")
	dev, err := s.tryConnect(path)
	if err != nil {
		s.respondError(w, err)
		return
	}

	acquired.dev = dev
	acquired.id = s.newSession()

	s.dlogger.Printf("http - acquire - new session is %s", acquired.id)

	s.sessions[acquired.id] = acquired

	type result struct {
		Session string `json:"session"`
	}

	err = json.NewEncoder(w).Encode(result{
		Session: acquired.id,
	})
	s.checkJSONError(w, err)
}

// Chrome tries to read from trezor immediately after connecting,
// ans so do we.  Bad timing can produce error on s.bus.Connect.
// Try 3 times with a 100ms delay.
func (s *Server) tryConnect(path string) (usb.Device, error) {
	tries := 0
	for {
		s.dlogger.Printf("http - tryConnect - try number %d", tries)
		dev, err := s.bus.Connect(path)
		if err != nil {
			if tries < 3 {
				s.dlogger.Println("http - tryConnect - sleeping")
				tries++
				time.Sleep(100 * time.Millisecond)
			} else {
				s.dlogger.Println("http - tryConnect - too many times, exiting")
				return nil, err
			}
		} else {
			return dev, nil
		}
	}
}

func (s *Server) release(session string) error {
	s.dlogger.Printf("http - inner release - session %s", session)
	acquired := s.sessions[session]
	if acquired == nil {
		s.dlogger.Println("http - inner release - session not found")
		return ErrSessionNotFound
	}
	delete(s.sessions, session)

	s.dlogger.Println("http - inner release - bus close")
	err := acquired.dev.Close()
	return err
}

func (s *Server) Release(w http.ResponseWriter, r *http.Request) {
	s.dlogger.Println("http - release - locking sessionsMutex")
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	vars := mux.Vars(r)
	session := vars["session"]

	err := s.release(session)

	if err != nil {
		s.respondError(w, err)
		return
	}

	s.dlogger.Println("http - release - done, encoding")
	err = json.NewEncoder(w).Encode(vars)
	s.checkJSONError(w, err)
}

func (s *Server) Call(w http.ResponseWriter, r *http.Request) {
	s.call(w, r, false)
}

func (s *Server) Post(w http.ResponseWriter, r *http.Request) {
	s.call(w, r, true)
}

func (s *Server) call(w http.ResponseWriter, r *http.Request, skipRead bool) {
	s.dlogger.Println("http - call - start")
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}
	cnn := cn.CloseNotify()

	s.dlogger.Println("http - call - callMutex lock")
	s.callMutex.Lock()

	s.dlogger.Println("http - call - callMutex set callInProgress true, unlock")
	s.callInProgress = true

	s.callMutex.Unlock()
	s.dlogger.Println("http - call - callMutex unlock done")

	defer func() {
		s.dlogger.Println("http - call - callMutex closing lock")
		s.callMutex.Lock()

		s.dlogger.Println("http - call - callMutex set callInProgress false, unlock")
		s.callInProgress = false

		s.callMutex.Unlock()
		s.dlogger.Println("http - call - callMutex closing unlock")
	}()

	vars := mux.Vars(r)
	session := vars["session"]
	s.dlogger.Printf("http - call - session is %s", session)

	s.dlogger.Println("http - call - sessionsMutex lock")
	s.sessionsMutex.Lock()
	acquired := s.sessions[session]

	s.sessionsMutex.Unlock()
	s.dlogger.Println("http - call - sessionsMutex unlock done")

	if acquired == nil {
		s.respondError(w, ErrSessionNotFound)
		return
	}

	s.dlogger.Println("http - call - checking other call on same session")
	freeToCall := atomic.CompareAndSwapInt32(&acquired.call, 0, 1)
	if !freeToCall {
		http.Error(w, "other call in progress", http.StatusInternalServerError)
		return
	}

	s.dlogger.Println("http - call - checking other call on same session done")
	defer func() {
		atomic.StoreInt32(&acquired.call, 0)
	}()

	finished := make(chan bool, 1)
	defer func() {
		finished <- true
	}()

	go func() {
		select {
		case <-finished:
			return
		case <-cnn:
			s.dlogger.Println("http - call - detected request close, auto-release")
			errRelease := s.release(session)
			if errRelease != nil {
				// just log, since request is already closed
				s.logger.Printf("Error while releasing: %s", errRelease.Error())
			}
		}
	}()

	s.dlogger.Println("http - call - before actual logic")
	err := s.readWriteDev(w, r, acquired.dev, skipRead)
	s.dlogger.Println("http - call - after actual logic")

	if err != nil {
		s.respondError(w, err)
	}
}

func (s *Server) readWriteDev(w io.Writer, r *http.Request, d io.ReadWriter, skipRead bool) error {
	s.dlogger.Println("http - readWrite - decodeRaw")
	msg, err := s.decodeRaw(r.Body)
	if err != nil {
		return err
	}

	s.dlogger.Println("http - readWrite - writeTo")
	_, err = msg.WriteTo(d)
	if err != nil {
		return err
	}
	if skipRead {
		s.dlogger.Println("http - readWrite - skipping read")
		_, err = w.Write([]byte{0})
		return err
	}

	s.dlogger.Println("http - readWrite - readFrom")
	_, err = msg.ReadFrom(d)
	if err != nil {
		return err
	}

	s.dlogger.Println("http - readWrite - encoding back")
	err = s.encodeRaw(w, msg)
	return err
}

var latestSessionID = 0

func (s *Server) newSession() string {
	latestSessionID++
	return strconv.Itoa(latestSessionID)
}

func (s *Server) decodeRaw(r io.Reader) (*wire.Message, error) {
	s.dlogger.Println("http - decode - readAll")

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	s.dlogger.Println("http - decode - decodeString")
	body, err = hex.DecodeString(string(body))
	if err != nil {
		return nil, err
	}
	if len(body) < 6 {
		s.dlogger.Println("http - decode - body too short")
		return nil, ErrMalformedData
	}

	kind := binary.BigEndian.Uint16(body[0:2])
	size := binary.BigEndian.Uint32(body[2:6])
	data := body[6:]
	if uint32(len(data)) != size {
		s.dlogger.Println("http - decode - wrong data length")
		return nil, ErrMalformedData
	}

	if wire.Validate(data) != nil {
		s.dlogger.Println("http - decode - invalid data")
		return nil, ErrMalformedData
	}

	s.dlogger.Println("http - decode - returning")
	return &wire.Message{
		Kind: kind,
		Data: data,

		Dlogger: s.dlogger,
	}, nil
}

func (s *Server) encodeRaw(w io.Writer, msg *wire.Message) error {
	s.dlogger.Println("http - encode - start")
	var (
		header [6]byte
		data   = msg.Data
		kind   = msg.Kind
		size   = uint32(len(msg.Data))
	)
	binary.BigEndian.PutUint16(header[0:2], kind)
	binary.BigEndian.PutUint32(header[2:6], size)

	st := hex.EncodeToString(header[:])

	s.dlogger.Println("http - encode - writing header")
	_, err := w.Write([]byte(st))
	if err != nil {
		return err
	}

	s.dlogger.Println("http - encode - writing data")
	st = hex.EncodeToString(data)
	_, err = w.Write([]byte(st))
	return err
}

func (s *Server) checkJSONError(w http.ResponseWriter, err error) {
	if err != nil {
		s.respondError(w, err)
	}
}

func (s *Server) respondError(w http.ResponseWriter, err error) {
	type jsonError struct {
		Error string `json:"error"`
	}
	log.Printf("Returning error: %s", err.Error())
	w.WriteHeader(http.StatusBadRequest)
	// if even the encoder of the error errors, just log the error
	err = json.NewEncoder(w).Encode(jsonError{
		Error: err.Error(),
	})
	if err != nil {
		s.logger.Printf("Error while writing error: %s", err.Error())
	}
}
