package types

type DeviceType int

const (
	TypeT1Hid        DeviceType = 0
	TypeT1Webusb     DeviceType = 1
	TypeT1WebusbBoot DeviceType = 2
	TypeT2           DeviceType = 3
	TypeT2Boot       DeviceType = 4
	TypeEmulator     DeviceType = 5
)

const (
	VendorT1            = 0x534c
	ProductT1Firmware   = 0x0001
	VendorT2            = 0x1209
	ProductT2Bootloader = 0x53C0
	ProductT2Firmware   = 0x53C1
)

type EnumerateEntry struct {
	Path    string     `json:"path"`
	Vendor  int        `json:"vendor"`
	Product int        `json:"product"`
	Type    DeviceType `json:"-"`     // used only in status page, not in JSON
	Debug   bool       `json:"debug"` // has debug enabled?

	Session      *string `json:"session"`
	DebugSession *string `json:"debugSession"`
}
