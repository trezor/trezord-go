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
	)
	flag.Parse()

	b, err := wire.Init(*debug)
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
