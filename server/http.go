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

	"github.com/trezor/trezord-go/usb"
	"github.com/trezor/trezord-go/wire"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

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
}

func New(bus *usb.USB, logger io.Writer) (*Server, error) {
	https := &http.Server{
		Addr: "127.0.0.1:21325",
	}
	s := &Server{
		bus:      bus,
		https:    https,
		sessions: make(map[string]*session),
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

func (s *Server) Info(w http.ResponseWriter, r *http.Request) {
	type info struct {
		Version string `json:"version"`
	}
	json.NewEncoder(w).Encode(info{
		Version: "2.0.10",
	})
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

	const (
		iterMax   = 600
		iterDelay = 500 // ms
	)
	var entries []entry

	err := json.NewDecoder(r.Body).Decode(&entries)
	defer r.Body.Close()

	if err != nil {
		respondError(w, err)
		return
	}

	sortEntries(entries)

	for i := 0; i < iterMax; i++ {
		e, err := s.enumerate()
		if err != nil {
			respondError(w, err)
			return
		}
		if reflect.DeepEqual(entries, e) {
			select {
			case <-cn.CloseNotify():
				return
			default:
				time.Sleep(iterDelay * time.Millisecond)
			}
		} else {
			entries = e
			break
		}
	}
	json.NewEncoder(w).Encode(entries)
}

func (s *Server) Enumerate(w http.ResponseWriter, r *http.Request) {
	e, err := s.enumerate()
	if err != nil {
		respondError(w, err)
		return
	}
	json.NewEncoder(w).Encode(e)
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
	// Also release all sessions of disconnected devices
	for ssid, ss := range s.sessions {
		connected := false
		for _, info := range infos {
			if ss.path == info.Path {
				connected = true
			}
		}
		if !connected {
			s.release(ssid)
		}
	}
	sortEntries(entries)
	return entries, nil
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
	} else {
		acquired.dev = dev
	}

	acquired.id = s.newSession()

	s.sessions[acquired.id] = acquired

	type result struct {
		Session string `json:"session"`
	}

	json.NewEncoder(w).Encode(result{
		Session: acquired.id,
	})
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
	acquired, _ := s.sessions[session]
	if acquired == nil {
		return ErrSessionNotFound
	}
	delete(s.sessions, session)

	acquired.dev.Close()
	return nil
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

	json.NewEncoder(w).Encode(vars)
}

func (s *Server) Call(w http.ResponseWriter, r *http.Request) {
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}

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
	acquired, _ := s.sessions[session]
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

	finished := make(chan bool)
	defer func() {
		finished <- true
	}()

	go func() {
		select {
		case <-finished:
			return
		case <-cn.CloseNotify():
			s.release(session)
		}
	}()

	msg, err := decodeRaw(r.Body)
	if err != nil {
		respondError(w, err)
		return
	}
	_, err = msg.WriteTo(acquired.dev)
	if err != nil {
		respondError(w, err)
		return
	}
	_, err = msg.ReadFrom(acquired.dev)
	if err != nil {
		respondError(w, err)
		return
	}
	err = encodeRaw(w, msg)
	if err != nil {
		respondError(w, err)
		return
	}
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

	var s string
	s = hex.EncodeToString(header[:])
	_, err := w.Write([]byte(s))
	if err != nil {
		return err
	}
	s = hex.EncodeToString(data)
	_, err = w.Write([]byte(s))
	if err != nil {
		return err
	}

	return nil
}

func respondError(w http.ResponseWriter, err error) {
	type jsonError struct {
		Error string `json:"error"`
	}
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(jsonError{
		Error: err.Error(),
	})
}
