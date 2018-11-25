package usb

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	lowlevel "github.com/trezor/trezord-go/usb/lowlevel/libusb"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
)

const (
	libusbPrefix   = "lib"
	usbConfigNum   = 1
	usbConfigIndex = 0
)

type libusbIfaceData struct {
	number     uint8
	altSetting uint8
	epIn       uint8
	epOut      uint8
}

var normalIface = libusbIfaceData{
	number:     0,
	altSetting: 0,
	epIn:       0x81,
	epOut:      0x01,
}

var debugIface = libusbIfaceData{
	number:     1,
	altSetting: 0,
	epIn:       0x82,
	epOut:      0x02,
}

type device struct {
	vendorID  int
	productID int
	dtype     core.DeviceType
	debug     bool // has debug enabled?

	// There is a bug in libusb that makes
	// device appear twice with the same path.
	// That's why we have an array; one of them will work
	// but we don't know which one
	devices []lowlevel.Device // c pointers
}

func (a *device) equals(b *device) bool {
	for _, adev := range a.devices {
		for _, bdev := range b.devices {
			if adev == bdev {
				return true
			}
		}
	}
	return false
}

type LibUSBDeviceList struct {
	biggestDeviceID int
	lastDevices     map[int](*device) // id => info
}

func (l *LibUSBDeviceList) add(devs map[string](*device)) {
	for _, ndev := range devs {
		add := true
		for _, ldev := range l.lastDevices {
			if ldev.equals(ndev) {
				add = false
			}
		}
		if add {
			newID := l.biggestDeviceID + 1
			l.biggestDeviceID = newID
			l.lastDevices[newID] = ndev
			for _, lldev := range ndev.devices {
				lowlevel.Ref_Device(lldev)
			}
		}
	}
}

func (l *LibUSBDeviceList) cleanup(devs map[string](*device)) {
	for id, ldev := range l.lastDevices {
		discard := true
		for _, ndev := range devs {
			if ldev.equals(ndev) {
				discard = false
			}
		}
		if discard {
			delete(l.lastDevices, id)
			for _, lldev := range ldev.devices {
				lowlevel.Unref_Device(lldev)
			}
		}
	}
}

func (l *LibUSBDeviceList) infos() (res []core.USBInfo) {
	for id, ldev := range l.lastDevices {
		info := core.USBInfo{
			Path:      libusbPrefix + strconv.Itoa(id),
			VendorID:  ldev.vendorID,
			ProductID: ldev.productID,
			Type:      ldev.dtype,
			Debug:     ldev.debug,
		}
		res = append(res, info)
	}
	return
}

type LibUSB struct {
	usb    lowlevel.Context
	mw     *memorywriter.MemoryWriter
	only   bool
	cancel bool
	detach bool

	list *LibUSBDeviceList
}

func InitLibUSB(mw *memorywriter.MemoryWriter, onlyLibusb, allowCancel, detach bool) (*LibUSB, error) {
	var usb lowlevel.Context
	mw.Log("init")
	lowlevel.SetLogWriter(mw)

	err := lowlevel.Init(&usb)
	if err != nil {
		return nil, err
	}

	mw.Log("init done")

	return &LibUSB{
		usb:    usb,
		mw:     mw,
		only:   onlyLibusb,
		cancel: allowCancel,
		detach: detach,
		list: &LibUSBDeviceList{
			lastDevices: make(map[int](*device)),
		},
	}, nil
}

func (b *LibUSB) Close() {
	b.mw.Log("all close (should happen only on exit)")
	lowlevel.Exit(b.usb)
}

func detectDebug(dev lowlevel.Device) (bool, error) {
	config, err := lowlevel.Get_Config_Descriptor(dev, usbConfigIndex)
	if err != nil {
		return false, err
	}

	ifaces := config.Interface
	for _, iface := range ifaces {
		for _, alt := range iface.Altsetting {
			if alt.BInterfaceNumber == debugIface.number &&
				alt.BAlternateSetting == debugIface.altSetting &&
				alt.BNumEndpoints == 2 &&
				alt.BInterfaceClass == lowlevel.CLASS_VENDOR_SPEC &&
				alt.Endpoint[0].BEndpointAddress == debugIface.epIn &&
				alt.Endpoint[1].BEndpointAddress == debugIface.epOut {
				return true, nil
			}
		}
	}
	return false, nil
}

