package memorywriter

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"time"
)

// This is a helper package that writes logs to memory,
// rotates the lines, but remembers some lines on the start
// It is useful for detailed logging, that would take too much memory

// to prevent possible memory issues, hardcode max line length
const maxLineLength = 500

type MemoryWriter struct {
	maxLineCount int
	lines        [][]byte // lines include newlines
	startCount   int
	startLines   [][]byte
	startTime    time.Time
	printTime    bool
}

func (m *MemoryWriter) Println(s string) {
	long := []byte(s + "\n")
	_, err := m.Write(long)
	if err != nil {
		// give up, just print on stdout
		fmt.Println(err)
	}
}

// Writer remembers lines in memory
func (m *MemoryWriter) Write(p []byte) (int, error) {
	if len(p) > maxLineLength {
		return 0, errors.New("input too long")
	}

	var newline []byte
	if !m.printTime {
		newline = make([]byte, len(p))
		copy(newline, p)
	} else {
		now := time.Now()
		elapsed := now.Sub(m.startTime)

		elapsedS := fmt.Sprintf("%.6f", elapsed.Seconds())
		nowS := now.Format("15:04:05")

		newline = []byte(fmt.Sprintf("[%s : %s] %s", elapsedS, nowS, string(p)))
	}

	if len(m.startLines) < m.startCount {
		// do not rotate
		m.startLines = append(m.startLines, newline)
	} else {
		// rotate
		for len(m.lines) >= m.maxLineCount {
			m.lines = m.lines[1:]
		}

		m.lines = append(m.lines, newline)
	}
	return len(p), nil
}

// Exports lines to a writer, plus adds additional text on top
// In our case, additional text is devcon exports and trezord version
func (m *MemoryWriter) writeTo(start string, w io.Writer) error {
	_, err := w.Write([]byte(start))
	if err != nil {
		return err
	}

	// Write end lines (latest on up)
	for i := len(m.lines) - 1; i >= 0; i-- {
		line := m.lines[i]
		_, err = w.Write(line)
		if err != nil {
			return err
		}
	}

	// ... to make space between start and end
	_, err = w.Write([]byte("...\n"))
	if err != nil {
		return err
	}

	// Write start lines
	for i := len(m.startLines) - 1; i >= 0; i-- {
		line := m.startLines[i]
		_, err = w.Write(line)
		if err != nil {
			return err
		}
	}

	return nil
}

// String exports as string
func (m *MemoryWriter) String(start string) (string, error) {
	var b bytes.Buffer
	err := m.writeTo(start, &b)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

// Gzip exports as GZip bytes
func (m *MemoryWriter) Gzip(start string) ([]byte, error) {
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	gw.Name = "log.txt"
	err = m.writeTo(start, gw)
	if err != nil {
		return nil, err
	}

	err = gw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func New(size int, startSize int, printTime bool) *MemoryWriter {
	return &MemoryWriter{
		maxLineCount: size,
		lines:        make([][]byte, 0, size),
		startCount:   startSize,
		startLines:   make([][]byte, 0, startSize),
		startTime:    time.Now(),
		printTime:    printTime,
	}
}
