package api

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/usb"
)

// See https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
// and https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html
// for notes on the initializer design

type API struct {
	c *core.Core
	b core.USBBus

	longMemoryWriter *memorywriter.MemoryWriter

	// init options
	withUSB bool
	reset   bool
	touples []usb.PortTouple
}

var defaultAPI = API{
	withUSB:          true,
	reset:            true,
	longMemoryWriter: memorywriter.Empty(),
}

type InitOption func(*API)

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

func LongMemoryWriter(m *memorywriter.MemoryWriter) InitOption {
	return func(a *API) {
		a.longMemoryWriter = m
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

func initUsb(wr *memorywriter.MemoryWriter) ([]core.USBBus, error) {
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

	bus := []core.USBBus{}

	if api.withUSB {
		newbus, err := initUsb(api.longMemoryWriter)
		if err != nil {
			return nil, err
		}
		bus = newbus
	}

	api.longMemoryWriter.Log(fmt.Sprintf("UDP port count - %d", len(api.touples)))

	if len(api.touples) > 0 {
		e, errUDP := usb.InitUDP(api.touples, api.longMemoryWriter)
		if errUDP != nil {
			return nil, errUDP
		}
		bus = append(bus, e)
	}

	if len(bus) == 0 {
		return nil, errors.New("no transports enabled")
	}

	b := usb.Init(bus...)
	api.b = b

	api.longMemoryWriter.Log("Creating core")
	c := core.New(b, api.longMemoryWriter, allowCancel(), api.reset)
	api.c = c
	return &api, nil
}

// Does OS allow sync canceling via our custom libusb patches?
func allowCancel() bool {
	return runtime.GOOS != "freebsd"
}

// Does OS detach kernel driver in libusb?
func detachKernelDriver() bool {
	return runtime.GOOS == "linux"
}
