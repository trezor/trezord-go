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
	// the "callers" int is a little magick-y way to
	// get the "actual" calling function name written
	// it's "magic", because it just goes 4 or 3 functions up the stack
	// because we know where exactly is the logger called.
	// TODO: less magic
	l.logIn(s, 4)
	return len(s), nil
}

func (l *Logger) Write(p []byte) (int, error) {
	l.logIn(string(p), 3)
	return len(p), nil
}

func (l *Logger) logIn(s string, callers int) {
	s = strings.TrimSuffix(s, "\n")
	pc := make([]uintptr, 15)
	// TODO: less magic, see WriteString
	n := runtime.Callers(callers, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	file := frame.File
	file = strings.TrimPrefix(file, internalPrefix)
	function := frame.Function
	function = strings.TrimPrefix(function, "github.com/trezor/trezord-go/")
	r := fmt.Sprintf("[%s %d %s]", file, frame.Line, function)
	l.println(r + " " + s)
}

func (l *Logger) Log(s string) {
	l.logIn(s, 3)
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
