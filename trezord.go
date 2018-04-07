package main

import (
	"flag"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/trezor/trezord-go/memorywriter"
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
	var withusb bool

	flag.StringVar(&logfile, "l", "", "Log into a file, rotating after 5MB")
	flag.Var(&ports, "e", "Use UDP port for emulator. Can be repeated for more ports. Example: trezord-go -e 21324 -e 21326")
	flag.BoolVar(&withusb, "u", true, "Use USB devices. Can be disabled for testing environments.")
	flag.Parse()

	var logger io.Writer
	if logfile != "" {
		logger = &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    5, // megabytes
			MaxBackups: 3,
		}
	} else {
		logger = os.Stderr
	}

	m := memorywriter.New(2000)
	logger = io.MultiWriter(logger, m)

	log.SetOutput(logger)
	log.Println("trezord is starting.")

	var bus []usb.Bus
	if withusb {
		w, err := usb.InitWebUSB()
		if err != nil {
			log.Fatalf("webusb: %s", err)
		}
		h, err := usb.InitHIDAPI()
		if err != nil {
			log.Fatalf("hidapi: %s", err)
		}
		bus = append(bus, w, h)
	}

	if len(ports) > 0 {
		e, errUDP := usb.InitUDP(ports)
		if errUDP != nil {
			log.Fatalf("emulator: %s", errUDP)
		}
		bus = append(bus, e)
	}

	if len(bus) == 0 {
		log.Fatalf("No transports enabled")
	}

	b := usb.Init(bus...)
	s, err := server.New(b, logger, m)
	if err != nil {
		log.Fatalf("https: %s", err)
	}
	err = s.Run()
	if err != nil {
		log.Fatalf("https: %s", err)
	}
}
