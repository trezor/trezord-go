package api

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

// See https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
// and https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html
// for notes on the initializer design

type API struct {
	touples []usb.PortTouple

	c      transport
	b      core.USBBus
	logger *logs.Logger

	// init options
	writer  io.Writer
	reset   bool
	withUSB bool

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

type InitOption func(*API)

func BridgeURL(s string) InitOption {
	return func(a *API) {
		a.bridgeURL = s
		a.bridge = true
	}
}

func DisableBridge() InitOption {
	return func(a *API) {
		a.bridge = false
		a.bridgeURL = ""
	}
}

func WithUSB(b bool) InitOption {
	return func(a *API) {
		a.withUSB = b
	}
}

func ResetDeviceOnAcquire(b bool) InitOption {
	return func(a *API) {
		a.reset = b
	}
}

func LogWriter(w io.Writer) InitOption {
	return func(a *API) {
		a.writer = w
	}
}

func AddUDPPort(i int) InitOption {
	return func(a *API) {
		a.touples = append(a.touples, usb.PortTouple{
			Normal: i,
			Debug:  0,
		})
	}
}

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

func (a *API) Close() {
	a.b.Close()
}

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
	a.b = b

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
