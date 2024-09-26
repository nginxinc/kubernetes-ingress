package glog

// Custom log levels https://go.dev/src/log/slog/example_custom_levels_test.go - for fatal & trace

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const (
	// LevelTrace - Trace Level Logging same as glog.V(3)
	LevelTrace = slog.Level(-8)
	// LevelDebug - Debug Level Logging same as glog.V(2)
	LevelDebug = slog.LevelDebug
	// LevelInfo - Info Level Logging same as glog.Info()
	LevelInfo = slog.LevelInfo
	// LevelWarning - Warn Level Logging same as glog.Warning()
	LevelWarning = slog.LevelWarn
	// LevelError - Error Level Logging same as glog.Error()
	LevelError = slog.LevelError
	// LevelFatal - Fatal Level Logging same as glog.Fatal()
	LevelFatal = slog.Level(12)
)

// Handler holds all the parameters for the handler
type Handler struct {
	opts Options
	mu   *sync.Mutex
	out  io.Writer
}

// Options contains the log Level
type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
}

// New - create a new Handler
func New(out io.Writer, opts *Options) *Handler {
	h := &Handler{out: out, mu: &sync.Mutex{}}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	return h
}

// Enabled - is this log level enabled?
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// WithGroup - not needed
func (h *Handler) WithGroup(_ string) slog.Handler {
	// not needed.
	return h
}

// WithAttrs - not needed
func (h *Handler) WithAttrs(_ []slog.Attr) slog.Handler {
	// not needed.
	return h
}

// Handle log event
// Format F20240920 16:53:18.817844   70741 main.go:285] message
//
//	<Level>YYYYMMDD HH:MM:SS.NNNNNN   <pid> <file>:<line> <msg>
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	// LogLevel
	switch r.Level {
	case LevelTrace:
		buf = append(buf, "I"...)
	case LevelDebug:
		buf = append(buf, "I"...)
	case LevelInfo:
		buf = append(buf, "I"...)
	case LevelWarning:
		buf = append(buf, "W"...)
	case LevelError:
		buf = append(buf, "E"...)
	case LevelFatal:
		buf = append(buf, "F"...)
	}

	// date/time
	if !r.Time.IsZero() {
		buf = append(buf, r.Time.Format("20060102 15:04:05.000000")...)
	}

	buf = append(buf, "   "...)

	// PID
	buf = append(buf, strconv.Itoa(os.Getpid())...)

	buf = append(buf, " "...)
	// Log line
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		buf = append(buf, getShortFileName(f.File)...)
		buf = append(buf, ":"...)
		buf = append(buf, strconv.Itoa(f.Line)...)
	}
	buf = append(buf, "]"...)
	buf = append(buf, " "...)
	buf = append(buf, r.Message...)
	buf = append(buf, "\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func getShortFileName(f string) string {
	fp := strings.Split(f, "/")
	return fp[len(fp)-1]
}
