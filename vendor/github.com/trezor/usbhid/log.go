package usbhid

import (
	"io"
)

import "C"

var writer io.Writer

func SetLogWriter(l io.Writer) {
	writer = l
}

//export goUsbHidLog
func goUsbHidLog(s *C.char) {
	if writer != nil {
		writer.Write([]byte(C.GoString(s)))
	}
}
