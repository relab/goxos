package elog

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	e "github.com/relab/goxos/elog/event"
)

const (
	flushInterval = 30 * time.Second
	bufferSize    = 1024 * 256
)

var logger eventLogger

type eventLogger struct {
	enabled bool
	mu      sync.Mutex
	*bufio.Writer
	*gob.Encoder
	*os.File
	stopRegFlush chan bool
}

func init() {
	flag.BoolVar(&logger.enabled, "log_events", false, "enable event logging")
	logger.stopRegFlush = make(chan bool)
	go logger.flushRegularly()
}

func (el *eventLogger) init() {
	var err error
	name, symlink := logName()
	el.File, err = os.Create(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "elog: exiting due to error: %s\n", err)
		os.Exit(2)
	}
	os.Remove(symlink)        // ignore err
	os.Symlink(name, symlink) // ignore err
	el.Writer = bufio.NewWriterSize(el.File, bufferSize)
	el.Encoder = gob.NewEncoder(el.Writer)
}

// IsEnabled reports whether the EventLogger is enabled.
func IsEnabled() bool {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	return logger.enabled
}

// Enable enables the EventLogger.
func Enable() {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	logger.enabled = true
}

// Disable disables the EventLogger.
func Disable() {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	logger.enabled = false
}

// Log logs event e if the EventLogger is enabled.
func Log(e e.Event) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if logger.enabled {
		if logger.Encoder == nil {
			logger.init()
		}
		logger.Encoder.Encode(e)
	}
}

// Flush flushes all pending events to file.
func Flush() {
	logger.flush()
}

func (el *eventLogger) flushRegularly() {
	for {
		select {
		case <-time.After(flushInterval):
			el.flush()
		case <-el.stopRegFlush:
			break
		}
	}
}

func (el *eventLogger) flush() {
	if el.Encoder != nil {
		el.Writer.Flush()
		el.File.Sync()
	}
}
