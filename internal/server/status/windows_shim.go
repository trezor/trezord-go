// +build !windows

package status

import (
	"github.com/trezor/trezord-go/internal/logs"
)

// Devcon is a tool for listing devices and drivers on windows
// These are empty functions that get called on *nix systems

func devconInfo(d *logs.Logger) (string, error) {
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

func oldLog() (string, error) {
	return "", nil
}

func libwdiReinstallLog() (string, error) {
	return "", nil
}

func setupAPIDevLog() (string, error) {
	return "", nil
}
