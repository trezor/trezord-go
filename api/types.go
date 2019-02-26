package api

import (
	"github.com/trezor/trezord-go/core"
)

type DeviceType = core.DeviceType

const (
	TypeT1Hid        DeviceType = core.TypeT1Hid
	TypeT1Webusb     DeviceType = core.TypeT1Webusb
	TypeT1WebusbBoot DeviceType = core.TypeT1WebusbBoot
	TypeT2           DeviceType = core.TypeT2
	TypeT2Boot       DeviceType = core.TypeT2Boot
	TypeEmulator     DeviceType = core.TypeEmulator
)

const (
	VendorT1            = core.VendorT1
	ProductT1Firmware   = core.ProductT1Firmware
	VendorT2            = core.VendorT2
	ProductT2Bootloader = core.ProductT2Bootloader
	ProductT2Firmware   = core.ProductT2Firmware
)

type EnumerateEntry = core.EnumerateEntry
