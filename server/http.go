package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/server/api"
	"github.com/trezor/trezord-go/server/status"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const version = "2.0.13"

type Server struct {
	https *http.Server

	writer io.Writer
}

func New(
	bus core.USBBus,
	stderrWriter io.Writer,
	shortWriter *memorywriter.MemoryWriter,
	longWriter *memorywriter.MemoryWriter,
) (*Server, error) {

	c := core.New(bus, longWriter)

	longWriter.Println("http - starting")

	https := &http.Server{
		Addr: "127.0.0.1:21325",
	}

	allWriter := io.MultiWriter(stderrWriter, shortWriter, longWriter)
	s := &Server{
		https:  https,
		writer: allWriter,
	}

	r := mux.NewRouter()
	statusRouter := r.PathPrefix("/status").Subrouter()
	postRouter := r.Methods("POST").Subrouter()
	redirectRouter := r.Methods("GET").Path("/").Subrouter()

	status.ServeStatus(statusRouter, c, version, shortWriter, longWriter)
	err := api.ServeAPI(postRouter, c, version, longWriter)
	if err != nil {
		panic(err) // only error is an error from originValidator regexp constructor
	}

	status.ServeStatusRedirect(redirectRouter)

	var h http.Handler = r

	// Log after the request is done, in the Apache format.
	h = handlers.LoggingHandler(allWriter, h)
	// Log when the request is received.
	h = s.logRequest(h)

	https.Handler = h

	longWriter.Println("http - server created")
	return s, nil
}

func (s *Server) logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := fmt.Sprintf("%s %s\n", r.Method, r.URL)
		s.writer.Write([]byte(text)) // nolint: errcheck, gas
		handler.ServeHTTP(w, r)
	})
}

func (s *Server) Run() error {
	return s.https.ListenAndServe()
}

func (s *Server) Close() error {
	return s.https.Close()
}
