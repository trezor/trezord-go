package status

import (
	"net/http"
)

type originCheck struct {
	handler http.Handler
	allowed map[string]string
}

const (
	originHeader      string = "Origin"
	frameOriginHeader string = "X-Frame-Options"
)

func (o *originCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get(originHeader)
	path := r.URL.Path

	if o.allowed[path] != origin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set(frameOriginHeader, "DENY")
	o.handler.ServeHTTP(w, r)

	return
}

func OriginCheck(allowed map[string]string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		ch := &originCheck{
			allowed: allowed,
			handler: h,
		}
		return ch
	}
}
