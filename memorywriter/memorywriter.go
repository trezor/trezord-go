package memorywriter

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
)

// to prevent possible memory issues
const maxLineLength = 500

type MemoryWriter struct {
	maxLineCount int
	lines        [][]byte // lines include newlines
	startCount   int
	startLines   [][]byte
}

func (m *MemoryWriter) Write(p []byte) (int, error) {
	if len(p) > maxLineLength {
		return 0, errors.New("Input too long")
	}
	newline := make([]byte, len(p))
	copy(newline, p)

	if len(m.startLines) < m.startCount {
		m.startLines = append(m.startLines, newline)
	} else {
		for len(m.lines) >= m.maxLineCount {
			m.lines = m.lines[1:]
		}

		m.lines = append(m.lines, newline)
	}
	return len(p), nil
}

func (t *MemoryWriter) writeTo(start string, w io.Writer) error {
	_, err := w.Write([]byte(start))
	if err != nil {
		return err
	}

	for i := len(t.lines) - 1; i >= 0; i-- {
		line := t.lines[i]
		_, err = w.Write(line)
		if err != nil {
			return err
		}
	}

	_, err = w.Write([]byte("...\n"))
	if err != nil {
		return err
	}

	for i := len(t.startLines) - 1; i >= 0; i-- {
		line := t.startLines[i]
		_, err = w.Write(line)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *MemoryWriter) String(start string) (string, error) {
	var b bytes.Buffer
	err := t.writeTo(start, &b)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func (t *MemoryWriter) gzip(start string) ([]byte, error) {
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

func New(size int, startSize int) *MemoryWriter {
	return &MemoryWriter{
		maxLineCount: size,
		lines:        make([][]byte, 0, size),
		startCount:   startSize,
		startLines:   make([][]byte, 0, startSize),
	}
}
