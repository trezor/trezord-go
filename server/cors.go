package server

import (
	"net/http"
	"strings"
)

// Based on https://github.com/gorilla/handlers/blob/master/cors.go
// Copyright (c) 2013 The Gorilla Handlers Authors, BSD license

// OriginValidator takes an origin string and returns whether or not that origin is allowed.
type OriginValidator func(string) bool

type cors struct {
	h                      http.Handler
	allowedOriginValidator OriginValidator
}

var (
	allowedHeaders = []string{"Accept", "Accept-Language", "Content-Language", "Origin", "Content-Type"}
	allowedMethods = []string{"POST", "OPTIONS"}
)

const (
	corsOptionMethod         string = "OPTIONS"
	corsAllowOriginHeader    string = "Access-Control-Allow-Origin"
	corsRequestMethodHeader  string = "Access-Control-Request-Method"
	corsRequestHeadersHeader string = "Access-Control-Request-Headers"
	corsOriginHeader         string = "Origin"
	frameOriginHeader        string = "X-Frame-Options"
)

func (ch *cors) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get(corsOriginHeader)

	if r.URL.Path == "/" && r.Method == "GET" {
		ch.serveStatusRedirect(w, r, origin)
		return
	}

	if r.URL.Path == "/status/" && r.Method == "GET" {
		ch.serveStatus(w, r, origin)
		return
	}

	if r.URL.Path == "/status/log.gz" && r.Method == "POST" {
		ch.serveStatusLog(w, r, origin)
		return
	}

	if !ch.allowedOriginValidator(origin) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.Method == corsOptionMethod {

		if _, ok := r.Header[corsRequestMethodHeader]; !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		method := r.Header.Get(corsRequestMethodHeader)
		if !ch.isMatch(method, allowedMethods) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		requestHeaders := strings.Split(r.Header.Get(corsRequestHeadersHeader), ",")
		for _, v := range requestHeaders {
			canonicalHeader := http.CanonicalHeaderKey(strings.TrimSpace(v))
			if ch.isMatch(canonicalHeader, allowedHeaders) {
				continue
			}

			w.WriteHeader(http.StatusForbidden)
		}
	}

	w.Header().Set(corsAllowOriginHeader, origin)

	if r.Method == corsOptionMethod {
		return
	}
	ch.h.ServeHTTP(w, r)
}

func (ch *cors) serveStatusLog(w http.ResponseWriter, r *http.Request, origin string) {
	if origin != "http://127.0.0.1:21325" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set(frameOriginHeader, "DENY")
	ch.h.ServeHTTP(w, r)
}

func (ch *cors) serveStatus(w http.ResponseWriter, r *http.Request, origin string) {
	if origin != "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set(frameOriginHeader, "DENY")
	ch.h.ServeHTTP(w, r)
}

func (ch *cors) serveStatusRedirect(w http.ResponseWriter, r *http.Request, origin string) {
	if origin != "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set(frameOriginHeader, "DENY")
	http.Redirect(w, r, "http://127.0.0.1:21325/status/", http.StatusMovedPermanently)
}

func CORS(validator OriginValidator) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		ch := &cors{
			allowedOriginValidator: validator,
		}
		ch.h = h
		return ch
	}
}

func (ch *cors) isMatch(needle string, haystack []string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}

	return false
}
