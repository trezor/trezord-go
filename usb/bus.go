package usb

import (
	"fmt"
	"io"
)

const (
	vendorT1  = 0x534c
	productT1 = 0x0001
)

var (
	ErrNotFound = fmt.Errorf("device not found")
)

type Device interface {
	io.ReadWriteCloser
}

type Bus interface {
	Enumerate() ([]string, error)
	Connect(id string) (Device, error)
	Has(id string) bool
}

type USB struct {
	buses []Bus
}

func Init(buses ...Bus) *USB {
	return &USB{
		buses: buses,
	}
}

func (b *USB) Enumerate() ([]string, error) {
	var paths []string

	for _, b := range b.buses {
		l, err := b.Enumerate()
		if err != nil {
			return nil, err
		}
		paths = append(paths, l...)
	}
	return paths, nil
}

func (b *USB) Connect(path string) (Device, error) {
	for _, b := range b.buses {
		if b.Has(path) {
			return b.Connect(path)
		}
	}
	return nil, ErrNotFound
}
