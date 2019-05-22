// +build darwin,!ios,cgo windows,cgo

package hidapi

import (
	"fmt"
	"io"
)

import "C"

var writer io.Writer

func SetLogWriter(l io.Writer) {
	writer = l
}

//export goHidLog
func goHidLog(s *C.char) {
	if writer != nil {
		_, err := writer.Write([]byte(C.GoString(s)))
		if err != nil {
			// whatever, just log it out
			fmt.Println(err)
		}
	}
}
