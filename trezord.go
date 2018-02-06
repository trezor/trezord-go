package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/trezor/trezord-go/server"
	"github.com/trezor/trezord-go/usb"
	"gopkg.in/natefinch/lumberjack.v2"
)

// possibly set as "true" by ldflags
var enableWindowsLogging string

func main() {
	var logfile string
	flag.StringVar(&logfile, "l", "", "Log into a file, rotating after 5MB")
	flag.Parse()

	var logger io.WriteCloser
	if logfile != "" || enableWindowsLogging == "true" {
		var address string

		if logfile != "" {
			address = logfile
		} else {
			address = os.Getenv("APPDATA") + "\\TREZOR Bridge\\trezord.log"
		}

		logger = &lumberjack.Logger{
			Filename:   address,
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
	b := usb.Init(w, h)

	s, err := server.New(b, logger)
	if err != nil {
		log.Fatalf("https: %s", err)
	}
	err = s.Run()
	if err != nil {
		log.Fatalf("https: %s", err)
	}
}
