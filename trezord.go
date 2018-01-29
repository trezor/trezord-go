package main

import (
	"flag"
	"log"

	"github.com/trezor/trezord-go/server"
	"github.com/trezor/trezord-go/usb"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	var (
		path = flag.String("l", "", "Log into a file, rotating after 5MB")
	)
	flag.Parse()

	if *path != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   *path,
			MaxSize:    5, // megabytes
			MaxBackups: 3,
		})
	}

	w, err := usb.InitWebUSB()
	if err != nil {
		log.Fatalf("webusb: %s", err)
	}
	h, err := usb.InitHIDAPI()
	if err != nil {
		log.Fatalf("hidapi: %s", err)
	}
	b := usb.Init(w, h)

	s, err := server.New(b)
	if err != nil {
		log.Fatalf("https: %s", err)
	}
	err = s.Run()
	if err != nil {
		log.Fatalf("https: %s", err)
	}
}
