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

const version = "2.0.12"

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

	mw *memorywriter.MemoryWriter
}

func New(bus *usb.USB, logger io.Writer, mw *memorywriter.MemoryWriter) (*Server, error) {
	https := &http.Server{
		Addr: "127.0.0.1:21325",
	}
	s := &Server{
		bus:      bus,
		https:    https,
		sessions: make(map[string]*session),

		mw: mw,
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

	v, err := corsValidator()
	if err != nil {
		return nil, err
	}

	var h http.Handler = r
	// Restrict cross-origin access.
	h = CORS(v)(h)
	// Log after the request is done, in the Apache format.
	h = handlers.LoggingHandler(logger, h)
	// Log when the request is received.
	h = logRequest(h)

	https.Handler = h

	return s, nil
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL)
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

func (s *Server) StatusPage(w http.ResponseWriter, r *http.Request) {
	e, err := s.enumerate()
	if err != nil {
		respondError(w, err)
		return
	}

	tdevs := make([]statusTemplateDevice, 0)

	for _, dev := range e {
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
		tdevs = append(tdevs, tdev)
	}

	origLog := s.mw.String()
	devconLog, err := devconInfo()
	if err != nil {
		respondError(w, err)
		return
	}
	log := devconLog + origLog

	data := &statusTemplateData{
		Version:     version,
		Devices:     tdevs,
		DeviceCount: len(tdevs),
		Log:         log,
	}

	err = statusTemplate.Execute(w, data)
	checkJSONError(w, err)
}

func (s *Server) Info(w http.ResponseWriter, r *http.Request) {
	type info struct {
		Version string `json:"version"`
	}
	err := json.NewEncoder(w).Encode(info{
		Version: version,
	})
	checkJSONError(w, err)
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

	err := json.NewDecoder(r.Body).Decode(&entries)
	defer func() {
		errClose := r.Body.Close()
		if errClose != nil {
			// just log
			log.Printf("Error on request close: %s", errClose.Error())
		}
	}()

	if err != nil {
		respondError(w, err)
		return
	}

	sortEntries(entries)

	for i := 0; i < iterMax; i++ {
		e, enumErr := s.enumerate()
		if enumErr != nil {
			respondError(w, enumErr)
			return
		}
		if reflect.DeepEqual(entries, e) {
			select {
			case <-cnn:
				return
			default:
				time.Sleep(iterDelay * time.Millisecond)
			}
		} else {
			entries = e
			break
		}
	}
	err = json.NewEncoder(w).Encode(entries)
	checkJSONError(w, err)
}

func (s *Server) Enumerate(w http.ResponseWriter, r *http.Request) {
	e, err := s.enumerate()
	if err != nil {
		respondError(w, err)
		return
	}
	err = json.NewEncoder(w).Encode(e)
	checkJSONError(w, err)
}

func (s *Server) enumerate() ([]entry, error) {
	// Lock for atomic access to s.sessions.
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	// Lock for atomic access to s.callInProgress.  It needs to be over
	// whole function, so that call does not actually start while
	// enumerating.
	s.callMutex.Lock()
	defer s.callMutex.Unlock()

	// Use saved info if call is in progress, otherwise enumerate.
	infos := s.lastInfos

	if !s.callInProgress {
		busInfos, err := s.bus.Enumerate()
		if err != nil {
			return nil, err
		}
		infos = busInfos
		s.lastInfos = infos
	}

	entries := s.createEnumerateEntries(infos)
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
			err := s.release(ssid)
			// just log if there is an error
			// they are disconnected anyway
			if err != nil {
				log.Printf("Error on releasing disconnected device: %s", err)
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
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	vars := mux.Vars(r)
	path := vars["path"]
	prev := vars["session"]
	if prev == "null" {
		prev = ""
	}

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
	if acquired.id != prev {
		respondError(w, ErrWrongPrevSession)
		return
	}

	if prev != "" {
		err := s.release(prev)
		if err != nil {
			respondError(w, err)
			return
		}
	}

	dev, err := s.tryConnect(path)
	if err != nil {
		respondError(w, err)
		return
	}

	acquired.dev = dev
	acquired.id = s.newSession()

	s.sessions[acquired.id] = acquired

	type result struct {
		Session string `json:"session"`
	}

	err = json.NewEncoder(w).Encode(result{
		Session: acquired.id,
	})
	checkJSONError(w, err)
}

// Chrome tries to read from trezor immediately after connecting,
// ans so do we.  Bad timing can produce error on s.bus.Connect.
// Try 3 times with a 100ms delay.
func (s *Server) tryConnect(path string) (usb.Device, error) {
	tries := 0
	for {
		dev, err := s.bus.Connect(path)
		if err != nil {
			if tries < 3 {
				tries++
				time.Sleep(100 * time.Millisecond)
			} else {
				return nil, err
			}
		} else {
			return dev, nil
		}
	}
}

func (s *Server) release(session string) error {
	acquired := s.sessions[session]
	if acquired == nil {
		return ErrSessionNotFound
	}
	delete(s.sessions, session)

	err := acquired.dev.Close()
	return err
}

func (s *Server) Release(w http.ResponseWriter, r *http.Request) {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	vars := mux.Vars(r)
	session := vars["session"]

	err := s.release(session)

	if err != nil {
		respondError(w, err)
		return
	}

	err = json.NewEncoder(w).Encode(vars)
	checkJSONError(w, err)
}

func (s *Server) Call(w http.ResponseWriter, r *http.Request) {
	s.call(w, r, false)
}

func (s *Server) Post(w http.ResponseWriter, r *http.Request) {
	s.call(w, r, true)
}

func (s *Server) call(w http.ResponseWriter, r *http.Request, skipRead bool) {
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}
	cnn := cn.CloseNotify()

	s.callMutex.Lock()
	s.callInProgress = true
	s.callMutex.Unlock()

	defer func() {
		s.callMutex.Lock()
		s.callInProgress = false
		s.callMutex.Unlock()
	}()

	vars := mux.Vars(r)
	session := vars["session"]

	s.sessionsMutex.Lock()
	acquired := s.sessions[session]
	s.sessionsMutex.Unlock()

	if acquired == nil {
		respondError(w, ErrSessionNotFound)
		return
	}

	freeToCall := atomic.CompareAndSwapInt32(&acquired.call, 0, 1)
	if !freeToCall {
		http.Error(w, "other call in progress", http.StatusInternalServerError)
		return
	}
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
			errRelease := s.release(session)
			if errRelease != nil {
				// just log, since request is already closed
				log.Printf("Error while releasing: %s", errRelease.Error())
			}
		}
	}()

	err := readWriteDev(w, r, acquired.dev, skipRead)
	if err != nil {
		respondError(w, err)
	}
}