func (b *LibUSB) devicesByPorts(list []lowlevel.Device) (
	devices map[string]*device,
) {
	devices = make(map[string]*device)

	for _, dev := range list {
		m, t := b.match(dev)
		if m {
			b.mw.Log("getting device descriptor")
			dd, err := lowlevel.Get_Device_Descriptor(dev)
			if err != nil {
				b.mw.Log("error getting device descriptor " + err.Error())
				continue
			}
			ports := b.getPortsPath(dev)
			deviceInfo := devices[ports]
			if deviceInfo == nil {
				debug, err := detectDebug(dev)
				if err != nil {
					b.mw.Log("error detecting debug " + err.Error())
					continue
				}
				newDevice := device{
					vendorID:  int(dd.IdVendor),
					productID: int(dd.IdProduct),
					dtype:     t,
					debug:     debug,
					devices:   []lowlevel.Device{dev},
				}
				devices[ports] = &newDevice
			} else {
				deviceInfo.devices = append(deviceInfo.devices, dev)
			}
		}
	}
	return
}

func (b *LibUSB) Enumerate() ([]core.USBInfo, error) {
	b.mw.Log("low level enumerating")
	list, err := lowlevel.Get_Device_List(b.usb)

	if err != nil {
		return nil, err
	}
	b.mw.Log("low level enumerating done")

	defer func() {
		b.mw.Log("freeing device list")
		lowlevel.Free_Device_List(list, 1) // unlink devices
		b.mw.Log("freeing device list done")
	}()

	devs := b.devicesByPorts(list)
	b.list.add(devs)
	b.list.cleanup(devs)
	return b.list.infos(), nil
}

func (b *LibUSB) Has(path string) bool {
	return strings.HasPrefix(path, libusbPrefix)
}

func (b *LibUSB) Connect(path string, debug bool, reset bool) (core.USBDevice, error) {
	b.mw.Log("reenumerating")
	_, err := b.Enumerate()

	if err != nil {
		return nil, err
	}
	b.mw.Log("low level enumerating done")

	id, err := strconv.Atoi(strings.TrimPrefix(path, libusbPrefix))
	if err != nil {
		return nil, err
	}

	if b.list.lastDevices[id] == nil {
		return nil, ErrNotFound
	}

	// This is already fixed in libusb 2.0.12;
	// however, 2.0.12 has other problems with windows, so we
	// patchfix it here
	mydevs := b.list.lastDevices[id].devices

	err = ErrNotFound
	for _, dev := range mydevs {
		res, errConn := b.connect(dev, debug, reset)
		if errConn == nil {
			return res, nil
		}
		err = errConn
	}
	return nil, err
}

func (b *LibUSB) setConfiguration(d lowlevel.Device_Handle) {
	currConf, err := lowlevel.Get_Configuration(d)
	if err != nil {
		b.mw.Log(fmt.Sprintf("current configuration err %s", err.Error()))
	} else {
		b.mw.Log(fmt.Sprintf("current configuration %d", currConf))
	}
	if currConf == usbConfigNum {
		b.mw.Log("not setting config, same")
	} else {
		b.mw.Log("set_configuration")
		err = lowlevel.Set_Configuration(d, usbConfigNum)
		if err != nil {
			// don't abort if set configuration fails
			// lowlevel.Close(d)
			// return nil, err
			b.mw.Log(fmt.Sprintf("Warning: error at configuration set: %s", err))
		}

		currConf, err = lowlevel.Get_Configuration(d)
		if err != nil {
			b.mw.Log(fmt.Sprintf("current configuration err %s", err.Error()))
		} else {
			b.mw.Log(fmt.Sprintf("current configuration %d", currConf))
		}
	}
}

func (b *LibUSB) claimInterface(d lowlevel.Device_Handle, debug bool) (bool, error) {
	attach := false
	usbIfaceNum := int(normalIface.number)
	if debug {
		usbIfaceNum = int(debugIface.number)
	}
	if b.detach {
		b.mw.Log("detecting kernel driver")
		kernel, errD := lowlevel.Kernel_Driver_Active(d, usbIfaceNum)
		if errD != nil {
			b.mw.Log("detecting kernel driver failed")
			lowlevel.Close(d)
			return false, errD
		}
		if kernel {
			attach = true
			b.mw.Log("kernel driver active, detach")
			errD = lowlevel.Detach_Kernel_Driver(d, usbIfaceNum)
			if errD != nil {
				b.mw.Log("detaching kernel driver failed")
				lowlevel.Close(d)
				return false, errD
			}
		}
	}
	b.mw.Log("claiming interface")
	err := lowlevel.Claim_Interface(d, usbIfaceNum)
	if err != nil {
		b.mw.Log("claiming interface failed")
		lowlevel.Close(d)
		return false, err
	}

	b.mw.Log("claiming interface done")

	return attach, nil
}

