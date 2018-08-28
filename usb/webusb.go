package usb

import (
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/trezor/trezord-go/usb/lowlevel"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
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
	usb lowlevel.Context
	mw  *memorywriter.MemoryWriter
}

func InitWebUSB(mw *memorywriter.MemoryWriter) (*WebUSB, error) {
	var usb lowlevel.Context
	mw.Println("webusb - init")
	lowlevel.SetLogWriter(mw)

	err := lowlevel.Init(&usb)
	if err != nil {
		return nil, err
	}

	mw.Println("webusb - init done")

	return &WebUSB{
		usb: usb,
		mw:  mw,
	}, nil
}

func (b *WebUSB) Close() {
	b.mw.Println("webusb - all close (should happen only on exit)")
	lowlevel.Exit(b.usb)
}

func (b *WebUSB) Enumerate() ([]core.USBInfo, error) {
	b.mw.Println("webusb - enum - low level enumerating")
	list, err := lowlevel.Get_Device_List(b.usb)

	if err != nil {
		return nil, err
	}
	b.mw.Println("webusb - enum - low level enumerating done")

	defer func() {
		b.mw.Println("webusb - enum - freeing device list")
		lowlevel.Free_Device_List(list, 1) // unlink devices
		b.mw.Println("webusb - enum - freeing device list done")
	}()

	var infos []core.USBInfo

	// There is a bug in libusb that makes
	// device appear twice with the same path.
	// This is already fixed in libusb 2.0.12;
	// however, 2.0.12 has other problems with windows, so we
	// patchfix it here
	paths := make(map[string]bool)

	for _, dev := range list {
		if b.match(dev) {
			b.mw.Println("webusb - enum - getting device descriptor")
			dd, err := lowlevel.Get_Device_Descriptor(dev)
			if err != nil {
				b.mw.Println("webusb - enum - error getting device descriptor " + err.Error())
				continue
			}
			path := b.identify(dev)
			inset := paths[path]
			if !inset {
				infos = append(infos, core.USBInfo{
					Path:      path,
					VendorID:  int(dd.IdVendor),
					ProductID: int(dd.IdProduct),
				})
				paths[path] = true
			}
		}
	}
	return infos, nil
}

func (b *WebUSB) Has(path string) bool {
	return strings.HasPrefix(path, webusbPrefix)
}

func (b *WebUSB) Connect(path string) (core.USBDevice, error) {
	b.mw.Println("webusb - connect - low level enumerating")
	list, err := lowlevel.Get_Device_List(b.usb)

	if err != nil {
		return nil, err
	}
	b.mw.Println("webusb - connect - low level enumerating done")

	defer func() {
		b.mw.Println("webusb - connect - freeing device list")
		lowlevel.Free_Device_List(list, 1) // unlink devices
		b.mw.Println("webusb - connect - freeing device list done")
	}()

	// There is a bug in libusb that makes
	// device appear twice with the same path.
	// This is already fixed in libusb 2.0.12;
	// however, 2.0.12 has other problems with windows, so we
	// patchfix it here
	mydevs := make([]lowlevel.Device, 0)
	for _, dev := range list {
		if b.match(dev) && b.identify(dev) == path {
			mydevs = append(mydevs, dev)
		}
	}

	err = ErrNotFound
	for _, dev := range mydevs {
		res, errConn := b.connect(dev)
		if errConn == nil {
			return res, nil
		}
		err = errConn
	}
	return nil, err
}

func (b *WebUSB) connect(dev lowlevel.Device) (*WUD, error) {
	b.mw.Println("webusb - connect - low level")
	d, err := lowlevel.Open(dev)
	if err != nil {
		return nil, err
	}
	b.mw.Println("webusb - connect - reset")
	err = lowlevel.Reset_Device(d)
	if err != nil {
		// don't abort if reset fails
		// lowlevel.Close(d)
		// return nil, err
		b.mw.Println(fmt.Sprintf("Warning: error at device reset: %s", err))
	}

	currConf, err := lowlevel.Get_Configuration(d)
	if err != nil {
		b.mw.Println(fmt.Sprintf("webusb - connect - current configuration err %s", err.Error()))
	} else {
		b.mw.Println(fmt.Sprintf("webusb - connect - current configuration %d", currConf))
	}

	b.mw.Println("webusb - connect - set_configuration")
	err = lowlevel.Set_Configuration(d, webConfigNum)
	if err != nil {
		// don't abort if set configuration fails
		// lowlevel.Close(d)
		// return nil, err
		b.mw.Println(fmt.Sprintf("Warning: error at configuration set: %s", err))
	}

	currConf, err = lowlevel.Get_Configuration(d)
	if err != nil {
		b.mw.Println(fmt.Sprintf("webusb - connect - current configuration err %s", err.Error()))
	} else {
		b.mw.Println(fmt.Sprintf("webusb - connect - current configuration %d", currConf))
	}

	b.mw.Println("webusb - connect - claiming interface")
	err = lowlevel.Claim_Interface(d, webIfaceNum)
	if err != nil {
		b.mw.Println("webusb - connect - claiming interface failed")
		lowlevel.Close(d)
		return nil, err
	}

	b.mw.Println("webusb - connect - claiming interface done")

	return &WUD{
		dev:    d,
		closed: 0,

		mw: b.mw,
	}, nil
}

