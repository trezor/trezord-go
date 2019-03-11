package main

import (
	"fmt"
	"io"

	"github.com/trezor/trezord-go/api"
	"github.com/trezor/trezord-go/internal/logs"
	"github.com/trezor/trezord-go/internal/server"
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
		stderrLogger.Fatalf("api: %s", err)
	}
	defer a.Close()

	err = initServer(a, stderrWriter, shortMemoryWriter, longMemoryWriter)
	if err != nil {
		stderrLogger.Fatal(err)
	}

	stderrLogger.Print("trezord ended successfully")
}

func initAPI(myOpts initOptions, mw io.Writer) (*api.API, error) {
	apiOpts := make([]api.InitOption, 0, 3+len(myOpts.ports)+len(myOpts.touples))
	apiOpts = append(apiOpts, api.WithUSB(myOpts.withusb))
	apiOpts = append(apiOpts, api.ResetDeviceOnAcquire(myOpts.reset))
	apiOpts = append(apiOpts, api.LogWriter(mw))
	for _, t := range myOpts.ports {
		apiOpts = append(apiOpts, api.AddUDPPort(t))
	}
	for _, t := range myOpts.touples {
		apiOpts = append(apiOpts, api.AddUDPTouple(t.Normal, t.Debug))
	}

	// disable bridge - bridge cannot call itself :)
	apiOpts = append(apiOpts, api.DisableBridge())

	return api.New(apiOpts...)
}

func initServer(
	a *api.API,
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
