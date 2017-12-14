package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/deadsy/libusb"
)

const (
	vendorID  = 0x534c
	productID = 0x0001
)

type Bus struct {
	usb libusb.Context
}

func Init(debug int) (*Bus, error) {
	var usb libusb.Context

	err := libusb.Init(&usb)
	if err != nil {
		return nil, err
	}
	libusb.Set_Debug(usb, debug)

	return &Bus{
		usb: usb,
	}, nil
}

func (b *Bus) Close() {
	libusb.Exit(b.usb)
}

func (b *Bus) Enumerate() ([]string, error) {
	list, err := libusb.Get_Device_List(b.usb)
	if err != nil {
		return nil, err
	}
	defer libusb.Free_Device_List(list, 1) // unlink devices

	var ids []string

	for _, dev := range list {
		dd, err := libusb.Get_Device_Descriptor(dev)
		if err != nil {
			return nil, err
		}
		if vendorID == dd.IdVendor && productID == dd.IdProduct {
			ids = append(ids, identify(dev))
		}
	}
	return ids, nil
}

const (
	ifaceNum  = 0
	epIn      = 0x82
	epOut     = 0x02
	epTimeout = 0
)

var (
	ErrNotFound = fmt.Errorf("device not found")
)

func identify(dev libusb.Device) string {
	var (
		bus    = libusb.Get_Bus_Number(dev)
		addr   = libusb.Get_Device_Address(dev)
		desc   = fmt.Sprintf("%d:%d", bus, addr)
		digest = sha256.Sum256([]byte(desc))
	)
	return hex.EncodeToString(digest[:])
}

func (b *Bus) Connect(id string) (Device, error) {
	list, err := libusb.Get_Device_List(b.usb)
	if err != nil {
		return nil, err
	}
	defer libusb.Free_Device_List(list, 1) // unlink devices

	for _, dev := range list {
		if identify(dev) != id {
			continue
		}
		if isWebUSB(dev) {
			return b.connectWebUSB(dev)
		}
	}
	return nil, ErrNotFound
}

func (b *Bus) connectWebUSB(dev libusb.Device) (*WebUSB, error) {
	d, err := libusb.Open(dev)
	if err != nil {
		return nil, err
	}
	err = libusb.Claim_Interface(d, ifaceNum)
	if err != nil {
		libusb.Close(d)
		return nil, err
	}
	return &WebUSB{
		dev: d,
	}, nil
}

type Device interface {
	io.ReadWriteCloser
}

type WebUSB struct {
	dev libusb.Device_Handle
}

func isWebUSB(libusb.Device) bool {
	return true
}

func (d *WebUSB) Close() error {
	libusb.Close(d.dev)
	return nil
}

func (d *WebUSB) Write(buf []byte) (int, error) {
	p, err := libusb.Interrupt_Transfer(d.dev, epOut, buf, epTimeout)
	return len(p), err
}

func (d *WebUSB) Read(buf []byte) (int, error) {
	p, err := libusb.Interrupt_Transfer(d.dev, epIn, buf, epTimeout)
	return len(p), err
}
