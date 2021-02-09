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
//	Info("Something important has occurred")
//	Error("Could not open file", llog.KV{"filename": filename}, llog.ErrKV(err))
//
package llog

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
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

// BlockByDefault controls whether the non-Fatal functions wait for the write
// to Out to complete. This can be useful to set to true for tests so that
// logging doesn't end up mangling test output.
var BlockByDefault = false

// DisplayTimestamp determines whether or not a timestamp is displayed in the
// log messages. By default one is not displayed. This can be changed by it
// should only be changed before any logging occurs
var DisplayTimestamp bool

// Truncate is a helper function to truncate a string to a given size. It will
// add 3 trailing elipses, so the returned string will be at most size+3
// characters long
func Truncate(s string, size int) string {
	if len(s) <= size {
		return s
	}
	return s[:size] + "..."
}

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

func logFuncFromLevel(l Level) LogFunc {
	switch l {
	case DebugLevel:
		return Debug
	case InfoLevel:
		return Info
	case WarnLevel:
		return Warn
	case ErrorLevel:
		return Error
	case FatalLevel:
		return Fatal
	default:
		panic(fmt.Errorf("unknown log level %q", l))
	}
}

// KV is used to provide context to a log entry in the form of a dynamic set of
// key/value pairs which can be different for every entry.
type KV map[string]interface{}

// Copy returns a copy of the KV being called on. This method will never return
// nil
func (kv KV) Copy() KV {
	nkv := make(KV, len(kv))
	for k, v := range kv {
		nkv[k] = v
	}
	return nkv
}

// Merge takes in multiple KVs and returns a single KV which is the union of all
// the passed in ones. Key/vals on the rightmost of the set take precedence over
// conflicting ones to the left. This function will never return nil
func Merge(kvs ...KV) KV {
	kv := make(KV, len(kvs))
	for i := range kvs {
		for k, v := range kvs[i] {
			kv[k] = v
		}
	}
	return kv
}

// Set returns a copy of the KV being called on with the given key/val set on
// it. The original KV is unaffected
func (kv KV) Set(k string, v interface{}) KV {
	nkv := kv.Copy()
	nkv[k] = v
	return nkv
}

// StringSlice converts the KV into a slice of [2]string entries (first index is
// the key, second is the string form of the value).
func (kv KV) StringSlice() [][2]string {
	slice := make([][2]string, 0, len(kv))
	for kstr, v := range kv {
		vstr := fmt.Sprint(v)
		// TODO this is only here because logstash is dumb and doesn't
		// properly handle escaped quotes. Once
		// https://github.com/elastic/logstash/issues/1645
		// gets figured out this Replace can be removed
		vstr = strings.Replace(vstr, `"`, `'`, -1)
		slice = append(slice, [2]string{kstr, vstr})
	}
	sort.Slice(slice, func(i, j int) bool {
		return slice[i][0] < slice[j][0]
	})
	return slice
}

type entry struct {
	blockCh chan struct{} // can be nil
	msg     string
	kvSlice [][2]string
	level   Level
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

func (e entry) printOut(w io.Writer, displayTS bool) error {
	var err error
	write := func(b []byte) {
		if err == nil {
			_, err = w.Write(b)
		}
	}

	write(prefix)
	if displayTS {
		write(tsPrefix)
		write([]byte(time.Now().String()))
		write(tsSuffix)
	}
	write([]byte(e.level.String()))
	write(separatorSpace)
	write([]byte(e.msg))
	if len(e.kvSlice) > 0 {
		write(separator)
		for _, kve := range e.kvSlice {
			write(space)
			write([]byte(kve[0]))
			write(equals)
			write([]byte(strconv.QuoteToASCII(kve[1])))
		}
	}
	write(newline)

	return err
}

type syncer interface {
	Sync()
}

type flusher interface {
	Flush()
}

var entryCh = make(chan entry)
var flushCh = make(chan chan bool)

func init() {
	go func() {
		for {
			select {
			case doneCh := <-flushCh:
				flush()
				close(doneCh)
			case e := <-entryCh:
				err := e.printOut(Out, DisplayTimestamp)

				// If we couldn't write the entry to Out we write an error to that
				// effect to Stdout, then try to write the original entry as well
				if err != nil && Out != defaultOut {
					erre := entry{
						level:   ErrorLevel,
						msg:     "Could not write to error Out",
						kvSlice: ErrKV(err).StringSlice(),
					}
					erre.printOut(defaultOut, DisplayTimestamp)
					e.printOut(defaultOut, DisplayTimestamp)
				}

				// If the error level is fatal this is the last entry we should ever
				// write. We do want to attempt to flush Out though, in case it's
				// buffered, otherwise exiting now will cause the fatal message to
				// never be shown.
				if e.level == FatalLevel {
					flush()
				}

				if e.blockCh != nil {
					close(e.blockCh)
				}
			}
		}
	}()
}

// does a raw flush on Out. Shouldn't be called outside the main loop
func flush() {
	// We try to cast to either an interface with a Sync or a Flush command as a
	// form of ghetto reflection, to see if the writer has either, and use one
	// if found.
	if so, ok := Out.(syncer); ok {
		so.Sync()
	} else if fo, ok := Out.(flusher); ok {
		fo.Flush()
	}
}

func logEntry(l Level, msg string, kvs []KV, block bool) {
	if l < GetLevel() {
		return
	}
	var blockCh chan struct{}
	if block {
		blockCh = make(chan struct{})
		defer func() {
			<-blockCh
		}()
	}
	entryCh <- entry{
		level:   l,
		msg:     msg,
		kvSlice: Merge(kvs...).StringSlice(),
		blockCh: blockCh,
	}
}

// LogFunc is the function signature used by the different log functions (Debug,
// Info, Warn, Error, and Fatal). It's useful for writing wrapper functions
type LogFunc func(string, ...KV)

// Debug writes a Debug message to Out, with an optional set of key/value pairs
// which will be Merge'd together.
func Debug(msg string, kv ...KV) {
	logEntry(DebugLevel, msg, kv, BlockByDefault)
}

// Info writes an Info message to Out, with an optional set of key/value pairs
// which will be Merge'd together.
func Info(msg string, kv ...KV) {
	logEntry(InfoLevel, msg, kv, BlockByDefault)
}

// Warn writes a Warn message to Out, with an optional set of key/value pairs
// which will be Merge'd together.
func Warn(msg string, kv ...KV) {
	logEntry(WarnLevel, msg, kv, BlockByDefault)
}

// Error writes an Error message to Out, with an optional set of key/value pairs
// which will be Merge'd together.
func Error(msg string, kv ...KV) {
	logEntry(ErrorLevel, msg, kv, BlockByDefault)
}

// Fatal writes a Fatal message to Out, with an optional set of key/value pairs
// which will be Merge'd together. Once written the process will be exited with
// an exit code of 1
func Fatal(msg string, kv ...KV) {
	logEntry(FatalLevel, msg, kv, true)
	os.Exit(1)
}

// Flush will attempts to flush any buffered data in Out. Will block until the
// flushing has been completed
func Flush() {
	doneCh := make(chan bool)
	flushCh <- doneCh
	<-doneCh
}
