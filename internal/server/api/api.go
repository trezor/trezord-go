package api

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/trezor/trezord-go/internal/logs"
	"github.com/trezor/trezord-go/internal/message"
	"github.com/trezor/trezord-go/trezorapi"
	types "github.com/trezor/trezord-go/trezorapi/trezortypes"

	"github.com/gorilla/mux"
)

// This package is for serving the actual trezord API.
// The actual logic of enumeration is in core package,
// in this package, we deal with converting the data from the request
// and then again formatting to the reply

type api struct {
	core    *trezorapi.API
	version string
	logger  *logs.Logger
}

func ServeAPI(r *mux.Router, a *trezorapi.API, v string, l *logs.Logger) {
	api := &api{
		core:    a,
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
	r.HandleFunc("/read/{session}", api.Read)
	r.HandleFunc("/debug/acquire/{path}", api.AcquireDebug)
	r.HandleFunc("/debug/acquire/{path}/{session}", api.AcquireDebug)
	r.HandleFunc("/debug/release/{session}", api.ReleaseDebug)
	r.HandleFunc("/debug/call/{session}", api.CallDebug)
	r.HandleFunc("/debug/post/{session}", api.PostDebug)
	r.HandleFunc("/debug/read/{session}", api.ReadDebug)
	corsv := corsValidator()
	r.Use(CORS(corsv))
}

func (a *api) Info(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("version " + a.version)

	err := json.NewEncoder(w).Encode(types.VersionInfo{
		Version: a.version,
	})
	a.checkJSONError(w, err)
}

func (a *api) Listen(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("starting")
	var entries []types.EnumerateEntry

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

	res, err := a.core.Listen(r.Context(), entries)
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
	res, err := a.core.Acquire(path, &prev, debug)

	if err != nil {
		a.respondError(w, err)
		return
	}

	err = json.NewEncoder(w).Encode(types.SessionInfo{
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

type callMode int

const (
	callModeRead      callMode = 0
	callModeWrite     callMode = 1
	callModeReadWrite callMode = 2
)

func (a *api) Call(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, callModeReadWrite, false)
}

func (a *api) Post(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, callModeWrite, false)
}

func (a *api) Read(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, callModeRead, false)
}

func (a *api) CallDebug(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, callModeReadWrite, true)
}

func (a *api) PostDebug(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, callModeWrite, true)
}

func (a *api) ReadDebug(w http.ResponseWriter, r *http.Request) {
	a.call(w, r, callModeRead, true)
}

func hexRead(mode callMode, r *http.Request, l io.Writer) (*types.Message, error) {
	if mode != callModeRead {
		hexbody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		binbody, err := hex.DecodeString(string(hexbody))
		if err != nil {
			return nil, err
		}

		inMsg, err := message.FromBridgeFormat(binbody, l)
		if err != nil {
			return nil, err
		}
		return inMsg, nil
	}
	return nil, nil
}

func hexWrite(mode callMode, outMsg *types.Message, httpWriter, log io.Writer) error {
	if mode != callModeWrite {
		binres, err := message.ToBridgeFormat(outMsg, log)

		if err != nil {
			return err
		}

		hexres := hex.EncodeToString(binres)
		_, err = httpWriter.Write([]byte(hexres))

		if err != nil {
			return err
		}
	}
	return nil
}

func (a *api) call(w http.ResponseWriter, r *http.Request, mode callMode, debug bool) {
	a.logger.Log("start")

	vars := mux.Vars(r)
	session := vars["session"]

	inMsg, err := hexRead(mode, r, a.logger)

	if err != nil {
		a.respondError(w, err)
		return
	}

	var outMsg *types.Message

	switch mode {
	case callModeRead:
		outMsg, err = a.core.Read(r.Context(), session, debug)
	case callModeWrite:
		err = a.core.Post(r.Context(), inMsg, session, debug)
	default:
		outMsg, err = a.core.Call(r.Context(), inMsg, session, debug)
	}

	if err != nil {
		a.respondError(w, err)
		return
	}

	err = hexWrite(mode, outMsg, w, a.logger)

	if err != nil {
		a.respondError(w, err)
		return
	}
}

func corsValidator() OriginValidator {
	tregex := regexp.MustCompile(`^https://([[:alnum:]\-_]+\.)*trezor\.io$`)
	// `localhost:8xxx` and `5xxx` are added for easing local development.
	lregex := regexp.MustCompile(`^https?://localhost:[58][[:digit:]]{3}$`)
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
