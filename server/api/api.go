package api

import (
	"encoding/hex"
	"encoding/json"
	"io"
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
	githash string
	logger  *memorywriter.MemoryWriter
}

func ServeAPI(r *mux.Router, c *core.Core, v, h string, l *memorywriter.MemoryWriter) {
	api := &api{
		core:    c,
		version: v,
		githash: h,
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
	r.HandleFunc("/read/{session}", api.Read)
	r.HandleFunc("/debug/acquire/{path}", api.AcquireDebug)
	r.HandleFunc("/debug/acquire/{path}/{session}", api.AcquireDebug)
	r.HandleFunc("/debug/release/{session}", api.ReleaseDebug)
	r.HandleFunc("/debug/call/{session}", api.CallDebug)
	r.HandleFunc("/debug/post/{session}", api.PostDebug)
	r.HandleFunc("/debug/read/{session}", api.ReadDebug)
	if !core.IsDebugBinary() {
		corsv := corsValidator()
		r.Use(CORS(corsv))
	}
}

func (a *api) Info(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("version " + a.version + " (rev " + a.githash + ")")

	type info struct {
		Version string `json:"version"`
		Githash string `json:"githash"`
	}
	err := json.NewEncoder(w).Encode(info{
		Version: a.version,
		Githash: a.githash,
	})
	a.checkJSONError(w, err)
}

func (a *api) Listen(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("starting")
	var entries []core.EnumerateEntry

	a.logger.Log("decoding entries")

	err := json.NewDecoder(r.Body).Decode(&entries)
	defer func() {
		errClose := r.Body.Close()
		if errClose != nil {
			// just log
			a.logger.Log("Error on request close: " + errClose.Error())
		}
	}()

	if err != nil {
		a.respondError(w, err)
		return
	}

	res, err := a.core.Listen(entries, r.Context())
	if err != nil {
		a.respondError(w, err)
		return
	}

	err = json.NewEncoder(w).Encode(res)
	a.checkJSONError(w, err)
}

func (a *api) Enumerate(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("start")
	e, err := a.core.Enumerate()
	if err != nil {
		a.respondError(w, err)
		return
	}
	a.logger.Log("encoding and exiting")
	err = json.NewEncoder(w).Encode(e)
	a.checkJSONError(w, err)
}

func (a *api) Acquire(w http.ResponseWriter, r *http.Request) {
	a.acquire(w, r, false)
}

func (a *api) AcquireDebug(w http.ResponseWriter, r *http.Request) {
	a.acquire(w, r, true)
}

func (a *api) acquire(w http.ResponseWriter, r *http.Request, debug bool) {
	vars := mux.Vars(r)
	path := vars["path"]
	prev := vars["session"]
	if prev == "null" {
		prev = ""
	}
	res, err := a.core.Acquire(path, prev, debug)

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
	a.release(w, r, false)
}

func (a *api) ReleaseDebug(w http.ResponseWriter, r *http.Request) {
	a.release(w, r, true)
}

func (a *api) release(w http.ResponseWriter, r *http.Request, debug bool) {
	a.logger.Log("start")

	vars := mux.Vars(r)
	session := vars["session"]

	err := a.core.Release(session, debug)

	if err != nil {
		a.respondError(w, err)
		return
	}

	a.logger.Log("done, encoding")
	err = json.NewEncoder(w).Encode(vars)
	a.checkJSONError(w, err)
}

func (a *api) Call(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, core.CallModeReadWrite, false)
}

func (a *api) Post(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, core.CallModeWrite, false)
}

func (a *api) Read(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, core.CallModeRead, false)
}

func (a *api) CallDebug(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, core.CallModeReadWrite, true)
}

func (a *api) PostDebug(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, core.CallModeWrite, true)
}

func (a *api) ReadDebug(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, core.CallModeRead, true)
}

func (a *api) call(w http.ResponseWriter, r *http.Request, mode core.CallMode, debug bool) {
	a.logger.Log("start")

	vars := mux.Vars(r)
	session := vars["session"]

	var binbody []byte
	if mode != core.CallModeRead {
		hexbody, err := io.ReadAll(r.Body)
		if err != nil {
			a.respondError(w, err)
			return
		}
		binbody, err = hex.DecodeString(string(hexbody))
		if err != nil {
			a.respondError(w, err)
			return
		}
	}

	binres, err := a.core.Call(binbody, session, mode, debug, r.Context())
	if err != nil {
		a.respondError(w, err)
		return
	}

	if mode != core.CallModeWrite {
		hexres := hex.EncodeToString(binres)
		_, err = w.Write([]byte(hexres))

		if err != nil {
			a.respondError(w, err)
		}
	}
}

func corsValidator() OriginValidator {
	// *.trezor.io
	trezorRegex := regexp.MustCompile(`^https://([[:alnum:]\-_]+\.)*trezor\.io$`)

	// *.trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion
	trezorOnionRegex := regexp.MustCompile(`^https?://([[:alnum:]\-_]+\.)*trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad\.onion$`)

	// `localhost:8xxx` and `5xxx` are added for easing local development
	localRegex := regexp.MustCompile(`^https?://localhost:[58][[:digit:]]{3}$`)

	// SatoshiLabs development servers
	develRegex := regexp.MustCompile(`^https://([[:alnum:]\-_]+\.)*sldev\.cz$`)

	v := func(origin string) bool {
		if trezorRegex.MatchString(origin) {
			return true
		}

		if trezorOnionRegex.MatchString(origin) {
			return true
		}

		if localRegex.MatchString(origin) {
			return true
		}

		if develRegex.MatchString(origin) {
			return true
		}

		return false
	}

	return v
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
	a.logger.Log("Returning error: " + err.Error())
	w.WriteHeader(http.StatusBadRequest)

	// if even the encoder of the error errors, just log the error
	err = json.NewEncoder(w).Encode(jsonError{
		Error: err.Error(),
	})
	if err != nil {
		a.logger.Log("Error while writing error: " + err.Error())
	}
}
