package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/deadsy/libusb"
	"github.com/karalabe/hid"
)

const (
	vendorID  = 0x534c
	productID = 0x0001
)

var (
	ErrNotFound = fmt.Errorf("device not found")
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
		switch {
		case isHID(dev):
			return b.connectHID(dev)
		case isWebUSB(dev):
			return b.connectWebUSB(dev)
		}
	}
	return nil, ErrNotFound
}

func identify(dev libusb.Device) string {
	var (
		bus    = libusb.Get_Bus_Number(dev)
		addr   = libusb.Get_Device_Address(dev)
		desc   = fmt.Sprintf("%d:%d", bus, addr)
		digest = sha256.Sum256([]byte(desc))
	)
	return hex.EncodeToString(digest[:])
}

type Device interface {
	io.ReadWriteCloser
}

// WebUSB
// ======

const (
	webIfaceNum   = 0
	webAltSetting = 0
	webEpIn       = 0x81
	webEpOut      = 0x01
)

func isWebUSB(dev libusb.Device) bool {
	c, err := libusb.Get_Active_Config_Descriptor(dev)
	if err != nil {
		return false
	}
	return c.BNumInterfaces > webIfaceNum &&
		c.Interface[webIfaceNum].Num_altsetting > webAltSetting &&
		c.Interface[webIfaceNum].Altsetting[webAltSetting].BInterfaceClass == libusb.CLASS_VENDOR_SPEC
}

func (b *Bus) connectWebUSB(dev libusb.Device) (*WebUSB, error) {
	d, err := libusb.Open(dev)
	if err != nil {
		return nil, err
	}
	err = libusb.Claim_Interface(d, webIfaceNum)
	if err != nil {
		libusb.Close(d)
		return nil, err
	}
	return &WebUSB{
		dev: d,
	}, nil
}

type WebUSB struct {
	dev libusb.Device_Handle
}

func (d *WebUSB) Close() error {
	libusb.Close(d.dev)
	return nil
}

func (d *WebUSB) Write(buf []byte) (int, error) {
	p, err := libusb.Interrupt_Transfer(d.dev, webEpOut, buf, 0) // infinite timeout
	return len(p), err
}

func (d *WebUSB) Read(buf []byte) (int, error) {
	p, err := libusb.Interrupt_Transfer(d.dev, webEpIn, buf, 0) // infinite timeout
	return len(p), err
}

// HIDAPI
// ======

const (
	hidIfaceNum   = 0
	hidAltSetting = 0
	hidUsagePage  = 0xFF00
)

func isHID(dev libusb.Device) bool {
	c, err := libusb.Get_Active_Config_Descriptor(dev)
	if err != nil {
		return false
	}
	return c.BNumInterfaces > hidIfaceNum &&
		c.Interface[hidIfaceNum].Num_altsetting > hidAltSetting &&
		c.Interface[hidIfaceNum].Altsetting[hidAltSetting].BInterfaceClass == libusb.CLASS_HID
}

func (b *Bus) connectHID(dev libusb.Device) (*HID, error) {
	list := hid.Enumerate(0, 0)

	for _, info := range list {
		if vendorID != info.VendorID || productID != info.ProductID {
			continue
		}
		if hidIfaceNum == info.Interface || hidUsagePage == info.UsagePage {
			d, err := info.Open()
			if err != nil {
				return nil, err
			}
			return &HID{
				dev: d,
			}, nil
		}
	}
	return nil, ErrNotFound
}

type HID struct {
	dev *hid.Device
}

func (d *HID) Close() error {
	return d.dev.Close()
}

func (d *HID) Write(buf []byte) (int, error) {
	return d.dev.Write(buf)
}

func (d *HID) Read(buf []byte) (int, error) {
	return d.dev.Read(buf)
}
