package usb

import (
	"encoding/hex"
	"strings"

	"github.com/trezor/usbhid"
)

const (
	webusbPrefix  = "web"
	webIfaceNum   = 0
	webAltSetting = 0
	webEpIn       = 0x81
	webEpOut      = 0x01
)

type WebUSB struct {
	usb usbhid.Context
}

func InitWebUSB(debug int) (*WebUSB, error) {
	var usb usbhid.Context
	err := usbhid.Init(&usb)
	if err != nil {
		return nil, err
	}
	usbhid.Set_Debug(usb, debug)

	return &WebUSB{
		usb: usb,
	}, nil
}

func (b *WebUSB) Close() {
	usbhid.Exit(b.usb)
}

func (b *WebUSB) Enumerate() ([]Info, error) {
	list, err := usbhid.Get_Device_List(b.usb)
	if err != nil {
		return nil, err
	}
	defer usbhid.Free_Device_List(list, 1) // unlink devices

	var infos []Info

	for _, dev := range list {
		if b.match(dev) {
			dd, err := usbhid.Get_Device_Descriptor(dev)
			if err != nil {
				continue
			}
			infos = append(infos, Info{
				Path:      b.identify(dev),
				VendorID:  int(dd.IdVendor),
				ProductID: int(dd.IdProduct),
			})
		}
	}
	return infos, nil
}

func (b *WebUSB) Has(path string) bool {
	return strings.HasPrefix(path, webusbPrefix)
}

func (b *WebUSB) Connect(path string) (Device, error) {
	list, err := usbhid.Get_Device_List(b.usb)
	if err != nil {
		return nil, err
	}
	defer usbhid.Free_Device_List(list, 1) // unlink devices

	for _, dev := range list {
		if b.match(dev) && b.identify(dev) == path {
			return b.connect(dev)
		}
	}
	return nil, ErrNotFound
}

func (b *WebUSB) connect(dev usbhid.Device) (*WUD, error) {
	d, err := usbhid.Open(dev)
	if err != nil {
		return nil, err
	}
	err = usbhid.Claim_Interface(d, webIfaceNum)
	if err != nil {
		usbhid.Close(d)
		return nil, err
	}
	return &WUD{
		dev: d,
	}, nil
}

func (b *WebUSB) match(dev usbhid.Device) bool {
	dd, err := usbhid.Get_Device_Descriptor(dev)
	if err != nil {
		return false
	}
	vid := dd.IdVendor
	pid := dd.IdProduct
	trezor1 := vid == vendorT1 && (pid == productT1Firmware || pid == productT1Bootloader)
	trezor2 := vid == vendorT2 && (pid == productT2Firmware || pid == productT2Bootloader)
	if !trezor1 && !trezor2 {
		return false
	}
	c, err := usbhid.Get_Active_Config_Descriptor(dev)
	if err != nil {
		return false
	}
	return (c.BNumInterfaces > webIfaceNum &&
		c.Interface[webIfaceNum].Num_altsetting > webAltSetting &&
		c.Interface[webIfaceNum].Altsetting[webAltSetting].BInterfaceClass == usbhid.CLASS_VENDOR_SPEC)
}

func (b *WebUSB) identify(dev usbhid.Device) string {
	var ports [8]byte
	p, err := usbhid.Get_Port_Numbers(dev, ports[:])
	if err != nil {
		return ""
	}
	return webusbPrefix + hex.EncodeToString(p)
}

type WUD struct {
	dev usbhid.Device_Handle
}

func (d *WUD) Close() error {
	usbhid.Close(d.dev)
	return nil
}

func (d *WUD) Write(buf []byte) (int, error) {
	p, err := usbhid.Interrupt_Transfer(d.dev, webEpOut, buf, 0) // infinite timeout
	return len(p), err
}

func (d *WUD) Read(buf []byte) (int, error) {
	p, err := usbhid.Interrupt_Transfer(d.dev, webEpIn, buf, 0) // infinite timeout
	return len(p), err
}
