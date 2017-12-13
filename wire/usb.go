package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/deadsy/libusb"
)

type Bus struct {
	usb libusb.Context
	Vid uint16
	Pid uint16
}

func Init(vid uint16, pid uint16, debug int) (*Bus, error) {
	var usb libusb.Context

	err := libusb.Init(&usb)
	if err != nil {
		return nil, err
	}
	libusb.Set_Debug(usb, debug)

	return &Bus{
		Vid: vid,
		Pid: pid,
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
		if b.Vid == dd.IdVendor && b.Pid == dd.IdProduct {
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

func locate(usb libusb.Context, id string) (libusb.Device_Handle, error) {
	list, err := libusb.Get_Device_List(usb)
	if err != nil {
		return nil, err
	}
	defer libusb.Free_Device_List(list, 1) // unlink devices

	for _, dev := range list {
		if identify(dev) != id {
			continue
		}
		return libusb.Open(dev)
	}
	return nil, ErrNotFound
}

type Device struct {
	dev libusb.Device_Handle
	vid uint16
	pid uint16
}

func (b *Bus) Connect(id string) (*Device, error) {
	dev, err := locate(b.usb, id)
	if err != nil {
		return nil, err
	}
	err = libusb.Claim_Interface(dev, ifaceNum)
	if err != nil {
		libusb.Close(dev)
		return nil, err
	}
	return &Device{
		dev: dev,
	}, nil
}

func (d *Device) Close() {
	libusb.Close(d.dev)
}

func (d *Device) Write(buf []byte) (int, error) {
	p, err := libusb.Interrupt_Transfer(d.dev, epOut, buf, epTimeout)
	return len(p), err
}

func (d *Device) Read(buf []byte) (int, error) {
	p, err := libusb.Interrupt_Transfer(d.dev, epIn, buf, epTimeout)
	return len(p), err
}
