package usb

import (
	"encoding/hex"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/trezor/usbhid"
)

const (
	webusbPrefix  = "web"
	webConfigNum  = 1
	webIfaceNum   = 0
	webAltSetting = 0
	webEpIn       = 0x81
	webEpOut      = 0x01
	usbTimeout    = 5000
)

type WebUSB struct {
	usb             usbhid.Context
	logger, dlogger *log.Logger
}

func InitWebUSB(logger, dlogger *log.Logger) (*WebUSB, error) {
	var usb usbhid.Context
	dlogger.Println("webusb - init")
	err := usbhid.Init(&usb)
	if err != nil {
		return nil, err
	}
	usbhid.Set_Debug(usb, int(usbhid.LOG_LEVEL_NONE))

	dlogger.Println("webusb - init done")

	return &WebUSB{
		usb:     usb,
		logger:  logger,
		dlogger: dlogger,
	}, nil
}

func (b *WebUSB) Close() {
	b.dlogger.Println("webusb - all close (should happen only on exit)")
	usbhid.Exit(b.usb)
}

func (b *WebUSB) Enumerate() ([]Info, error) {
	b.dlogger.Println("webusb - enum - low level enumerating")
	list, err := usbhid.Get_Device_List(b.usb)

	if err != nil {
		return nil, err
	}
	b.dlogger.Println("webusb - enum - low level enumerating done")

	defer func() {
		b.dlogger.Println("webusb - enum - freeing device list")
		usbhid.Free_Device_List(list, 1) // unlink devices
		b.dlogger.Println("webusb - enum - freeing device list done")
	}()

	var infos []Info

	for _, dev := range list {
		if b.match(dev) {
			b.dlogger.Println("webusb - enum - getting device descriptor")
			dd, err := usbhid.Get_Device_Descriptor(dev)
			if err != nil {
				b.dlogger.Printf("webusb - enum - error getting device descriptor %s", err.Error())
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
	b.dlogger.Println("webusb - connect - low level enumerating")
	list, err := usbhid.Get_Device_List(b.usb)

	if err != nil {
		return nil, err
	}
	b.dlogger.Println("webusb - connect - low level enumerating done")

	defer func() {
		b.dlogger.Println("webusb - connect - freeing device list")
		usbhid.Free_Device_List(list, 1) // unlink devices
		b.dlogger.Println("webusb - connect - freeing device list done")
	}()

	for _, dev := range list {
		if b.match(dev) && b.identify(dev) == path {
			return b.connect(dev)
		}
	}
	return nil, ErrNotFound
}

func (b *WebUSB) connect(dev usbhid.Device) (*WUD, error) {
	b.dlogger.Println("webusb - connect - low level")
	d, err := usbhid.Open(dev)
	if err != nil {
		return nil, err
	}
	b.dlogger.Println("webusb - connect - reset")
	err = usbhid.Reset_Device(d)
	if err != nil {
		// don't abort if reset fails
		// usbhid.Close(d)
		// return nil, err
		b.logger.Printf("Warning: error at device reset: %s", err)
	}

	currConf, err := usbhid.Get_Configuration(d)
	if err != nil {
		b.dlogger.Printf("webusb - connect - current configuration err %s", err.Error())
	} else {
		b.dlogger.Printf("webusb - connect - current configuration %d", currConf)
	}

	b.dlogger.Println("webusb - connect - set_configuration")
	err = usbhid.Set_Configuration(d, webConfigNum)
	if err != nil {
		// don't abort if set configuration fails
		// usbhid.Close(d)
		// return nil, err
		b.logger.Printf("Warning: error at configuration set: %s", err)
	}

	currConf, err = usbhid.Get_Configuration(d)
	if err != nil {
		b.dlogger.Printf("webusb - connect - current configuration err %s", err.Error())
	} else {
		b.dlogger.Printf("webusb - connect - current configuration %d", currConf)
	}

	b.dlogger.Println("webusb - connect - claiming interface")
	err = usbhid.Claim_Interface(d, webIfaceNum)
	if err != nil {
		b.dlogger.Println("webusb - connect - claiming interface failed")
		usbhid.Close(d)
		return nil, err
	}

	b.dlogger.Println("webusb - connect - claiming interface done")

	return &WUD{
		dev:    d,
		closed: 0,

		dlogger: b.dlogger,
	}, nil
}

func (b *WebUSB) match(dev usbhid.Device) bool {
	dd, err := usbhid.Get_Device_Descriptor(dev)
	if err != nil {
		b.dlogger.Println("webusb - match - error getting descriptor", err.Error())
		return false
	}

	vid := dd.IdVendor
	pid := dd.IdProduct
	if !b.matchVidPid(vid, pid) {
		return false
	}

	c, err := usbhid.Get_Active_Config_Descriptor(dev)
	if err != nil {
		b.dlogger.Printf("webusb - match - error getting config descriptor %s", err.Error())
		return false
	}
	return (c.BNumInterfaces > webIfaceNum &&
		c.Interface[webIfaceNum].Num_altsetting > webAltSetting &&
		c.Interface[webIfaceNum].Altsetting[webAltSetting].BInterfaceClass == usbhid.CLASS_VENDOR_SPEC)
}

func (b *WebUSB) matchVidPid(vid uint16, pid uint16) bool {
	trezor1 := vid == VendorT1 && (pid == ProductT1Firmware)
	trezor2 := vid == VendorT2 && (pid == ProductT2Firmware || pid == ProductT2Bootloader)
	return trezor1 || trezor2
}

func (b *WebUSB) identify(dev usbhid.Device) string {
	var ports [8]byte
	p, err := usbhid.Get_Port_Numbers(dev, ports[:])
	if err != nil {
		b.dlogger.Printf("webusb - identify - error getting port numbers %s", err.Error())
		return ""
	}
	return webusbPrefix + hex.EncodeToString(p)
}

type WUD struct {
	dev usbhid.Device_Handle

	closed int32 // atomic

	transferMutex sync.Mutex
	// closing cannot happen while interrupt_transfer is hapenning,
	// otherwise interrupt_transfer hangs forever

	dlogger *log.Logger
}

func (d *WUD) Close() error {
	d.dlogger.Println("webusb - close - storing d.closed")
	atomic.StoreInt32(&d.closed, 1)

	d.dlogger.Println("webusb - close - finishing read queue")
	d.finishReadQueue()

	d.dlogger.Println("webusb - close - wait for transferMutex lock")
	d.transferMutex.Lock()
	d.dlogger.Println("webusb - close - low level close")
	usbhid.Close(d.dev)
	d.transferMutex.Unlock()

	d.dlogger.Println("webusb - close - done")

	return nil
}

func (d *WUD) finishReadQueue() {
	d.dlogger.Println("webusb - close - rq - wait for transfermutex lock")
	d.transferMutex.Lock()
	var err error
	var buf [64]byte

	for err == nil {
		d.dlogger.Println("webusb - close - rq - transfer")
		_, err = usbhid.Interrupt_Transfer(d.dev, webEpIn, buf[:], 50)
	}
	d.transferMutex.Unlock()
	d.dlogger.Println("webusb - close - rq - done")
}

func (d *WUD) readWrite(buf []byte, endpoint uint8) (int, error) {
	d.dlogger.Println("webusb - rw - start")
	for {
		d.dlogger.Println("webusb - rw - checking closed")
		closed := (atomic.LoadInt32(&d.closed)) == 1
		if closed {
			d.dlogger.Println("webusb - rw - closed, skip")
			return 0, errClosedDevice
		}

		d.dlogger.Println("webusb - rw - lock transfer mutex")
		d.transferMutex.Lock()
		d.dlogger.Println("webusb - rw - actual interrupt transport")
		p, err := usbhid.Interrupt_Transfer(d.dev, endpoint, buf, usbTimeout)
		d.transferMutex.Unlock()
		d.dlogger.Println("webusb - rw - single transfer done")

		if err == nil {
			// sometimes, empty report is read, skip it
			if len(p) > 0 {
				d.dlogger.Println("webusb - rw - single transfer succesful")
				return len(p), err
			}
			d.dlogger.Println("webusb - rw - skipping empty transfer - go again")
		}

		if err != nil {
			d.dlogger.Printf("webusb - rw - error seen - %s", err.Error())
			if err.Error() == usbhid.Error_Name(int(usbhid.ERROR_IO)) ||
				err.Error() == usbhid.Error_Name(int(usbhid.ERROR_NO_DEVICE)) {
				d.dlogger.Println("webusb - rw - device probably disconnected")
				return 0, errDisconnect
			}

			if err.Error() != usbhid.Error_Name(int(usbhid.ERROR_TIMEOUT)) {
				d.dlogger.Println("webusb - rw - other error")
				return 0, err
			}
			d.dlogger.Println("webusb - rw - timeout - go again")
		}

		// continue the for cycle
	}
}

func (d *WUD) Write(buf []byte) (int, error) {
	d.dlogger.Println("webusb - rw - write start")
	return d.readWrite(buf, webEpOut)
}

func (d *WUD) Read(buf []byte) (int, error) {
	d.dlogger.Println("webusb - rw - read start")
	return d.readWrite(buf, webEpIn)
}
