//go:build !debug
// +build !debug

package core

func IsDebugBinary() bool {
	return false
}
