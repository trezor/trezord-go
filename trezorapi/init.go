package trezorapi

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"

	"github.com/trezor/trezord-go/internal/core"
	"github.com/trezor/trezord-go/internal/logs"
	"github.com/trezor/trezord-go/internal/usb"
)

// API connects to devices.
// It recognizes, if Bridge is already running or not, and connects to it,
// unless this is disabled with DisableBridge InitOption.
type API struct {
	// init option
	touples []usb.PortTouple

	// actual core
	c      transport
	logger *logs.Logger

	// other init options
	writer    io.Writer
	reset     bool
	withUSB   bool
	bridge    bool
	bridgeURL string
}

var defaultAPI = API{
	withUSB: true,
	reset:   true,
	writer:  ioutil.Discard,

	bridge:    true,
	bridgeURL: "http://127.0.0.1:21325",
}

// InitOption is an option that could be given to API. Ideally, you don't need to
// add any option, or maybe adding UDP for testing with an emulator.
//
// Note - when API has enabled bridge (enabled by default),
// and bridge is running, API is ignoring all the settings
// about USB and UDP, since all the connections are going through the bridge!
type InitOption func(*API)

// See https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
// and https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html
// for notes on the initializer design

// BridgeURL sets Bridge URL. It's usually not needed;
// the correct URL is set by default.
func BridgeURL(s string) InitOption {
	return func(a *API) {
		a.bridgeURL = s
		a.bridge = true
	}
}

// DisableBridge disables connecting to bridge,
// forcing API to always connect to USB level
func DisableBridge() InitOption {
	return func(a *API) {
		a.bridge = false
		a.bridgeURL = ""
	}
}

// WithUSB enables or disables USB. Irrelevant if bridge is used.
//
// It's sometimes necessary to disable USB, for example, when
// on Travis or in Docker. (You should, however, enable UDP)
func WithUSB(b bool) InitOption {
	return func(a *API) {
		a.withUSB = b
	}
}

// ResetDeviceOnAcquire sets if device is hard-reset on each acquiring. On by default.
// Irrelevant if bridge is used.
//
// This is useful to disable when testing device with debug link, since every acquire
// resets the connection, which closes the debug link.
func ResetDeviceOnAcquire(b bool) InitOption {
	return func(a *API) {
		a.reset = b
	}
}

// LogWriter sets up writer for writing debug logs.
// Not used for anything if bridge is used.
//
// Note that API is writing a LOT of USB debug logs - libusb is set to
// the highest verbosity, etc.
func LogWriter(w io.Writer) InitOption {
	return func(a *API) {
		a.writer = w
	}
}

// AddUDPPort adds a UDP port for emulator.
// Works with t1 and t2 emulators.
func AddUDPPort(i int) InitOption {
	return func(a *API) {
		a.touples = append(a.touples, usb.PortTouple{
			Normal: i,
			Debug:  0,
		})
	}
}

// AddUDPTouple adds 2 UDP ports for emulator.
//
// Emulator can have both normal connection and debug link
func AddUDPTouple(normal int, debug int) InitOption {
	return func(a *API) {
		a.touples = append(a.touples, usb.PortTouple{
			Normal: normal,
			Debug:  debug,
		})
	}
}

func initUsb(wr *logs.Logger) ([]core.USBBus, error) {
	wr.Log("Initing libusb")

	w, err := usb.InitLibUSB(wr, !usb.HIDUse, allowCancel(), detachKernelDriver())
	if err != nil {
		return nil, fmt.Errorf("libusb: %s", err)
	}

	// linux has no HID
	if !usb.HIDUse {
		return []core.USBBus{w}, nil
	}

	wr.Log("Initing hidapi")
	h, err := usb.InitHIDAPI(wr)
	if err != nil {
		return nil, fmt.Errorf("hidapi: %s", err)
	}
	return []core.USBBus{w, h}, nil
}

// Close cleans up USB connection. Should be used at the end of program.
func (a *API) Close() {
	a.c.Close()
}

// New creates an API. See InitOption documentation.
func New(options ...InitOption) (*API, error) {
	api := defaultAPI // copy struct
	for _, option := range options {
		option(&api)
	}

	var t transport
	if api.bridge {
		b, err := newBridge(api.bridgeURL)
		if err == nil {
			t = b
		}
	}

	// note - if bridge initialized, nothing else is (including UDP)
	if t == nil {
		c, err := api.initLowlevel()
		if err != nil {
			return nil, err
		}
		t = c
	}
	api.c = t
	return &api, nil
}

func (a *API) initLowlevel() (transport, error) {
	a.logger = &logs.Logger{Writer: a.writer}

	bus := []core.USBBus{}

	if a.withUSB {
		newbus, err := initUsb(a.logger)
		if err != nil {
			return nil, err
		}
		bus = newbus
	}

	a.logger.Log(fmt.Sprintf("UDP port count - %d", len(a.touples)))

	if len(a.touples) > 0 {
		e, errUDP := usb.InitUDP(a.touples, a.logger)
		if errUDP != nil {
			return nil, errUDP
		}
		bus = append(bus, e)
	}

	if len(bus) == 0 {
		return nil, errors.New("no transports enabled")
	}

	b := usb.Init(bus...)

	a.logger.Log("Creating core")
	c := core.New(b, a.logger, allowCancel(), a.reset)
	return c, nil
}

// Does OS allow sync canceling via our custom libusb patches?
func allowCancel() bool {
	return runtime.GOOS != "freebsd"
}

// Does OS detach kernel driver in libusb?
func detachKernelDriver() bool {
	return runtime.GOOS == "linux"
}