func (b *WebUSB) match(dev lowlevel.Device) bool {
	dd, err := lowlevel.Get_Device_Descriptor(dev)
	if err != nil {
		b.mw.Println("webusb - match - error getting descriptor -" + err.Error())
		return false
	}

	vid := dd.IdVendor
	pid := dd.IdProduct
	if !b.matchVidPid(vid, pid) {
		return false
	}

	c, err := lowlevel.Get_Active_Config_Descriptor(dev)
	if err != nil {
		b.mw.Println("webusb - match - error getting config descriptor " + err.Error())
		return false
	}
	if runtime.GOOS != "freebsd" {
		return (c.BNumInterfaces > webIfaceNum &&
			c.Interface[webIfaceNum].Num_altsetting > webAltSetting &&
			c.Interface[webIfaceNum].Altsetting[webAltSetting].BInterfaceClass == lowlevel.CLASS_VENDOR_SPEC)
	} else {
		return (c.BNumInterfaces > webIfaceNum &&
			c.Interface[webIfaceNum].Num_altsetting > webAltSetting)
	}
}

func (b *WebUSB) matchVidPid(vid uint16, pid uint16) bool {
	trezor1 := vid == core.VendorT1 && (pid == core.ProductT1Firmware)
	trezor2 := vid == core.VendorT2 && (pid == core.ProductT2Firmware || pid == core.ProductT2Bootloader)
	return trezor1 || trezor2
}

func (b *WebUSB) identify(dev lowlevel.Device) string {
	var ports [8]byte
	p, err := lowlevel.Get_Port_Numbers(dev, ports[:])
	if err != nil {
		b.mw.Println(fmt.Sprintf("webusb - identify - error getting port numbers %s", err.Error()))
		return ""
	}
	return webusbPrefix + hex.EncodeToString(p)
}

type WUD struct {
	dev lowlevel.Device_Handle

	closed        int32 // atomic
	transferMutex sync.Mutex
	// closing cannot happen while interrupt_transfer is hapenning,
	// otherwise interrupt_transfer hangs forever

	mw *memorywriter.MemoryWriter
}

func (d *WUD) Close() error {
	d.mw.Println("webusb - close - storing d.closed")
	atomic.StoreInt32(&d.closed, 1)

	d.mw.Println("webusb - close - finishing read queue")
	d.finishReadQueue()

	d.mw.Println("webusb - close - wait for transferMutex lock")
	d.transferMutex.Lock()
	d.mw.Println("webusb - close - low level close")
	lowlevel.Close(d.dev)
	d.transferMutex.Unlock()

	d.mw.Println("webusb - close - done")

	return nil
}

func (d *WUD) finishReadQueue() {
	d.mw.Println("webusb - close - rq - wait for transfermutex lock")
	d.transferMutex.Lock()
	var err error
	var buf [64]byte

	for err == nil {
		d.mw.Println("webusb - close - rq - transfer")
		_, err = lowlevel.Interrupt_Transfer(d.dev, webEpIn, buf[:], 50)
	}
	d.transferMutex.Unlock()
	d.mw.Println("webusb - close - rq - done")
}

func (d *WUD) readWrite(buf []byte, endpoint uint8) (int, error) {
	d.mw.Println("webusb - rw - start")
	for {
		d.mw.Println("webusb - rw - checking closed")
		closed := (atomic.LoadInt32(&d.closed)) == 1
		if closed {
			d.mw.Println("webusb - rw - closed, skip")
			return 0, errClosedDevice
		}

		d.mw.Println("webusb - rw - lock transfer mutex")
		d.transferMutex.Lock()
		d.mw.Println("webusb - rw - actual interrupt transport")
		p, err := lowlevel.Interrupt_Transfer(d.dev, endpoint, buf, usbTimeout)
		d.transferMutex.Unlock()
		d.mw.Println("webusb - rw - single transfer done")

		if err == nil {
			// sometimes, empty report is read, skip it
			if len(p) > 0 {
				d.mw.Println("webusb - rw - single transfer succesful")
				return len(p), err
			}
			d.mw.Println("webusb - rw - skipping empty transfer - go again")
		}

		if err != nil {
			d.mw.Println(fmt.Sprintf("webusb - rw - error seen - %s", err.Error()))
			if err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_IO)) ||
				err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_NO_DEVICE)) ||
				err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_PIPE)) {
				// according to libusb docs, disconnecting device should cause only
				// LIBUSB_ERROR_NO_DEVICE error, but in real life, it causes also
				// LIBUSB_ERROR_IO and LIBUSB_ERROR_PIPE
				d.mw.Println("webusb - rw - device probably disconnected")
				return 0, errDisconnect
			}

			if err.Error() != lowlevel.Error_Name(int(lowlevel.ERROR_TIMEOUT)) {
				d.mw.Println("webusb - rw - other error")
				return 0, err
			}
			d.mw.Println("webusb - rw - timeout - go again")
		}

		// continue the for cycle
	}
}

func (d *WUD) Write(buf []byte) (int, error) {
	d.mw.Println("webusb - rw - write start")
	return d.readWrite(buf, webEpOut)
}

func (d *WUD) Read(buf []byte) (int, error) {
	d.mw.Println("webusb - rw - read start")
	return d.readWrite(buf, webEpIn)
}
