package llog

import (
	"io"
	"log"
	"strings"
)

type llogWriter struct {
	fn      LogFunc
	kv      KV
	filters []func(string) (string, error)
}

// Write implements the io.Writer interface
func (lw *llogWriter) Write(b []byte) (int, error) {
	msg := strings.TrimSpace(string(b))
	for _, fn := range lw.filters {
		m, err := fn(msg)
		if err != nil {
			return 0, err
		}
		// ignore if there's no message left
		if m == "" {
			// pretend we still wrote everything
			return len(b), nil
		}
		msg = m
	}
	// TODO: what should we do with multi-line messages
	lw.fn(msg, lw.kv)
	return len(b), nil
}

// NewLogger returns an instance of log.Logger that uses llog to log the
// messages under the sent level with the passed KV. Multiple filter functions
// can be passed. If an error is returned from the filter function, it's sent
// to the caller of the Write method. If an empty string is returned then the
// message is ignored.
func NewLogger(lvl Level, kv KV, filters ...func(string) (string, error)) *log.Logger {
	return newErrorLogger(logFuncFromLevel(lvl), kv, filters)
}

func newErrorLogger(fn LogFunc, kv KV, filters []func(string) (string, error)) *log.Logger {
	return log.New(newWriter(fn, kv, filters...), "", 0)
}

// NewWriter returns an io.Writer that uses llog to log the sent writes with the
// sent log level. Multiple filter functions can be passed. If an error is
// returned from the filter function, it's sent to the caller of the Write
// method. If an empty string is returned then the message is ignored.
func NewWriter(lvl Level, kv KV, filters ...func(string) (string, error)) io.Writer {
	return newWriter(logFuncFromLevel(lvl), kv, filters...)
}

func newWriter(fn LogFunc, kv KV, filters ...func(string) (string, error)) io.Writer {
	return &llogWriter{
		fn:      fn,
		kv:      kv,
		filters: filters,
	}
}
