// Package llog is a generic logging library used by leven labs. The log methods
// come in different severities: Debug, Info, Warn, Error, and Fatal.
//
// The log methods take in a string describing the error, and a set of key/value
// pairs giving the specific context around the error. The string is intended to
// always be the same no matter what, while the key/value pairs give information
// like which userID the error happened to, or any other relevant dynamic
// information.
//
// By default logs will be output to Stdout, without a timestamp attached to
// them, and only showing entries of level Info or above. All of these can be
// configured.
//
// All public functions in this package are thread-safe and can be called at any
// time. The public variables in this package are NOT thread-safe and should
// only be modified before any logging takes place
//
// Examples:
//
//	Info("Something important has occured")
//	Error("Could not open file", llog.KV{"filename": filename, "err": err})
//
package llog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Out is the io.Writer all log entries will be written to. It can be changed to
// anything you like, but the change should happen before any logging occurs. If
// an error occurs while writing to Out the entry will be written to Stdout
// instead
var Out io.Writer = os.Stdout
var defaultOut = os.Stdout

// DisplayTimestamp determines whether or not a timestamp is displayed in the
// log messages. By default one is not displayed. This can be changed by it
// should only be changed before any logging occurs
var DisplayTimestamp bool

// Level describes the severity of a particular log message
type Level int

// All defined log levels
const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	}
	return "unknown level"
}

var currLevel = InfoLevel
var currLevelLock sync.RWMutex

// GetLevel returns the current log level
func GetLevel() Level {
	currLevelLock.RLock()
	defer currLevelLock.RUnlock()
	return currLevel
}

// SetLevel sets the current minimum log level which will be written to Out
func SetLevel(l Level) {
	currLevelLock.Lock()
	defer currLevelLock.Unlock()
	currLevel = l
}

// SetLevelFromString attempts to interpret the given string as a log level and
// sets the current log level to that. If the string can't be interpreted an
// error is returned and the log level remains what it was
func SetLevelFromString(ls string) error {
	switch strings.ToUpper(ls) {
	case "DEBUG":
		SetLevel(DebugLevel)
	case "INFO":
		SetLevel(InfoLevel)
	case "WARN":
		SetLevel(WarnLevel)
	case "ERROR":
		SetLevel(ErrorLevel)
	case "FATAL":
		SetLevel(FatalLevel)
	default:
		return fmt.Errorf("unknown log level %q", ls)
	}

	return nil
}

// KV is used to provide context to a log entry in the form of a dynamic set of
// key/value pairs which can be different for every entry.
type KV map[string]interface{}

type entry struct {
	level   Level
	msg     string
	kv      KV
	blockCh chan struct{} // can be nil
}

var (
	prefix         = []byte("~ ")
	separator      = []byte(" --")
	separatorSpace = append(separator, ' ')
	tsPrefix       = []byte("[")
	tsSuffix       = []byte("] ")
	space          = []byte(" ")
	equals         = []byte("=")
	newline        = []byte("\n")
)

func writeHelper(b []byte, w io.Writer, lastErr error) error {
	if lastErr != nil {
		return lastErr
	}
	_, lastErr = w.Write(b)
	return lastErr
}

func (e entry) printOut(w io.Writer, displayTS bool) error {
	var err error
	err = writeHelper(prefix, w, err)
	if displayTS {
		err = writeHelper(tsPrefix, w, err)
		err = writeHelper([]byte(time.Now().String()), w, err)
		err = writeHelper(tsSuffix, w, err)
	}
	err = writeHelper([]byte(e.level.String()), w, err)
	err = writeHelper(separatorSpace, w, err)
	err = writeHelper([]byte(e.msg), w, err)
	if len(e.kv) > 0 {
		err = writeHelper(separator, w, err)
		for k, v := range e.kv {
			err = writeHelper(space, w, err)
			err = writeHelper([]byte(k), w, err)
			err = writeHelper(equals, w, err)
			err = writeHelper([]byte(fmt.Sprint(v)), w, err)
		}
	}
	err = writeHelper(newline, w, err)

	return err
}

type syncer interface {
	Sync()
}

type flusher interface {
	Flush()
}

var entryCh = make(chan entry)

func init() {
	go func() {
		for e := range entryCh {
			err := e.printOut(Out, DisplayTimestamp)

			// If we couldn't write the entry to Out we write an error to that
			// effect to Stdout, then try to write the original entry as well
			if err != nil && Out != defaultOut {
				erre := entry{
					level: ErrorLevel,
					msg:   "Could not write to error Out",
					kv: KV{
						"err": err,
					},
				}
				erre.printOut(defaultOut, DisplayTimestamp)
				e.printOut(defaultOut, DisplayTimestamp)
			}

			// If the error level is fatal this is the last entry we should ever
			// write. We do want to attempt to flush Out though, in case it's
			// buffered, otherwise exiting now will cause the fatal message to
			// never be shown. We try to cast to either an interface with a Sync
			// or a Flush command as a form of ghetto reflection, to see if the
			// writer has either, and use one if found.
			if e.level == FatalLevel {
				if so, ok := Out.(syncer); ok {
					so.Sync()
				} else if fo, ok := Out.(flusher); ok {
					fo.Flush()
				}
			}

			if e.blockCh != nil {
				close(e.blockCh)
			}
		}
	}()
}

func kvNormalize(kv []KV) KV {
	if len(kv) > 0 {
		return kv[0]
	}
	return nil
}

func logEntry(l Level, msg string, kv []KV, blockCh chan struct{}) {
	if l >= GetLevel() {
		entryCh <- entry{
			level:   l,
			msg:     msg,
			kv:      kvNormalize(kv),
			blockCh: blockCh,
		}
	}
}

// Debug writes a Debug message to Out, with an optional set of key/value pairs
func Debug(msg string, kv ...KV) {
	logEntry(DebugLevel, msg, kv, nil)
}

// Info writes an Info message to Out, with an optional set of key/value pairs
func Info(msg string, kv ...KV) {
	logEntry(InfoLevel, msg, kv, nil)
}

// Warn writes a Warn message to Out, with an optional set of key/value pairs
func Warn(msg string, kv ...KV) {
	logEntry(WarnLevel, msg, kv, nil)
}

// Error writes an Error message to Out, with an optional set of key/value pairs
func Error(msg string, kv ...KV) {
	logEntry(ErrorLevel, msg, kv, nil)
}

// Fatal writes a Fatal message to Out, with an optional set of key/value pairs.
// Once written the process will be exited with an exit code of 1
func Fatal(msg string, kv ...KV) {
	blockCh := make(chan struct{})
	logEntry(FatalLevel, msg, kv, blockCh)
	<-blockCh
	os.Exit(1)
}
