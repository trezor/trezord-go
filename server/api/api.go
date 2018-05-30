package api

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"

	"github.com/gorilla/mux"
)

// This package is for serving the actual trezord API.
// The actual logic of enumeration is in core package,
// in this package, we deal with converting the data from the request
// and then again formatting to the reply

type api struct {
	core    *core.Core
	version string
	logger  *memorywriter.MemoryWriter
}

func ServeAPI(r *mux.Router, c *core.Core, v string, l *memorywriter.MemoryWriter) error {
	api := &api{
		core:    c,
		version: v,
		logger:  l,
	}
	r.HandleFunc("/", api.Info)
	r.HandleFunc("/configure", api.Info)
	r.HandleFunc("/listen", api.Listen)
	r.HandleFunc("/enumerate", api.Enumerate)
	r.HandleFunc("/acquire/{path}", api.Acquire)
	r.HandleFunc("/acquire/{path}/{session}", api.Acquire)
	r.HandleFunc("/release/{session}", api.Release)
	r.HandleFunc("/call/{session}", api.Call)
	r.HandleFunc("/post/{session}", api.Post)

	corsv, err := corsValidator()
	if err != nil {
		return err
	}
	r.Use(CORS(corsv))
	return nil
}

func (a *api) log(s string) {
	a.logger.Println("api - " + s)
}

func (a *api) Info(w http.ResponseWriter, r *http.Request) {
	a.log("version " + a.version)

	type info struct {
		Version string `json:"version"`
	}
	err := json.NewEncoder(w).Encode(info{
		Version: a.version,
	})
	a.checkJSONError(w, err)
}

func (a *api) Listen(w http.ResponseWriter, r *http.Request) {
	a.log("listen starting")
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}
	cnn := cn.CloseNotify()

	var entries []core.EnumerateEntry

	a.log("listen decoding entries")

	err := json.NewDecoder(r.Body).Decode(&entries)
	defer func() {
		errClose := r.Body.Close()
		if errClose != nil {
			// just log
			a.log("Error on request close: " + errClose.Error())
		}
	}()

	if err != nil {
		a.respondError(w, err)
		return
	}

	res, err := a.core.Listen(entries, cnn)
	if err != nil {
		a.respondError(w, err)
		return
	}

	err = json.NewEncoder(w).Encode(res)
	a.checkJSONError(w, err)
}

func (a *api) Enumerate(w http.ResponseWriter, r *http.Request) {
	a.log("Enumerate start")
	e, err := a.core.Enumerate()
	if err != nil {
		a.respondError(w, err)
		return
	}
	a.log("Enumerate encoding and exiting")
	err = json.NewEncoder(w).Encode(e)
	a.checkJSONError(w, err)
}

func (a *api) Acquire(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]
	prev := vars["session"]
	if prev == "null" {
		prev = ""
	}
	res, err := a.core.Acquire(path, prev)

	if err != nil {
		a.respondError(w, err)
		return
	}

	type result struct {
		Session string `json:"session"`
	}

	err = json.NewEncoder(w).Encode(result{
		Session: res,
	})
	a.checkJSONError(w, err)
}

func (a *api) Release(w http.ResponseWriter, r *http.Request) {
	a.log("release - locking sessionsMutex")

	vars := mux.Vars(r)
	session := vars["session"]

	err := a.core.Release(session)

	if err != nil {
		a.respondError(w, err)
		return
	}

	a.log("release - done, encoding")
	err = json.NewEncoder(w).Encode(vars)
	a.checkJSONError(w, err)
}

func (a *api) Call(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, false)
}

func (a *api) Post(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, true)
}

func (a *api) call(w http.ResponseWriter, r *http.Request, skipRead bool) {
	a.log("call - start")
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}
	cnn := cn.CloseNotify()

	vars := mux.Vars(r)
	session := vars["session"]

	hexbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondError(w, err)
		return
	}

	binbody, err := hex.DecodeString(string(hexbody))
	if err != nil {
		a.respondError(w, err)
		return
	}

	binres, err := a.core.Call(binbody, session, skipRead, cnn)
	if err != nil {
		a.respondError(w, err)
		return
	}

	hexres := hex.EncodeToString(binres)
	_, err = w.Write([]byte(hexres))

	if err != nil {
		a.respondError(w, err)
	}
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

func (a *api) checkJSONError(w http.ResponseWriter, err error) {
	if err != nil {
		a.respondError(w, err)
	}
}

func (a *api) respondError(w http.ResponseWriter, err error) {
	type jsonError struct {
		Error string `json:"error"`
	}
	a.log("Returning error: " + err.Error())
	w.WriteHeader(http.StatusBadRequest)

	// if even the encoder of the error errors, just log the error
	err = json.NewEncoder(w).Encode(jsonError{
		Error: err.Error(),
	})
	if err != nil {
		a.logger.Println("Error while writing error: " + err.Error())
	}
}
