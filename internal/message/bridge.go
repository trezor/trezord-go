package message

import (
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"

	"github.com/trezor/trezord-go/types"
)

var ErrMalformedData = errors.New("malformed data")

// FromHex decodes from bridge format
func FromBridge(body []byte, logger io.Writer) (*types.Message, error) {
	if logger == nil {
		logger = ioutil.Discard
	}

	io.WriteString(logger, "decodeString\n")

	if len(body) < 6 {
		io.WriteString(logger, "body too short\n")
		return nil, ErrMalformedData
	}

	kind := binary.BigEndian.Uint16(body[0:2])
	size := binary.BigEndian.Uint32(body[2:6])
	data := body[6:]
	if uint32(len(data)) != size {
		io.WriteString(logger, "wrong data length\n")
		return nil, ErrMalformedData
	}

	if validate(data) != nil {
		io.WriteString(logger, "invalid data\n")
		return nil, ErrMalformedData
	}

	io.WriteString(logger, "returning\n")
	return &types.Message{
		Kind: kind,
		Data: data,
	}, nil
}

func ToBridge(msg *types.Message, logger io.Writer) ([]byte, error) {
	if logger == nil {
		logger = ioutil.Discard
	}

	io.WriteString(logger, "start\n")
	var header [6]byte
	data := msg.Data
	kind := msg.Kind
	size := uint32(len(msg.Data))

	binary.BigEndian.PutUint16(header[0:2], kind)
	binary.BigEndian.PutUint32(header[2:6], size)

	res := append(header[:], data...)

	return res, nil
}
