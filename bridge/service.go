// Package bridge provides Start/Stop lifecycle for the Trezor Communication Daemon.
package bridge

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/server"
	"github.com/trezor/trezord-go/usb"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Version is the bridge version string.
const Version = "2.0.34"

// Options configures bridge startup.
type Options struct {
	Port     int
	Logfile  string
	Verbose  bool
	WithUSB  bool
	Reset    bool
	UDPPorts []int
	UDPPairs []usb.PortTouple
	Version  string
	GitHash  string
	// Writer overrides stderr logging destination when set.
	Writer io.Writer
}

// Service is a running bridge instance.
type Service struct {
	mu     sync.Mutex
	opts   Options
	core   *core.Core
	bus    *usb.USB
	server *server.Server
	done   chan error
}

// ResolveGitHash returns the short VCS revision from build info, or "unknown".
func ResolveGitHash() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, v := range info.Settings {
		if v.Key == "vcs.revision" && len(v.Value) >= 7 {
			return v.Value[0:7]
		}
	}
	return "unknown"
}

// Start initializes USB/UDP transports and runs the HTTP server.
func Start(opts Options) (*Service, error) {
	if opts.Port == 0 {
		opts.Port = 21325
	}
	if opts.Version == "" {
		opts.Version = Version
	}
	if opts.GitHash == "" {
		opts.GitHash = ResolveGitHash()
	}

	var stderrWriter io.Writer
	if opts.Writer != nil {
		stderrWriter = opts.Writer
	} else if opts.Logfile != "" {
		stderrWriter = &lumberjack.Logger{
			Filename:   opts.Logfile,
			MaxSize:    20, // megabytes
			MaxBackups: 3,
		}
	} else {
		stderrWriter = os.Stderr
	}

	stderrLogger := log.New(stderrWriter, "", log.LstdFlags)
	shortMemoryWriter := memorywriter.New(2000, 200, false, nil)

	verboseWriter := stderrWriter
	if !opts.Verbose {
		verboseWriter = nil
	}
	longMemoryWriter := memorywriter.New(90000, 200, true, verboseWriter)

	stderrLogger.Printf("trezord v%s (rev %s) is starting on port %d", opts.Version, opts.GitHash, opts.Port)
	if core.IsDebugBinary() {
		stderrLogger.Print("!! DEBUG mode enabled! Please contact Trezor support in case you did not initiate this. !!")
	}

	buses, err := initUSB(opts.WithUSB, longMemoryWriter)
	if err != nil {
		return nil, err
	}

	longMemoryWriter.Log(fmt.Sprintf("UDP port count - %d", len(opts.UDPPorts)))

	touples := append([]usb.PortTouple{}, opts.UDPPairs...)
	for _, t := range opts.UDPPorts {
		touples = append(touples, usb.PortTouple{
			Normal: t,
			Debug:  0,
		})
	}
	if len(touples) > 0 {
		e, errUDP := usb.InitUDP(touples, longMemoryWriter)
		if errUDP != nil {
			return nil, errUDP
		}
		buses = append(buses, e)
	}

	if len(buses) == 0 {
		return nil, fmt.Errorf("no transports enabled")
	}

	b := usb.Init(buses...)
	longMemoryWriter.Log("Creating core")
	c := core.New(b, longMemoryWriter, allowCancel(), opts.Reset)

	longMemoryWriter.Log("Creating HTTP server")
	s, err := server.New(c, opts.Port, stderrWriter, shortMemoryWriter, longMemoryWriter, opts.Version, opts.GitHash)
	if err != nil {
		b.Close()
		return nil, fmt.Errorf("https: %w", err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", opts.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		b.Close()
		return nil, fmt.Errorf("cannot listen on %s: %w (is another Bridge already running?)", addr, err)
	}

	svc := &Service{
		opts:   opts,
		core:   c,
		bus:    b,
		server: s,
		done:   make(chan error, 1),
	}

	longMemoryWriter.Log("Running HTTP server")
	go func() {
		serveErr := s.Serve(ln)
		if serveErr == http.ErrServerClosed {
			serveErr = nil
		}
		svc.done <- serveErr
	}()

	return svc, nil
}

// Stop gracefully shuts down the HTTP server and closes USB.
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.server.Shutdown(ctx)
	if s.bus != nil {
		s.bus.Close()
	}
	s.server = nil
	s.bus = nil
	s.core = nil

	select {
	case serveErr := <-s.done:
		if err == nil {
			err = serveErr
		}
	case <-time.After(6 * time.Second):
	}

	return err
}

// Port returns the configured HTTP port.
func (s *Service) Port() int {
	return s.opts.Port
}

// VersionString returns the running version.
func (s *Service) VersionString() string {
	return s.opts.Version
}

// GitHash returns the git revision.
func (s *Service) GitHash() string {
	return s.opts.GitHash
}

// Enumerate lists connected devices.
func (s *Service) Enumerate() ([]core.EnumerateEntry, error) {
	s.mu.Lock()
	c := s.core
	s.mu.Unlock()
	if c == nil {
		return nil, fmt.Errorf("bridge is not running")
	}
	return c.Enumerate()
}

// Wait blocks until the HTTP server stops and returns any serve error.
func (s *Service) Wait() error {
	return <-s.done
}

func initUSB(init bool, wr *memorywriter.MemoryWriter) ([]core.USBBus, error) {
	if !init {
		return nil, nil
	}

	wr.Log("Initing libusb")
	w, err := usb.InitLibUSB(wr, !usb.HIDUse, allowCancel(), detachKernelDriver())
	if err != nil {
		return nil, fmt.Errorf("libusb: %w", err)
	}

	if !usb.HIDUse {
		return []core.USBBus{w}, nil
	}

	wr.Log("Initing hidapi")
	h, err := usb.InitHIDAPI(wr)
	if err != nil {
		w.Close()
		return nil, fmt.Errorf("hidapi: %w", err)
	}
	return []core.USBBus{w, h}, nil
}

func allowCancel() bool {
	return runtime.GOOS != "freebsd" && runtime.GOOS != "openbsd"
}

func detachKernelDriver() bool {
	return runtime.GOOS == "linux"
}
