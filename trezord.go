package main

import (
	"flag"
	"log"
	"trezord-go/server"
	"trezord-go/wire"
)

func main() {
	var (
		debug = flag.Int("d", 3, "Debug level for libusb.")
		vid   = flag.Int("v", 0x534c, "USB vendor ID.")
		pid   = flag.Int("p", 0x0001, "USB product ID.")
	)
	flag.Parse()

	b, err := wire.Init(
		uint16(*vid),
		uint16(*pid),
		*debug,
	)
	if err != nil {
		log.Fatalf("bus: %s", err)
	}

	s, err := server.New(b)
	if err != nil {
		log.Fatalf("https: %s", err)
	}
	err = s.Run()
	if err != nil {
		log.Fatalf("https: %s", err)
	}
}
