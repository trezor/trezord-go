// Package trezortypes is "kitchen sink" of all different structs,
// that need to be shared across packages
package trezortypes

// DeviceType represents type of device.
//
// HOWEVER, when bridge is used as a backend, we don't
// really know the type of the device, because bridge doesn't say this.
// You need to read that from Initialize{} messages etc,
// if DeviceType is TypeBridgeTransport
type DeviceType int

const (
	// TypeT1Hid is Trezor One, original, FW/BL
	TypeT1Hid DeviceType = 0

	// TypeT1Webusb is Trezor One, WebUSB, FW
	TypeT1Webusb DeviceType = 1

	// TypeT1WebusbBoot is Trezor One, WebUSB, BL
	TypeT1WebusbBoot DeviceType = 2

	// TypeT2 is Trezor T, FW
	TypeT2 DeviceType = 3

	// TypeT2Boot is Trezor T, BL
	TypeT2Boot DeviceType = 4

	// TypeEmulator is any emulator
	TypeEmulator DeviceType = 5

	// TypeBridgeTransport is returned if backend is bridge
	TypeBridgeTransport DeviceType = -1
)

const (
	// VendorT1 is vendor ID used in T1 (only on HID)
	VendorT1 = uint16(0x534c)
	// ProductT1Firmware is product ID used in T1 (only on HID)
	ProductT1Firmware = uint16(0x0001)
	// VendorT2 is vendor ID used in new T1 devices and TT
	VendorT2 = uint16(0x1209)
	// ProductT2Bootloader is product ID used in new T1 and TT, BL
	ProductT2Bootloader = uint16(0x53C0)
	// ProductT2Firmware is product ID used in new T1 and TT, FW
	ProductT2Firmware = uint16(0x53C1)
)

// EnumerateEntry represents device.
type EnumerateEntry struct {
	// Path is always unique on physical reconnect
	Path string `json:"path"`

	// Type used only in status page, not JSON
	// when used with bridge transport, has always -1
	// as bridge does not export device type in JSON response explicitly
	// (for backwards compatibility reasons)
	Type DeviceType `json:"-"`

	// Session that currently uses Trezor. Nil == nobody uses.
	Session *string `json:"session"`
	// DebugSession - session, that currently uses Trezor on debug link
	DebugSession *string `json:"debugSession"`

	// Vendor ID
	Vendor uint16 `json:"vendor"`
	// Product ID
	Product uint16 `json:"product"`

	// Debug signals whether device can be connected in debugLink mode
	Debug bool `json:"debug"` // has debug enabled?
}

// VersionInfo represents version of bridge. Used internally.
type VersionInfo struct {
	// Version, as 2.0.25
	Version string `json:"version"`
}

// SessionInfo represents session. Used internally.
type SessionInfo struct {
	//Session ID
	Session string `json:"session"`
}

// Message from/to trezor, with raw protobuf bytes.
type Message struct {
	// Kind is protobuf message type
	Kind uint16
	// Data is the actual message data
	Data []byte
}
