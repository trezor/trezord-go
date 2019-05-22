package main

import (
	"io"
	"log"
	"os"

	"github.com/trezor/trezord-go/internal/logs"
	"gopkg.in/natefinch/lumberjack.v2"
)

func initLoggers(logfile string, verbose bool) (
	stderrWriter io.Writer, // where we write short messages to stderr (on windows to file)
	stderrLogger *log.Logger, // logger for stderrWriter
	shortMemoryWriter *logs.MemoryWriter, // what we write to status page
	longMemoryWriter *logs.MemoryWriter, // what we write to detailed status file
) {
	if logfile != "" {
		stderrWriter = &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    20, // megabytes
			MaxBackups: 3,
		}
	} else {
		stderrWriter = os.Stderr
	}

	stderrLogger = log.New(stderrWriter, "", log.LstdFlags)
	shortMemoryWriter, err := logs.NewMemoryWriter(2000, 200, false, nil)
	if err != nil {
		stderrLogger.Fatalf("writer: %s", err)
	}

	verboseWriter := stderrWriter
	if !verbose {
		verboseWriter = nil
	}

	longMemoryWriter, err = logs.NewMemoryWriter(90000, 200, true, verboseWriter)
	if err != nil {
		stderrLogger.Fatalf("writer: %s", err)
	}
	return stderrWriter, stderrLogger, shortMemoryWriter, longMemoryWriter
}
