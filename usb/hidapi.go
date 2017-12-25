package usb

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/karalabe/hid"
)

const (
	hidapiPrefix = "hid"
	hidIfaceNum  = 0
	hidUsagePage = 0xFF00
)

type HIDAPI struct{}

func InitHIDAPI() (*HIDAPI, error) {
	return &HIDAPI{}, nil
}

func (b *HIDAPI) Enumerate() ([]string, error) {
	list := hid.Enumerate(0, 0)

	var paths []string

	for _, dev := range list {
		if b.match(&dev) {
			paths = append(paths, b.identify(&dev))
		}
	}
	return paths, nil
}

func (b *HIDAPI) Has(path string) bool {
	return strings.HasPrefix(path, hidapiPrefix)
}

func (b *HIDAPI) Connect(path string) (Device, error) {
	list := hid.Enumerate(0, 0)

	for _, dev := range list {
		if b.match(&dev) && b.identify(&dev) == path {
			d, err := dev.Open()
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

func (b *HIDAPI) match(dev *hid.DeviceInfo) bool {
	return (dev.VendorID == vendorT1 &&
		dev.ProductID == productT1 &&
		(dev.Interface == hidIfaceNum || dev.UsagePage == hidUsagePage))
}

func (b *HIDAPI) identify(dev *hid.DeviceInfo) string {
	path := []byte(dev.Path)
	digest := sha256.Sum256(path)
	return hidapiPrefix + hex.EncodeToString(digest[:])
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
