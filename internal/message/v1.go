package message

import (
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"

	types "github.com/trezor/trezord-go/trezorapi/trezortypes"
)

const (
	repMarker = '?'
	repMagic  = '#'
	packetLen = 64
)

// WriteToDevice encodes message to trezor format
func WriteToDevice(m *types.Message, device io.Writer, logger io.Writer) (int64, error) {
	if logger == nil {
		logger = ioutil.Discard
	}
	ioWriteStringCheck(logger, "start\n")

	var (
		rep  [packetLen]byte
		kind = m.Kind
		size = uint32(len(m.Data))
	)
	// pack header
	rep[0] = repMarker
	rep[1] = repMagic
	rep[2] = repMagic
	binary.BigEndian.PutUint16(rep[3:], kind)
	binary.BigEndian.PutUint32(rep[5:], size)

	ioWriteStringCheck(logger, "actually writing\n")

	var (
		written = 0 // number of written bytes
		offset  = 9 // just after the header
	)
	for written < len(m.Data) {
		n := copy(rep[offset:], m.Data[written:])
		written += n
		offset += n
		if offset >= len(rep) {
			_, err := device.Write(rep[:])
			if err != nil {
				return int64(written), err
			}
			offset = 1 // just after the marker
		}
	}
	if offset != 1 {
		for offset < len(rep) {
			rep[offset] = 0x00
			offset++
		}
		_, err := device.Write(rep[:])
		if err != nil {
			return int64(written), err
		}
	}

	return int64(written), nil
}

var (
	ErrMalformedMessage = errors.New("malformed wire format")
)

// ReadFromDevice decodes message from trezor format to Message
func ReadFromDevice(device io.Reader, logger io.Writer) (*types.Message, error) {
	if logger == nil {
		logger = ioutil.Discard
	}
	ioWriteStringCheck(logger, "start\n")
	var (
		rep  [packetLen]byte
		read = 0 // number of read bytes
	)
	n, err := device.Read(rep[:])
	if err != nil {
		return nil, err
	}

	// skip all the previous messages in the bus
	for rep[0] != repMarker || rep[1] != repMagic || rep[2] != repMagic {
		ioWriteStringCheck(logger, "detected previous message, skipping\n")
		n, err = device.Read(rep[:])
		if err != nil {
			return nil, err
		}
	}
	read += n

	ioWriteStringCheck(logger, "actual reading started\n")

	// parse header
	var (
		kind = binary.BigEndian.Uint16(rep[3:])
		size = binary.BigEndian.Uint32(rep[5:])
		data = make([]byte, 0, size)
	)
	data = append(data, rep[9:]...) // read data after header

	for uint32(len(data)) < size {
		n, err := device.Read(rep[:])
		if err != nil {
			return nil, err
		}
		if rep[0] != repMarker {
			return nil, ErrMalformedMessage
		}
		read += n
		data = append(data, rep[1:]...) // read data after marker
	}
	data = data[:size]

	ioWriteStringCheck(logger, "actual reading finished\n")

	return &types.Message{
		Kind: kind,
		Data: data,
	}, nil
}
