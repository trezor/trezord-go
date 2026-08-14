package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/trezor/trezord-go/bridge"
	"github.com/trezor/trezord-go/usb"
)

type udpTouples []usb.PortTouple

func (i *udpTouples) String() string {
	res := ""
	for i, p := range *i {
		if i > 0 {
			res += ","
		}
		res += strconv.Itoa(p.Normal) + ":" + strconv.Itoa(p.Debug)
	}
	return res
}

func (i *udpTouples) Set(value string) error {
	split := strings.Split(value, ":")
	n, err := strconv.Atoi(split[0])
	if err != nil {
		return err
	}
	d, err := strconv.Atoi(split[1])
	if err != nil {
		return err
	}
	*i = append(*i, usb.PortTouple{
		Normal: n,
		Debug:  d,
	})
	return nil
}

type udpPorts []int

func (i *udpPorts) String() string {
	res := ""
	for i, p := range *i {
		if i > 0 {
			res += ","
		}
		res += strconv.Itoa(p)
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
	var port int
	var ports udpPorts
	var touples udpTouples
	var withusb bool
	var verbose bool
	var reset bool
	var versionFlag bool

	flag.StringVar(
		&logfile,
		"l",
		"",
		"Log into a file, rotating after 20MB",
	)
	flag.IntVar(
		&port,
		"p",
		21325,
		"Use a different port for the HTTP server. Default is 21325.",
	)
	flag.Var(
		&ports,
		"e",
		"Use UDP port for emulator. Can be repeated for more ports. Example: trezord-go -e 21324 -e 21326",
	)
	flag.Var(
		&touples,
		"ed",
		"Use UDP port for emulator with debug link. Can be repeated for more ports. Example: trezord-go -ed 21324:21326",
	)
	flag.BoolVar(
		&withusb,
		"u",
		true,
		"Use USB devices. Can be disabled for testing environments. Example: trezord-go -e 21324 -u=false",
	)
	flag.BoolVar(
		&verbose,
		"v",
		false,
		"Write verbose logs to either stderr or logfile",
	)
	flag.BoolVar(
		&versionFlag,
		"version",
		false,
		"Write version",
	)
	flag.BoolVar(
		&reset,
		"r",
		true,
		"Reset USB device on session acquiring. Enabled by default (to prevent wrong device states); set to false if you plan to connect to debug link outside of bridge.",
	)
	flag.Parse()

	githash := bridge.ResolveGitHash()

	if versionFlag {
		fmt.Printf("trezord version %s (rev %s)", bridge.Version, githash)
		return
	}

	svc, err := bridge.Start(bridge.Options{
		Port:     port,
		Logfile:  logfile,
		Verbose:  verbose,
		WithUSB:  withusb,
		Reset:    reset,
		UDPPorts: ports,
		UDPPairs: touples,
		Version:  bridge.Version,
		GitHash:  githash,
	})
	if err != nil {
		log.Fatalf("%v", err)
	}

	err = svc.Wait()
	if err != nil {
		log.Fatalf("https: %s", err)
	}
}
