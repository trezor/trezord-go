package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/server"
	"github.com/trezor/trezord-go/usb"
	"gopkg.in/natefinch/lumberjack.v2"
)

const version = "2.0.24"

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

func initUsb(init bool, wr *memorywriter.MemoryWriter, sl *log.Logger) []core.USBBus {
	if init {
		wr.Println("Initing webusb")

		w, err := usb.InitWebUSB(wr, useOnlyLibusb(), allowCancel())
		if err != nil {
			sl.Fatalf("webusb: %s", err)
		}
		// defer w.Close()
		// not defering - originally in main, now here, here makes no sense

		if useOnlyLibusb() {
			return []core.USBBus{w}
		}

		wr.Println("Initing hidapi")
		h, err := usb.InitHIDAPI(wr)
		if err != nil {
			sl.Fatalf("hidapi: %s", err)
		}
		return []core.USBBus{w, h}
	}
	return nil
}

func main() {
	var logfile string
	var ports udpPorts
	var withusb bool

	flag.StringVar(&logfile, "l", "", "Log into a file, rotating after 5MB")
	flag.Var(&ports, "e", "Use UDP port for emulator. Can be repeated for more ports. Example: trezord-go -e 21324 -e 21326")
	flag.BoolVar(&withusb, "u", true, "Use USB devices. Can be disabled for testing environments. Example: trezord-go -e 21324 -u=false")
	flag.Parse()

	var stderrWriter io.Writer
	if logfile != "" {
		stderrWriter = &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    5, // megabytes
			MaxBackups: 3,
		}
	} else {
		stderrWriter = os.Stderr
	}

	stderrLogger := log.New(stderrWriter, "", log.LstdFlags)

	shortMemoryWriter := memorywriter.New(2000, 200, false)

	longMemoryWriter := memorywriter.New(90000, 200, true)

	stderrLogger.Print("trezord is starting.")

	bus := initUsb(withusb, longMemoryWriter, stderrLogger)

	longMemoryWriter.Println(fmt.Sprintf("UDP port count - %d", len(ports)))

	if len(ports) > 0 {
		e, errUDP := usb.InitUDP(ports)
		if errUDP != nil {
			panic(errUDP)
		}
		bus = append(bus, e)
	}

	if len(bus) == 0 {
		stderrLogger.Fatalf("No transports enabled")
	}

	b := usb.Init(bus...)
	longMemoryWriter.Println("Creating core")
	c := core.New(b, longMemoryWriter, allowCancel())
	longMemoryWriter.Println("Creating HTTP server")
	s, err := server.New(c, stderrWriter, shortMemoryWriter, longMemoryWriter, version)

	if err != nil {
		stderrLogger.Fatalf("https: %s", err)
	}

	longMemoryWriter.Println("Running HTTP server")
	err = s.Run()
	if err != nil {
		stderrLogger.Fatalf("https: %s", err)
	}

	longMemoryWriter.Println("Main ended successfully")
}

// Does OS allow sync canceling via our custom libusb patches?
func allowCancel() bool {
	return runtime.GOOS != "freebsd"
}

// Does OS use libusb for HID devices?
func useOnlyLibusb() bool {
	return runtime.GOOS == "freebsd" || runtime.GOOS == "linux"
}
