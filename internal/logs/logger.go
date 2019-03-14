package logs

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
)

type Logger struct {
	Writer io.Writer
	mutex  sync.Mutex
}

func findInternalPrefix() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	file := frame.File
	return strings.TrimSuffix(file, "internal/logs/logger.go")
}

var internalPrefix = findInternalPrefix()

func (l *Logger) WriteString(s string) (int, error) {
	l.Log(s)
	return len(s), nil
}

func (l *Logger) Write(p []byte) (int, error) {
	l.Log(string(p))
	return len(p), nil
}

func (l *Logger) Log(s string) {
	s = strings.TrimSuffix(s, "\n")
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	file := frame.File
	file = strings.TrimPrefix(file, internalPrefix)
	function := frame.Function
	function = strings.TrimPrefix(function, "github.com/trezor/trezord-go/")
	r := fmt.Sprintf("[%s %d %s]", file, frame.Line, function)
	l.println(r + " " + s)
}

func (l *Logger) println(s string) {
	l.mutex.Lock()
	defer func() {
		l.mutex.Unlock()
	}()
	long := []byte(s + "\n")
	_, err := l.Writer.Write(long)
	if err != nil {
		// give up, just print on stdout
		fmt.Println(err)
	}
}
