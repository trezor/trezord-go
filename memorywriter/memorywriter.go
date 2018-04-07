package memorywriter

import (
	"bytes"
	"compress/gzip"
	"errors"
)

// to prevent possible memory issues
const maxLineLength = 500

type MemoryWriter struct {
	maxLineCount int
	lines        [][]byte // lines include newlines
}

func (m *MemoryWriter) Write(p []byte) (int, error) {
	if len(p) > maxLineLength {
		return 0, errors.New("Input too long")
	}
	for len(m.lines) >= m.maxLineCount {
		m.lines = m.lines[1:]
	}

	newline := make([]byte, len(p))
	copy(newline, p)
	m.lines = append(m.lines, newline)

	return len(p), nil
}

func (t *MemoryWriter) String(start string) string {
	res := make([]byte, 0)

	for i := len(t.lines) - 1; i >= 0; i-- {
		line := t.lines[i]
		res = append(res, line...)
	}

	return start + string(res)
}

func New(size int) *MemoryWriter {
	return &MemoryWriter{
		maxLineCount: size,
		lines:        make([][]byte, 0, size),
	}
}

func (t *MemoryWriter) gzip(start string) ([]byte, error) {
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	gw.Name = "log.txt"
	gw.Write([]byte(start))

	for i := len(t.lines) - 1; i >= 0; i-- {
		line := t.lines[i]
		gw.Write(line)
	}

	err = gw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), err
}

func (t *MemoryWriter) GzipJsArray(start string) ([]int, error) {
	zip, err := t.gzip(start)
	if err != nil {
		return nil, err
	}
	res := make([]int, 0, len(zip))

	for _, b := range zip {
		res = append(res, int(b))
	}
	return res, nil
}
