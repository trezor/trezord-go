package main

import (
	"flag"
	"strconv"
	"strings"
)

type PortTouple struct {
	Normal int
	Debug  int
}

type udpTouples []PortTouple

func (i *udpTouples) String() string {
	res := ""
	for i, p := range *i {
		if i > 0 {
			res += ","
		}
		res = res + strconv.Itoa(p.Normal) + ":" + strconv.Itoa(p.Debug)
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
	*i = append(*i, PortTouple{
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

type initOptions struct {
	logfile     string
	ports       udpPorts
	touples     udpTouples
	withusb     bool
	verbose     bool
	reset       bool
	versionFlag bool
}

func parseFlags() initOptions {
	var options initOptions
	flag.StringVar(
		&(options.logfile),
		"l",
		"",
		"Log into a file, rotating after 20MB",
	)
	flag.Var(
		&(options.ports),
		"e",
		"Use UDP port for emulator. Can be repeated for more ports. Example: trezord-go -e 21324 -e 21326",
	)
	flag.Var(
		&(options.touples),
		"ed",
		"Use UDP port for emulator with debug link. Can be repeated for more ports. Example: trezord-go -ed 21324:21326",
	)
	flag.BoolVar(
		&(options.withusb),
		"u",
		true,
		"Use USB devices. Can be disabled for testing environments. Example: trezord-go -e 21324 -u=false",
	)
	flag.BoolVar(
		&(options.verbose),
		"v",
		false,
		"Write verbose logs to either stderr or logfile",
	)
	flag.BoolVar(
		&(options.versionFlag),
		"version",
		false,
		"Write version",
	)
	flag.BoolVar(
		&(options.reset),
		"r",
		true,
		"Reset USB device on session acquiring. "+
			"Enabled by default (to prevent wrong device states); "+
			"set to false if you plan to connect to debug link outside of bridge.",
	)
	flag.BoolVar(
		&(options.versionFlag),
		"version",
		false,
		"Write version",
	)
	flag.Parse()
	return options
}
