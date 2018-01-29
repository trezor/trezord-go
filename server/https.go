package server

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/jpochyla/trezord-go/usb"
	"github.com/jpochyla/trezord-go/wire"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type session struct {
	path string
	id   string
	dev  usb.Device
}

type server struct {
	https    *http.Server
	bus      *usb.USB
	sessions map[string]*session
}

func New(bus *usb.USB) (*server, error) {
	certs, err := downloadCerts()
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: certs,
	}
	https := &http.Server{
		Addr:      "127.0.0.1:21324",
		TLSConfig: config,
	}
	s := &server{
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
	headers := handlers.AllowedHeaders([]string{"Content-Type"})
	methods := handlers.AllowedMethods([]string{"HEAD", "POST", "OPTIONS"})

	var h http.Handler = r
	// restrict cross-origin access
	h = handlers.CORS(headers, v, methods)(h)
	// log after the request is done, in the Apache format
	h = handlers.LoggingHandler(os.Stdout, h)
	// log when the request is received
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

func corsValidator() (handlers.CORSOption, error) {
	tregex, err := regexp.Compile(`^https://([[:alnum:]]+\.)*trezor\.io$`)
	if err != nil {
		return nil, err
	}
	v := handlers.AllowedOriginValidator(func(origin string) bool {
		// `localhost:8000` is added for easing local development
		if origin == "https://localhost:8000" {
			return true
		}

		// `null` is for electron apps or chrome extensions
		if origin == "null" {
			return true
		}

		if tregex.MatchString(origin) {
			return true
		}

		return false
	})

	return v, nil
}

func (s *server) Run() error {
	return s.https.ListenAndServeTLS("", "")
}

func (s *server) Close() error {
	return s.https.Close()
}

func (s *server) Info(w http.ResponseWriter, r *http.Request) {
	type info struct {
		Version string `json:"version"`
	}
	json.NewEncoder(w).Encode(info{
		Version: "2.0.0",
	})
}

type entry struct {
	Path    string  `json:"path"`
	Vendor  int     `json:"vendor"`
	Product int     `json:"product"`
	Session *string `json:"session"`
}

func (s *server) Listen(w http.ResponseWriter, r *http.Request) {
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

func (s *server) Enumerate(w http.ResponseWriter, r *http.Request) {
	e, err := s.enumerate()
	if err != nil {
		respondError(w, err)
		return
	}
	json.NewEncoder(w).Encode(e)
}

func (s *server) enumerate() ([]entry, error) {
	infos, err := s.bus.Enumerate()
	if err != nil {
		return nil, err
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
				// copying to prevent overwriting on acquire
				// and wrong comparison in Listen
				ssIdCopy := ss.id
				e.Session = &ssIdCopy
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

var (
	ErrSessionNotFound = errors.New("wrong previous session")
	ErrMalformedData   = errors.New("malformed data")
)

func (s *server) Acquire(w http.ResponseWriter, r *http.Request) {
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
		acquired = &session{path: path}
	}
	if acquired.id != prev {
		respondError(w, ErrSessionNotFound)
		return
	}

	if prev != "" {
		err := s.release(prev)
		if err != nil {
			respondError(w, err)
			return
		}
	}

	// Chrome tries to read from Trezor immediately after connecting
	// So do we
	// Bad timing can produce error on s.bus.Connect
	// => try 3 times with a 100ms delay
	tries := 0
	for {
		dev, err := s.bus.Connect(path)
		if err != nil {
			if tries < 3 {
				tries++
				time.Sleep(100 * time.Millisecond)
			} else {
				respondError(w, err)
				return
			}
		} else {
			acquired.dev = dev
			break
		}
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

func (s *server) release(session string) error {
	acquired, _ := s.sessions[session]
	if acquired == nil {
		return ErrSessionNotFound
	}
	delete(s.sessions, session)

	acquired.dev.Close()
	return nil
}

func (s *server) Release(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session := vars["session"]

	err := s.release(session)

	if err != nil {
		respondError(w, err)
		return
	}

	json.NewEncoder(w).Encode(vars)
}

func (s *server) Call(w http.ResponseWriter, r *http.Request) {
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	session := vars["session"]

	acquired, _ := s.sessions[session]
	if acquired == nil {
		respondError(w, ErrSessionNotFound)
		return
	}

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

var latestSessionId = 0

func (s *server) newSession() string {
	latestSessionId++
	return strconv.Itoa(latestSessionId)
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

const (
	certURI    = "https://wallet.trezor.io/data/bridge/cert/localback.crt"
	privkeyURI = "https://wallet.trezor.io/data/bridge/cert/localback.key"
)

func downloadCerts() ([]tls.Certificate, error) {
	cert, err := getURI(certURI)
	if err != nil {
		return nil, err
	}
	privkey, err := getURI(privkeyURI)
	if err != nil {
		return nil, err
	}
	crt, err := tls.X509KeyPair(cert, privkey)
	if err != nil {
		return nil, err
	}
	return []tls.Certificate{crt}, nil
}

func getURI(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