func readWriteDev(w io.Writer, r *http.Request, d io.ReadWriter, skipRead bool) error {
	msg, err := decodeRaw(r.Body)
	if err != nil {
		return err
	}
	_, err = msg.WriteTo(d)
	if err != nil {
		return err
	}
	if skipRead {
		_, err = w.Write([]byte{0})
		return err
	}
	_, err = msg.ReadFrom(d)
	if err != nil {
		return err
	}
	err = encodeRaw(w, msg)
	return err
}

var latestSessionID = 0

func (s *Server) newSession() string {
	latestSessionID++
	return strconv.Itoa(latestSessionID)
}

func decodeRaw(r io.Reader) (*wire.Message, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	body, err = hex.DecodeString(string(body))
	if err != nil {
		return nil, err
	}
	if len(body) < 6 {
		return nil, ErrMalformedData
	}
	kind := binary.BigEndian.Uint16(body[0:2])
	size := binary.BigEndian.Uint32(body[2:6])
	data := body[6:]
	if uint32(len(data)) != size {
		return nil, ErrMalformedData
	}

	if wire.Validate(data) != nil {
		return nil, ErrMalformedData
	}

	return &wire.Message{
		Kind: kind,
		Data: data,
	}, nil
}

func encodeRaw(w io.Writer, msg *wire.Message) error {
	var (
		header [6]byte
		data   = msg.Data
		kind   = msg.Kind
		size   = uint32(len(msg.Data))
	)
	binary.BigEndian.PutUint16(header[0:2], kind)
	binary.BigEndian.PutUint32(header[2:6], size)

	s := hex.EncodeToString(header[:])
	_, err := w.Write([]byte(s))
	if err != nil {
		return err
	}
	s = hex.EncodeToString(data)
	_, err = w.Write([]byte(s))
	return err
}

func checkJSONError(w http.ResponseWriter, err error) {
	if err != nil {
		respondError(w, err)
	}
}

func respondError(w http.ResponseWriter, err error) {
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
		log.Printf("Error while writing error: %s", err.Error())
	}
}
