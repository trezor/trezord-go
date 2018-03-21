package memorywriter

import (
	"errors"
)

// to prevent possible memory issues
const maxLineLength = 500

type MemoryWriter struct {
	maxLineCount int
	lines        [][]byte // lines include newlines
}

func (m *MemoryWriter) Write(p []byte) (n int, err error) {
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

func (t *MemoryWriter) String() string {
	res := make([]byte, 0)

	for i := len(t.lines) - 1; i >= 0; i-- {
		line := t.lines[i]
		res = append(res, line...)
	}

	return string(res)
}

func New(size int) *MemoryWriter {
	return &MemoryWriter{
		maxLineCount: size,
		lines:        make([][]byte, 0, size),
	}
}
