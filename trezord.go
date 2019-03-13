package main

import (
	"fmt"
	"io"

	"github.com/trezor/trezord-go/internal/logs"
	"github.com/trezor/trezord-go/internal/server"
	"github.com/trezor/trezord-go/trezorapi"
)

const version = "2.0.26"

func main() {
	opts := parseFlags()

	if opts.versionFlag {
		fmt.Printf("trezord version %s", version)
		return
	}

	stderrWriter,
		stderrLogger,
		shortMemoryWriter,
		longMemoryWriter := initLoggers(opts.logfile, opts.verbose)

	stderrLogger.Print("trezord is starting.")

	a, err := initAPI(opts, longMemoryWriter)
	if err != nil {
		stderrLogger.Fatalf("trezorapi: %s", err)
	}
	defer a.Close()

	err = initServer(a, stderrWriter, shortMemoryWriter, longMemoryWriter)
	if err != nil {
		stderrLogger.Fatal(err)
	}

	stderrLogger.Print("trezord ended successfully")
}

func initAPI(myOpts initOptions, mw io.Writer) (*trezorapi.API, error) {
	apiOpts := make([]trezorapi.InitOption, 0, 3+len(myOpts.ports)+len(myOpts.touples))
	apiOpts = append(apiOpts, trezorapi.WithUSB(myOpts.withusb))
	apiOpts = append(apiOpts, trezorapi.ResetDeviceOnAcquire(myOpts.reset))
	apiOpts = append(apiOpts, trezorapi.LogWriter(mw))
	for _, t := range myOpts.ports {
		apiOpts = append(apiOpts, trezorapi.AddUDPPort(t))
	}
	for _, t := range myOpts.touples {
		apiOpts = append(apiOpts, trezorapi.AddUDPTouple(t.Normal, t.Debug))
	}

	// disable bridge - bridge cannot call itself :)
	apiOpts = append(apiOpts, trezorapi.DisableBridge())

	return trezorapi.New(apiOpts...)
}

func initServer(
	a *trezorapi.API,
	stderrWriter io.Writer,
	shortMemoryWriter, longMemoryWriter *logs.MemoryWriter,
) error {
	logger := &logs.Logger{Writer: longMemoryWriter}
	logger.Log("Creating HTTP server")
	s, err := server.New(a, stderrWriter, shortMemoryWriter, longMemoryWriter, version)

	if err != nil {
		return fmt.Errorf("https: %s", err)
	}

	logger.Log("Running HTTP server")
	err = s.Run()
	if err != nil {
		return fmt.Errorf("https: %s", err)
	}
	return nil
}
