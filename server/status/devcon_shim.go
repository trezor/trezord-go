// +build !windows

package status

import (
	"github.com/trezor/trezord-go/memorywriter"
)

// Devcon is a tool for listing devices and drivers on windows
// These are empty functions that get called on *nix systems

func devconInfo(d *memorywriter.MemoryWriter) (string, error) {
	return "", nil
}

func devconAllStatusInfo() (string, error) {
	return "", nil
}

func runMsinfo() (string, error) {
	return "", nil
}

func isWindows() bool {
	return false
}

func libwdiReinstallLog() (string, error) {
	return "", nil
}
