// +build linux freebsd

// shim for linux and freebsd so that ../hidapi.go builds

package hidapi

type HidDeviceInfo struct {
	VendorID  uint16 // Device Vendor ID
	ProductID uint16 // Device Product ID
	Interface int
	UsagePage uint16 // Usage Page for this Device/Interface (Windows/Mac only)
	Path      string // Platform-specific device path
}

type HidDevice struct {
	HidDeviceInfo // Embed the infos for easier access
}

func HidEnumerate(vendorID uint16, productID uint16) []HidDeviceInfo {
	panic("not implemented for linux and freebsd")
}

func (info HidDeviceInfo) Open() (*HidDevice, error) {
	panic("not implemented for linux and freebsd")
}

func (dev *HidDevice) Close() error {
	panic("not implemented for linux and freebsd")
}

func (dev *HidDevice) Write(b []byte, prepend bool) (int, error) {
	panic("not implemented for linux and freebsd")
}

func (dev *HidDevice) Read(b []byte, milliseconds int) (int, error) {
	panic("not implemented for linux and freebsd")
}
