package usb

import (
	"bytes"
	"net"
)

const (
	emulatorPath    = "emulator"
	emulatorNetwork = "udp"
	emulatorAddress = "127.0.0.1:21324"
)

var (
	emulatorPing = []byte("PINGPING")
	emulatorPong = []byte("PONGPONG")
)

type Emulator struct {
}

func InitEmulator() (*Emulator, error) {
	return &Emulator{}, nil
}

func (b *Emulator) Enumerate() ([]Info, error) {
	var infos []Info

	if b.hasEmulator() {
		infos = append(infos, Info{
			Path:      emulatorPath,
			VendorID:  0,
			ProductID: 0,
		})
	}
	return infos, nil
}

func (b *Emulator) Has(path string) bool {
	return path == emulatorPath
}

func (b *Emulator) hasEmulator() bool {
	dev, err := b.Connect(emulatorPath)
	if err != nil {
		return false
	}

	_, err = dev.Write(emulatorPing)
	if err != nil {
		return false
	}

	response := make([]byte, len(emulatorPong))

	_, err = dev.Read(response)
	if err != nil {
		return false
	}

	if !bytes.Equal(response, emulatorPong) {
		return false
	}

	return true
}

func (b *Emulator) Connect(path string) (Device, error) {
	return net.Dial(emulatorNetwork, emulatorAddress)
}
