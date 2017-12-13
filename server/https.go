package server

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"trezord-go/wire"

	"github.com/gorilla/mux"
)

type session struct {
	path string
	id   string
	dev  *wire.Device
}

type server struct {
	https    *http.Server
	bus      *wire.Bus
	sessions map[string]*session
}

func New(bus *wire.Bus) (*server, error) {
	certs, err := downloadCerts()
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: certs,
	}
	https := &http.Server{
		Addr:      ":21324",
		TLSConfig: config,
	}
	s := &server{
		bus:      bus,
		https:    https,
		sessions: make(map[string]*session),
	}
	r := mux.NewRouter()

	r.HandleFunc("/", s.Info)
	r.HandleFunc("/listen", s.Enumerate)
	r.HandleFunc("/enumerate", s.Enumerate)
	r.HandleFunc("/acquire/{path}", s.Acquire)
	r.HandleFunc("/acquire/{path}/{session}", s.Acquire)
	r.HandleFunc("/release/{session}", s.Release)
	r.HandleFunc("/call/{session}", s.Call)
	https.Handler = r

	return s, nil
}

func (s *server) Run() error {
	return s.https.ListenAndServeTLS("", "")
}

func (s *server) Close() error {
	return s.https.Close()
}

func (s *server) Info(w http.ResponseWriter, r *http.Request) {
	type info struct {
		Version    string `json:"version"`
		Configured bool   `json:"configured"`
		ValidUntil int    `json:"validUntil"`
	}
	json.NewEncoder(w).Encode(info{
		Version:    "2.0.0",
		Configured: true,
		ValidUntil: 0,
	})
}

func (s *server) Enumerate(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Path    string  `json:"path"`
		Vendor  int     `json:"vendor"`
		Product int     `json:"product"`
		Session *string `json:"session"`
	}

	paths, err := s.bus.Enumerate()
	if err != nil {
		respondError(w, err)
		return
	}

	var e entry
	var entries []entry
	for _, path := range paths {
		e.Path = path
		e.Vendor = int(s.bus.Vid)
		e.Product = int(s.bus.Pid)
		for _, ss := range s.sessions {
			if ss.path == path {
				e.Session = &ss.id
			}
		}
		entries = append(entries, e)
	}
	json.NewEncoder(w).Encode(entries)
}

var (
	ErrSessionNotFound = errors.New("session not found")
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

	dev, err := s.bus.Connect(path)
	if err != nil {
		respondError(w, err)
		return
	}
	acquired.dev = dev
	acquired.id = s.newSession()

	s.sessions[acquired.id] = acquired

	json.NewEncoder(w).Encode(acquired.id)
}

func (s *server) Release(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session := vars["session"]

	acquired, _ := s.sessions[session]
	if acquired == nil {
		respondError(w, ErrSessionNotFound)
		return
	}
	delete(s.sessions, session)

	acquired.dev.Close()

	json.NewEncoder(w).Encode(vars)
}

func (s *server) Call(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session := vars["session"]

	acquired, _ := s.sessions[session]
	if acquired == nil {
		respondError(w, ErrSessionNotFound)
		return
	}

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

func (s *server) newSession() string {
	return "random"
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

	_, err := w.Write(header[:])
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

func respondError(w http.ResponseWriter, err error) {
	type jsonError struct {
		Error string `json:"error"`
	}
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
