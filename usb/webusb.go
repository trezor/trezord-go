package usb

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"github.com/google/gousb"
)

const (
	webusbPrefix  = "web"
	webIfaceNum   = 0
	webAltSetting = 0
	webEpIn       = 0x81
	webEpOut      = 0x01
)

type WebUSB struct {
	usb *gousb.Context
}

func InitWebUSB() (*WebUSB, error) {
	ctx := gousb.NewContext()

	return &WebUSB{
		usb: ctx,
	}, nil
}

func (b *WebUSB) Close() {
	b.usb.Close()
}

func (b *WebUSB) Enumerate() ([]Info, error) {
	var infos []Info
	_, err := b.usb.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if b.match(desc) {
			path := b.identify(desc)
			infos = append(infos, Info{
				Path:      path,
				VendorID:  int(desc.Vendor),
				ProductID: int(desc.Product),
			})
		}
		return false
	})
	if err != nil {
		return nil, err
	}
	return infos, nil
}

func (b *WebUSB) Has(path string) bool {
	return strings.HasPrefix(path, webusbPrefix)
}

func (b *WebUSB) Connect(path string) (Device, error) {

	first := false

	devs, err := b.usb.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if b.match(desc) {
			dpath := b.identify(desc)
			same := dpath == path
			if same && !first {
				first = true
				return true
			}
			return false
		}
		return false
	})

	if err != nil {
		for _, d := range devs {
			d.Close()
		}
		return nil, err
	}

	if len(devs) == 0 {
		return nil, ErrNotFound
	}
	return b.connect(devs[0])
}

func (b *WebUSB) connect(dev *gousb.Device) (*WUD, error) {
	dev.Reset()
	iface, done, err := dev.DefaultInterface()
	if err != nil {
		dev.Close()
		return nil, err
	}

	outEndpoint, err := iface.OutEndpoint(webEpOut)
	if err != nil {
		dev.Close()
		return nil, err
	}

	inEndpoint, err := iface.InEndpoint(webEpIn)
	if err != nil {
		dev.Close()
		return nil, err
	}

	return &WUD{
		inEndpoint:  inEndpoint,
		outEndpoint: outEndpoint,
		device:      dev,
		done:        done,
	}, nil
}

func (b *WebUSB) match(desc *gousb.DeviceDesc) bool {
	vid := desc.Vendor
	pid := desc.Product
	trezor1 := vid == vendorT1 && (pid == productT1Firmware || pid == productT1Bootloader)
	trezor2 := vid == vendorT2 && (pid == productT2Firmware || pid == productT2Bootloader)
	if !trezor1 && !trezor2 {
		return false
	}

	if len(desc.Configs) < 1 {
		return false
	}
	config := desc.Configs[1]

	if len(config.Interfaces) <= webIfaceNum {
		return false
	}
	iface := config.Interfaces[webIfaceNum]

	if len(iface.AltSettings) <= webAltSetting {
		return false
	}

	altsetting := iface.AltSettings[webAltSetting]
	return altsetting.Class == 0xff
}

func (b *WebUSB) identify(desc *gousb.DeviceDesc) string {
	bus := strconv.Itoa(int(desc.Bus))
	address := strconv.Itoa(int(desc.Address))
	path := bus + "-" + address
	pathb := []byte(path)
	digest := sha256.Sum256(pathb)
	hexed := hex.EncodeToString(digest[:])

	return webusbPrefix + hexed
}

type WUD struct {
	inEndpoint  *gousb.InEndpoint
	outEndpoint *gousb.OutEndpoint
	done        func()
	device      *gousb.Device
}

func (d *WUD) Close() error {
	d.done()
	err := d.device.Close()

	return err
}

var closedDeviceError = errors.New("Closed device")

func (d *WUD) Write(buf []byte) (int, error) {
	return d.outEndpoint.Write(buf)
}

func (d *WUD) Read(buf []byte) (int, error) {
	return d.inEndpoint.Read(buf)
}
