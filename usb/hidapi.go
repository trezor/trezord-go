package usb

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"strings"

	"github.com/trezor/usbhid"
)

const (
	hidapiPrefix = "hid"
	hidIfaceNum  = 0
	hidUsagePage = 0xFF00
)

type HIDAPI struct {
	logger, dlogger *log.Logger
}

func InitHIDAPI(logger, dlogger *log.Logger) (*HIDAPI, error) {
	return &HIDAPI{
		logger:  logger,
		dlogger: dlogger,
	}, nil
}

func (b *HIDAPI) Enumerate() ([]Info, error) {
	var infos []Info

	b.dlogger.Println("hidapi - enumerate - low level")
	devs := usbhid.HidEnumerate(0, 0)

	b.dlogger.Println("hidapi - enumerate - low level done")

	for _, dev := range devs { // enumerate all devices
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
	b.dlogger.Println("hidapi - connect - enumerate to find")
	devs := usbhid.HidEnumerate(0, 0)
	b.dlogger.Println("hidapi - connect - enumerate done")

	for _, dev := range devs { // enumerate all devices
		if b.match(&dev) && b.identify(&dev) == path {
			b.dlogger.Println("hidapi - connect - low level open")
			d, err := dev.Open()
			if err != nil {
				return nil, err
			}
			b.dlogger.Println("hidapi - connect - detecting prepend")
			prepend, err := detectPrepend(d)
			if err != nil {
				return nil, err
			}
			b.dlogger.Printf("hidapi - connect - done (prepend %t)", prepend)
			return &HID{
				dev:     d,
				prepend: prepend,
				dlogger: b.dlogger,
			}, nil
		}
	}
	return nil, ErrNotFound
}

func (b *HIDAPI) match(d *usbhid.HidDeviceInfo) bool {
	vid := d.VendorID
	pid := d.ProductID
	trezor1 := vid == VendorT1 && (pid == ProductT1Firmware)
	trezor2 := vid == VendorT2 && (pid == ProductT2Firmware || pid == ProductT2Bootloader)
	return (trezor1 || trezor2) && (d.Interface == hidIfaceNum || d.UsagePage == hidUsagePage)
}

func (b *HIDAPI) identify(dev *usbhid.HidDeviceInfo) string {
	path := []byte(dev.Path)
	digest := sha256.Sum256(path)
	return hidapiPrefix + hex.EncodeToString(digest[:])
}

type HID struct {
	dev     *usbhid.HidDevice
	prepend bool // on windows, see detectPrepend

	dlogger *log.Logger
}

func (d *HID) Close() error {
	d.dlogger.Println("hidapi - close - low level close")
	err := d.dev.Close()
	d.dlogger.Println("hidapi - close - low level close done")
	return err
}

var unknownErrorMessage = "hidapi: unknown failure"

// This will write a useless buffer to trezor
// to test whether it is an older HID version on reportid 63
// or a newer one that is on id 0.
// The older one does not need prepending, the newer one does
// This makes difference only on windows
func detectPrepend(dev *usbhid.HidDevice) (bool, error) {
	buf := []byte{63}
	for i := 0; i < 63; i++ {
		buf = append(buf, 0xff)
	}

	// first test newer version
	w, _ := dev.Write(buf, true)
	if w == 65 {
		return true, nil
	}

	// then test older version
	w, err := dev.Write(buf, false)
	if err != nil {
		return false, err
	}
	if w == 64 {
		return false, nil
	}

	return false, errors.New("Unknown HID version")
}

func (d *HID) readWrite(buf []byte, read bool) (int, error) {
	var w int
	var err error

	if read {
		d.dlogger.Println("hidapi - read - start")
		w, err = d.dev.Read(buf)
		d.dlogger.Println("hidapi - read - end")
	} else {
		d.dlogger.Println("hidapi - write - start")
		w, err = d.dev.Write(buf, d.prepend)
		d.dlogger.Println("hidapi - write - end")
	}
	if err == nil {
		d.dlogger.Println("hidapi - readwrite - err nil")
	} else {
		d.dlogger.Printf("hidapi - readwrite - err %s", err.Error())
	}

	if err != nil && err.Error() == unknownErrorMessage {
		return 0, errDisconnect
	}
	return w, err
}

func (d *HID) Write(buf []byte) (int, error) {
	return d.readWrite(buf, false)
}

func (d *HID) Read(buf []byte) (int, error) {
	return d.readWrite(buf, true)
}
