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

func main() {
	var logfile string
	flag.StringVar(&logfile, "l", "", "Log into a file, rotating after 5MB")
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
	//h, err := usb.InitHIDAPI()
	//if err != nil {
	//	log.Fatalf("hidapi: %s", err)
	//}
	b := usb.Init(w/*, h*/)

	s, err := server.New(b, logger)
	if err != nil {
		log.Fatalf("https: %s", err)
	}
	err = s.Run()
	if err != nil {
		log.Fatalf("https: %s", err)
	}
}
