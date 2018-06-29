package usb

import (
	"errors"
	"fmt"

	"github.com/trezor/trezord-go/core"
)

var (
	ErrNotFound = fmt.Errorf("device not found")
)

type Info = core.USBInfo
type Device = core.USBDevice
type Bus = core.USBBus

type USB struct {
	buses []Bus
}

func Init(buses ...Bus) *USB {
	return &USB{
		buses: buses,
	}
}

func (b *USB) Has(path string) bool {
	for _, b := range b.buses {
		if b.Has(path) {
			return true
		}
	}
	return false
}

func (b *USB) Enumerate() ([]Info, error) {
	var infos []Info

	for _, b := range b.buses {
		l, err := b.Enumerate()
		if err != nil {
			return nil, err
		}
		infos = append(infos, l...)
	}
	return infos, nil
}

func (b *USB) Connect(path string) (Device, error) {
	for _, b := range b.buses {
		if b.Has(path) {
			return b.Connect(path)
		}
	}
	return nil, ErrNotFound
}

var errDisconnect = errors.New("device disconnected during action")
var errClosedDevice = errors.New("closed device")
