// +build !windows

package server

import (
	"log"
)

func devconInfo(dlogger *log.Logger) (string, error) {
	return "", nil
}

func devconAllStatusInfo() (string, error) {
	return "", nil
}

func runMsinfo() (string, error) {
	return "", nil
}