func (b *LibUSB) connect(dev lowlevel.Device, debug bool, reset bool) (*LibUSBDevice, error) {
	b.mw.Log("low level")
	d, err := lowlevel.Open(dev)
	if err != nil {
		return nil, err
	}
	b.mw.Log("reset")
	if reset {
		err = lowlevel.Reset_Device(d)
		if err != nil {
			// don't abort if reset fails
			// lowlevel.Close(d)
			// return nil, err
			b.mw.Log(fmt.Sprintf("Warning: error at device reset: %s", err))
		}
	}

	b.setConfiguration(d)
	attach, err := b.claimInterface(d, debug)
	if err != nil {
		return nil, err
	}
	return &LibUSBDevice{
		dev:    d,
		closed: 0,

		mw:     b.mw,
		cancel: b.cancel,
		attach: attach,
		debug:  debug,
	}, nil
}

func matchType(dd *lowlevel.Device_Descriptor) core.DeviceType {
	if dd.IdProduct == core.ProductT1Firmware {
		// this is HID, in platforms where we don't use hidapi (linux, bsd)
		return core.TypeT1Hid
	}

	if dd.IdProduct == core.ProductT2Bootloader {
		if int(dd.BcdDevice>>8) == 1 {
			return core.TypeT1WebusbBoot
		}
		return core.TypeT2Boot
	}

	if int(dd.BcdDevice>>8) == 1 {
		return core.TypeT1Webusb
	}

	return core.TypeT2
}

func (b *LibUSB) match(dev lowlevel.Device) (bool, core.DeviceType) {
	b.mw.Log("start")
	dd, err := lowlevel.Get_Device_Descriptor(dev)
	if err != nil {
		b.mw.Log("error getting descriptor -" + err.Error())
		return false, 0
	}

	vid := dd.IdVendor
	pid := dd.IdProduct
	if !b.matchVidPid(vid, pid) {
		b.mw.Log("unmatched")
		return false, 0
	}

	b.mw.Log("matched, get active config")
	c, err := lowlevel.Get_Active_Config_Descriptor(dev)
	if err != nil {
		b.mw.Log("error getting config descriptor " + err.Error())
		return false, 0
	}

	b.mw.Log("let's test")

	var is bool
	usbIfaceNum := normalIface.number
	usbAltSetting := normalIface.altSetting
	if b.only {

		// if we don't use hidapi at all, keep HID devices
		is = (c.BNumInterfaces > usbIfaceNum &&
			c.Interface[usbIfaceNum].Num_altsetting > int(usbAltSetting))

	} else {

		is = (c.BNumInterfaces > usbIfaceNum &&
			c.Interface[usbIfaceNum].Num_altsetting > int(usbAltSetting) &&
			c.Interface[usbIfaceNum].Altsetting[usbAltSetting].BInterfaceClass == lowlevel.CLASS_VENDOR_SPEC)
	}

	if !is {
		b.mw.Log("not matched")
		return false, 0
	}
	b.mw.Log("matched")
	return true, matchType(dd)

}

func (b *LibUSB) matchVidPid(vid uint16, pid uint16) bool {
	// Note: Trezor1 libusb will actually have the T2 vid/pid
	trezor2 := vid == core.VendorT2 && (pid == core.ProductT2Firmware || pid == core.ProductT2Bootloader)

	if b.only {
		trezor1 := vid == core.VendorT1 && (pid == core.ProductT1Firmware)
		return trezor1 || trezor2
	}

	return trezor2
}

func (b *LibUSB) getPortsPath(dev lowlevel.Device) string {
	var ports [8]byte
	p, err := lowlevel.Get_Port_Numbers(dev, ports[:])
	if err != nil {
		b.mw.Log(fmt.Sprintf("error getting port numbers %s", err.Error()))
		return ""
	}
	return hex.EncodeToString(p)
}

type LibUSBDevice struct {
	dev lowlevel.Device_Handle

	closed              int32 // atomic
	normalTransferMutex sync.Mutex
	debugTransferMutex  sync.Mutex
	// two interrupt_transfers should not happen at the same time

	cancel bool
	attach bool
	debug  bool

	mw *memorywriter.MemoryWriter
}

