package message

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	types "github.com/trezor/trezord-go/trezorapi/trezortypes"
)

var ErrMalformedData = errors.New("malformed data")

func ioWriteStringCheck(logger io.Writer, str string) {
	_, wErr := io.WriteString(logger, str)
	if wErr != nil {
		// ?? not important to return
		fmt.Println(wErr)
	}
}

func FromBridgeFormat(body []byte, logger io.Writer) (*types.Message, error) {
	if logger == nil {
		logger = ioutil.Discard
	}

	ioWriteStringCheck(logger, "decodeString\n")

	if len(body) < 6 {
		ioWriteStringCheck(logger, "body too short\n")
		return nil, ErrMalformedData
	}

	kind := binary.BigEndian.Uint16(body[0:2])
	size := binary.BigEndian.Uint32(body[2:6])
	data := body[6:]
	if uint32(len(data)) != size {
		ioWriteStringCheck(logger, "wrong data length\n")
		return nil, ErrMalformedData
	}

	if validate(data) != nil {
		ioWriteStringCheck(logger, "invalid data\n")

		return nil, ErrMalformedData
	}

	ioWriteStringCheck(logger, "returning\n")

	return &types.Message{
		Kind: kind,
		Data: data,
	}, nil
}

func ToBridgeFormat(msg *types.Message, logger io.Writer) ([]byte, error) {
	if logger == nil {
		logger = ioutil.Discard
	}

	ioWriteStringCheck(logger, "start\n")
	var header [6]byte
	data := msg.Data
	kind := msg.Kind
	size := uint32(len(msg.Data))

	binary.BigEndian.PutUint16(header[0:2], kind)
	binary.BigEndian.PutUint32(header[2:6], size)

	res := append(header[:], data...)

	return res, nil
}
