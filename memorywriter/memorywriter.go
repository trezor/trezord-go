package memorywriter

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
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
}

func (m *MemoryWriter) Println(s string) {
	long := []byte(s + "\n")
	m.Write(long)
}

// Writer remembers lines in memory
func (m *MemoryWriter) Write(p []byte) (int, error) {
	if len(p) > maxLineLength {
		return 0, errors.New("Input too long")
	}
	newline := make([]byte, len(p))
	copy(newline, p)

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
func (t *MemoryWriter) writeTo(start string, w io.Writer) error {
	_, err := w.Write([]byte(start))
	if err != nil {
		return err
	}

	// Write end lines (latest on up)
	for i := len(t.lines) - 1; i >= 0; i-- {
		line := t.lines[i]
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
	for i := len(t.startLines) - 1; i >= 0; i-- {
		line := t.startLines[i]
		_, err = w.Write(line)
		if err != nil {
			return err
		}
	}

	return nil
}

// Export as string
func (t *MemoryWriter) String(start string) (string, error) {
	var b bytes.Buffer
	err := t.writeTo(start, &b)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

// Export as GZip bytes
func (t *MemoryWriter) Gzip(start string) ([]byte, error) {
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	gw.Name = "log.txt"
	err = t.writeTo(start, gw)
	if err != nil {
		return nil, err
	}

	err = gw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func New(size int, startSize int) *MemoryWriter {
	return &MemoryWriter{
		maxLineCount: size,
		lines:        make([][]byte, 0, size),
		startCount:   startSize,
		startLines:   make([][]byte, 0, startSize),
	}
}
