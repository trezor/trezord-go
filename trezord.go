package main

import (
	"flag"
	"log"

	"github.com/jpochyla/trezord-go/server"
	"github.com/jpochyla/trezord-go/usb"
)

func main() {
	var (
		debug = flag.Int("d", 3, "Debug level for libusb.")
	)
	flag.Parse()

	w, err := usb.InitWebUSB(*debug)
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
