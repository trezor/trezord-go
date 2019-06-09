package status

import (
	"net/http"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// This package serves the status page on /status/ and the
// log file at /status/log.gz with the detailed log

type status struct {
	core                                *core.Core
	version                             string
	shortMemoryWriter, longMemoryWriter *memorywriter.MemoryWriter
}

const csrfkey = "slk0118h51w2qiw4fhrfyd84f59j81ln"

func ServeStatusRedirect(r *mux.Router) {
	r.HandleFunc("/", redirect)
	r.Use(OriginCheck(map[string]string{
		"": "",
	}))
}

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://127.0.0.1:21325/status/", http.StatusMovedPermanently)
}

func ServeStatus(r *mux.Router, c *core.Core, v string, mw, dmw *memorywriter.MemoryWriter) {
	status := &status{
		core:              c,
		version:           v,
		shortMemoryWriter: mw,
		longMemoryWriter:  dmw,
	}
	r.Methods("GET").Path("/").HandlerFunc(status.statusPage)
	r.Methods("POST").Path("/log.gz").HandlerFunc(status.statusGzip)

	r.Use(csrf.Protect([]byte(csrfkey), csrf.Secure(false)))
	r.Use(OriginCheck(map[string]string{
		"/status/":       "",
		"/status/log.gz": "http://127.0.0.1:21325",
	}))
}

func (s *status) statusGzip(w http.ResponseWriter, r *http.Request) {
	s.longMemoryWriter.Log("building gzip")

	devconLog, err := devconInfo(s.longMemoryWriter)
	if err != nil {
		s.longMemoryWriter.Log("devcon err " + err.Error())
		respondError(w, err)
		return
	}

	devconLogD, err := devconAllStatusInfo()
	if err != nil {
		s.longMemoryWriter.Log("devcon err " + err.Error())
		respondError(w, err)
		return
	}

	msinfo, err := runMsinfo()
	if err != nil {
		s.longMemoryWriter.Log("msinfo err " + err.Error())
		respondError(w, err)
		return
	}

	s.longMemoryWriter.Log("getting libwdi")
	libwdi, err := libwdiReinstallLog()
	if err != nil {
		s.longMemoryWriter.Log("lbwdi err " + err.Error())
		respondError(w, err)
		return
	}

	s.longMemoryWriter.Log("getting old log")
	old, err := oldLog()
	if err != nil {
		s.longMemoryWriter.Log("old log err " + err.Error())
		respondError(w, err)
		return
	}

	s.longMemoryWriter.Log("getting setupapi")
	setupapi, err := setupAPIDevLog()
	if err != nil {
		s.longMemoryWriter.Log("setupapi err " + err.Error())
		respondError(w, err)
		return
	}

	start := s.version + "\n" +
		msinfo + "\n" +
		devconLog + devconLogD + "\n" +
		old +
		libwdi +
		setupapi +
		"\nCurrent log:\n"

	gzip, err := s.longMemoryWriter.Gzip(start)
	if err != nil {
		respondError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/gzip")

	_, err = w.Write(gzip)
	if err != nil {
		respondError(w, err)
		return
	}
}

func (s *status) statusPage(w http.ResponseWriter, r *http.Request) {
	s.longMemoryWriter.Log("building status page")

	var templateErr error
	tdevs, err := s.statusEnumerate()
	if err != nil {
		s.longMemoryWriter.Log("enumerate err" + err.Error())
		templateErr = err
	}

	devconLog, err := devconInfo(s.longMemoryWriter)
	if err != nil {
		s.longMemoryWriter.Log("devcon err " + err.Error())
		respondError(w, err)
		return
	}

	start := s.version + "\n" + devconLog

	log, err := s.shortMemoryWriter.String(start)
	if err != nil {
		respondError(w, err)
		return
	}

	s.longMemoryWriter.Log("actually building status data")

	isErr := templateErr != nil
	strErr := ""
	if templateErr != nil {
		strErr = templateErr.Error()
	}

	data := &statusTemplateData{
		Version:     s.version,
		Devices:     tdevs,
		DeviceCount: len(tdevs),
		Log:         log,
		IsError:     isErr,
		Error:       strErr,
		CSRFField:   csrf.TemplateField(r),
		IsWindows:   isWindows(),
	}

	err = statusTemplate.Execute(w, data)
	if err != nil {
		respondError(w, err)
		return
	}
}

func respondError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func (s *status) statusEnumerate() ([]statusTemplateDevice, error) {
	e, err := s.core.Enumerate()
	if err != nil {
		s.longMemoryWriter.Log("enumerate err" + err.Error())
		return nil, err
	}

	tdevs := make([]statusTemplateDevice, 0)

	for _, dev := range e {
		tdevs = append(tdevs, makeStatusTemplateDevice(dev))
	}
	return tdevs, nil
}

func makeStatusTemplateDevice(dev core.EnumerateEntry) statusTemplateDevice {
	var session string
	if dev.Session != nil {
		session = *dev.Session
	}
	tdev := statusTemplateDevice{
		Path:    dev.Path,
		Type:    dev.Type,
		Used:    dev.Session != nil,
		Session: session,
	}
	return tdev
}
