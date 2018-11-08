package usb

import (
	"errors"
	"fmt"

	"github.com/trezor/trezord-go/core"
)

var (
	ErrNotFound = fmt.Errorf("device not found")
)

type USB struct {
	buses []core.USBBus
}

func Init(buses ...core.USBBus) *USB {
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

func (b *USB) Enumerate() ([]core.USBInfo, error) {
	var infos []core.USBInfo

	for _, b := range b.buses {
		l, err := b.Enumerate()
		if err != nil {
			return nil, err
		}
		infos = append(infos, l...)
	}
	return infos, nil
}

func (b *USB) Connect(path string, debug bool, reset bool) (core.USBDevice, error) {
	for _, b := range b.buses {
		if b.Has(path) {
			return b.Connect(path, debug, reset)
		}
	}
	return nil, ErrNotFound
}

var errDisconnect = errors.New("device disconnected during action")
var errClosedDevice = errors.New("closed device")
var errNotDebug = errors.New("not debug link")
