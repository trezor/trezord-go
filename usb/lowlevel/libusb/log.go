package libusb

import (
	"io"
)

import "C"

var writer io.Writer

func SetLogWriter(l io.Writer) {
	writer = l
}

//export goLibusbLog
func goLibusbLog(s *C.char) {
	if writer != nil {
		_, _ = writer.Write([]byte(C.GoString(s)))
	}
}