func (d *LibUSBDevice) Close(disconnected bool) error {
	d.mw.Log("storing d.closed")
	atomic.StoreInt32(&d.closed, 1)

	if d.cancel {
		// libusb close does NOT cancel transfers on close
		// => we are using our own function that we added to libusb/sync.c
		// this "unblocks" Interrupt_Transfer in readWrite

		d.mw.Log("canceling previous transfers")
		lowlevel.Cancel_Sync_Transfers_On_Device(d.dev)

		// reading recently disconnected device sometimes causes weird issues
		// => if we *know* it is disconnected, don't finish read queue
		//
		// Finishing read queue is not necessary when we don't allow cancelling
		// (since when we don't allow cancelling, we don't allow session stealing)
		if !disconnected {
			d.mw.Log("finishing read queue")
			d.finishReadQueue(d.debug)
		}
	}

	d.mw.Log("releasing interface")
	iface := int(normalIface.number)
	if d.debug {
		iface = int(debugIface.number)
	}
	err := lowlevel.Release_Interface(d.dev, iface)
	if err != nil {
		// do not throw error, it is just release anyway
		d.mw.Log(fmt.Sprintf("Warning: error at releasing interface: %s", err))
	}

	if d.attach {
		err = lowlevel.Attach_Kernel_Driver(d.dev, iface)
		if err != nil {
			// do not throw error, it is just re-attach anyway
			d.mw.Log(fmt.Sprintf("Warning: error at re-attaching driver: %s", err))
		}
	}

	d.mw.Log("low level close")
	lowlevel.Close(d.dev)
	d.mw.Log("done")

	return nil
}

func (d *LibUSBDevice) transferMutexLock(debug bool) {
	if debug {
		d.debugTransferMutex.Lock()
	} else {
		d.normalTransferMutex.Lock()
	}
}

func (d *LibUSBDevice) transferMutexUnlock(debug bool) {
	if debug {
		d.debugTransferMutex.Unlock()
	} else {
		d.normalTransferMutex.Unlock()
	}
}

func (d *LibUSBDevice) finishReadQueue(debug bool) {
	d.mw.Log("wait for transfermutex lock")
	usbEpIn := normalIface.epIn
	if debug {
		usbEpIn = debugIface.epIn
	}
	d.transferMutexLock(debug)
	var err error
	var buf [64]byte

	for err == nil {
		// these transfers have timeouts => should not interfer with
		// cancel_sync_transfers_on_device
		d.mw.Log("transfer")
		_, err = lowlevel.Interrupt_Transfer(d.dev, usbEpIn, buf[:], 50)
	}
	d.transferMutexUnlock(debug)
	d.mw.Log("done")
}

func (d *LibUSBDevice) readWrite(buf []byte, endpoint uint8) (int, error) {
	d.mw.Log("start")
	for {
		d.mw.Log("checking closed")
		closed := (atomic.LoadInt32(&d.closed)) == 1
		if closed {
			d.mw.Log("closed, skip")
			return 0, errClosedDevice
		}

		d.mw.Log("lock transfer mutex")
		d.transferMutexLock(d.debug)
		d.mw.Log("actual interrupt transport")
		// This has no timeout, but is stopped by Cancel_Sync_Transfers_On_Device
		p, err := lowlevel.Interrupt_Transfer(d.dev, endpoint, buf, 0)
		d.transferMutexUnlock(d.debug)
		d.mw.Log("single transfer done")

		if err != nil {
			d.mw.Log(fmt.Sprintf("error seen - %s", err.Error()))
			if isErrorDisconnect(err) {
				d.mw.Log("device probably disconnected")
				return 0, errDisconnect
			}

			d.mw.Log("other error")
			return 0, err
		}

		// sometimes, empty report is read, skip it
		// TODO: is this still needed with 0 timeouts?
		if len(p) > 0 {
			d.mw.Log("single transfer succesful")
			return len(p), err
		}
		d.mw.Log("skipping empty transfer, go again")
		// continue the for cycle if empty transfer
	}
}

func isErrorDisconnect(err error) bool {
	// according to libusb docs, disconnecting device should cause only
	// LIBUSB_ERROR_NO_DEVICE error, but in real life, it causes also
	// LIBUSB_ERROR_IO, LIBUSB_ERROR_PIPE, LIBUSB_ERROR_OTHER

	return (err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_IO)) ||
		err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_NO_DEVICE)) ||
		err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_OTHER)) ||
		err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_PIPE)))
}

func (d *LibUSBDevice) Write(buf []byte) (int, error) {
	d.mw.Log("write start")
	usbEpOut := normalIface.epOut
	if d.debug {
		usbEpOut = debugIface.epOut
	}
	return d.readWrite(buf, usbEpOut)
}

func (d *LibUSBDevice) Read(buf []byte) (int, error) {
	d.mw.Log("read start")
	usbEpIn := normalIface.epIn
	if d.debug {
		usbEpIn = debugIface.epIn
	}
	return d.readWrite(buf, usbEpIn)
}
