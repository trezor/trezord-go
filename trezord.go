package main

import (
	"flag"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/trezor/trezord-go/server"
	"github.com/trezor/trezord-go/usb"
	"gopkg.in/natefinch/lumberjack.v2"
)

type udpPorts []int

func (i *udpPorts) String() string {
	res := ""
	for i, p := range *i {
		if i > 0 {
			res = res + ","
		}
		res = res + strconv.Itoa(p)
	}
	return res
}

func (i *udpPorts) Set(value string) error {
	p, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*i = append(*i, p)
	return nil
}

func main() {
	var logfile string
	var ports udpPorts

	flag.StringVar(&logfile, "l", "", "Log into a file, rotating after 5MB")
	flag.Var(&ports, "e", "Use UDP port for emulator. Can be repeated for more ports. Example: trezord-go -e 21324 -e 21326")
	flag.Parse()

	var logger io.WriteCloser
	if logfile != "" {
		logger = &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    5, // megabytes
			MaxBackups: 3,
		}
	} else {
		logger = os.Stderr
	}
	log.SetOutput(logger)

	w, err := usb.InitWebUSB()
	if err != nil {
		log.Fatalf("webusb: %s", err)
	}
	h, err := usb.InitHIDAPI()
	if err != nil {
		log.Fatalf("hidapi: %s", err)
	}

	var b *usb.USB

	if len(ports) > 0 {
		e, err := usb.InitUDP(ports)
		if err != nil {
			log.Fatalf("emulator: %s", err)
		}
		b = usb.Init(w, h, e)
	} else {
		b = usb.Init(w, h)
	}

	s, err := server.New(b, logger)
	if err != nil {
		log.Fatalf("https: %s", err)
	}
	err = s.Run()
	if err != nil {
		log.Fatalf("https: %s", err)
	}
}
