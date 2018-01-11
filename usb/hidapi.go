package usb

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/trezor/usbhid"
)

const (
	hidapiPrefix = "hid"
	hidIfaceNum  = 0
	hidUsagePage = 0xFF00
)

type HIDAPI struct {
}

func InitHIDAPI() (*HIDAPI, error) {
	return &HIDAPI{}, nil
}

func (b *HIDAPI) Enumerate() ([]Info, error) {
	var infos []Info

	for _, dev := range usbhid.HidEnumerate(0, 0) { // enumerate all devices
		if b.match(&dev) {
			infos = append(infos, Info{
				Path:      b.identify(&dev),
				VendorID:  int(dev.VendorID),
				ProductID: int(dev.ProductID),
			})
		}
	}
	return infos, nil
}

func (b *HIDAPI) Has(path string) bool {
	return strings.HasPrefix(path, hidapiPrefix)
}

func (b *HIDAPI) Connect(path string) (Device, error) {
	for _, dev := range usbhid.HidEnumerate(0, 0) { // enumerate all devices
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

func (b *HIDAPI) match(d *usbhid.HidDeviceInfo) bool {
	vid := d.VendorID
	pid := d.ProductID
	trezor1 := vid == vendorT1 && (pid == productT1Firmware || pid == productT1Bootloader)
	trezor2 := vid == vendorT2 && (pid == productT2Firmware || pid == productT2Bootloader)
	return (trezor1 || trezor2) && (d.Interface == hidIfaceNum || d.UsagePage == hidUsagePage)
}

func (b *HIDAPI) identify(dev *usbhid.HidDeviceInfo) string {
	path := []byte(dev.Path)
	digest := sha256.Sum256(path)
	return hidapiPrefix + hex.EncodeToString(digest[:])
}

type HID struct {
	dev *usbhid.HidDevice
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
