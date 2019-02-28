package server

import (
	"fmt"
	"io"
	"net/http"

	coreapi "github.com/trezor/trezord-go/api"

	"github.com/trezor/trezord-go/internal/memorywriter"
	"github.com/trezor/trezord-go/internal/server/api"
	"github.com/trezor/trezord-go/internal/server/status"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type serverPrivate struct {
	*http.Server
}

type Server struct {
	serverPrivate

	writer io.Writer
}

func New(
	a *coreapi.API,
	stderrWriter io.Writer,
	shortWriter *memorywriter.MemoryWriter,
	longWriter *memorywriter.MemoryWriter,
	version string,
) (*Server, error) {
	longWriter.Log("starting")

	https := &http.Server{
		Addr: "127.0.0.1:21325",
	}

	allWriter := io.MultiWriter(stderrWriter, shortWriter, longWriter)
	s := &Server{
		serverPrivate: serverPrivate{
			Server: https,
		},
		writer: allWriter,
	}

	r := mux.NewRouter()
	statusRouter := r.PathPrefix("/status").Subrouter()
	postRouter := r.Methods("POST").Subrouter()
	redirectRouter := r.Methods("GET").Path("/").Subrouter()

	status.ServeStatus(statusRouter, a, version, shortWriter, longWriter)
	api.ServeAPI(postRouter, a, version, longWriter)

	status.ServeStatusRedirect(redirectRouter)

	var h http.Handler = r

	// Log after the request is done, in the Apache format.
	h = handlers.LoggingHandler(allWriter, h)
	// Log when the request is received.
	h = s.logRequest(h)

	https.Handler = h

	longWriter.Log("server created")
	return s, nil
}

func (s *Server) logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := fmt.Sprintf("%s %s\n", r.Method, r.URL)
		_, err := s.writer.Write([]byte(text))
		if err != nil {
			// give up, just print on stdout
			fmt.Println(err)
		}
		handler.ServeHTTP(w, r)
	})
}

func (s *Server) Run() error {
	return s.ListenAndServe()
}
