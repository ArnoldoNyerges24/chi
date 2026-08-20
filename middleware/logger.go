package middleware

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

// Logger is a middleware that logs the start and end of each request, along
// with some useful data about what was requested, what the response status was,
// and how long it took to return. When standard output is a TTY, Logger will
// print in color, otherwise it will print in black and white.
//
// Logger has been deprecated in favor of RequestLogger.
func Logger(next http.Handler) http.Handler {
	return RequestLogger(DefaultLogger)(next)
}

// RequestLogger can wrap any LogFormatter to create a structured logger
// middleware.
func RequestLogger(f LogFormatter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := f.NewLogEntry(r)
			ww := NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				entry.Write(ww.Status(), ww.BytesWritten(), ww.Header(), time.Since(t1), nil)
			}()

			next.ServeHTTP(ww, WithLogEntry(r, entry))
		}
		return http.HandlerFunc(fn)
	}
}

// LogFormatter initiates the beginning of a new log entry per request.
// The returned LogEntry is used to write the log at the end of the request.
type LogFormatter interface {
	NewLogEntry(r *http.Request) LogEntry
}

// LogEntry records the lifecycle of a request.
type LogEntry interface {
	Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{})
	Panic(v interface{}, stack []byte)
}

// GetLogEntry returns the in-context LogEntry for a request.
func GetLogEntry(r *http.Request) LogEntry {
	entry, _ := r.Context().Value(LogEntryCtxKey).(LogEntry)
	return entry
}

// WithLogEntry sets the in-context LogEntry for a request.
func WithLogEntry(r *http.Request, entry LogEntry) *http.Request {
	r = r.WithContext(context.WithValue(r.Context(), LogEntryCtxKey, entry))
	return r
}

// LoggerCtxKey is the context.Context key to store the request log entry.
var LogEntryCtxKey = &contextKey{"LogEntry"}

// DefaultLogger is the default LogFormatter.
var DefaultLogger = &DefaultLogFormatter{Logger: log.New(os.Stdout, "", log.LstdFlags)}

// DefaultLogFormatter is a simple logger that prints to a Writer.
type DefaultLogFormatter struct {
	Logger  Logger
	NoColor bool
}

// NewLogEntry creates a new LogEntry for the request.
func (f *DefaultLogFormatter) NewLogEntry(r *http.Request) LogEntry {
	useColor := !f.NoColor

	entry := &defaultLogEntry{
		DefaultLogFormatter: f,
		req:                 r,
		buf:                 &bytes.Buffer{},
	}

	reqID := GetReqID(r.Context())
	if reqID != "" {
		cWrite(entry.buf, useColor, nYellow, "[%s] ", reqID)
	}

	cWrite(entry.buf, useColor, nCyan, `"%s `, r.Method)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	cWrite(entry.buf, useColor, nCyan, "%s://%s%s %s" ", scheme, r.Host, r.RequestURI, r.Proto)

	cWrite(entry.buf, useColor, nBlue, "from %s", r.RemoteAddr)

	return entry
}

type defaultLogEntry struct {
	*DefaultLogFormatter
	req *http.Request
	buf *bytes.Buffer
}

func (l *defaultLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	useColor := !l.NoColor

	cWrite(l.buf, useColor, statusColor(status), " - %03d", status)

	cWrite(l.buf, useColor, nReset, " %dB", bytes)

	cWrite(l.buf, useColor, nReset, " in %s", elapsed)

	if rctx := chi.RouteContext(l.req.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			cWrite(l.buf, useColor, nReset, " route:%q", pattern)
		}
	}

	l.buf.WriteByte('\n')

	l.Logger.Print(l.buf.String())
}

func (l *defaultLogEntry) Panic(v interface{}, stack []byte) {
	l.buf.Write(stack)
	l.Logger.Print(l.buf.String())
}

// Logger interface defines the methods needed to simplify logging.
type Logger interface {
	Print(v ...interface{})
}

// Color codes
var (
	nReset   = []byte("\033[0m")
	nRed     = []byte("\033[31m")
	nGreen   = []byte("\033[32m")
	nYellow  = []byte("\033[33m")
	nBlue    = []byte("\033[34m")
	nMagenta = []byte("\033[35m")
	nCyan    = []byte("\033[36m")
	nWhite   = []byte("\033[37m")
)

func statusColor(status int) []byte {
	switch {
	case status < 200:
		return nBlue
	case status < 300:
		return nGreen
	case status < 400:
		return nCyan
	case status < 500:
		return nYellow
	default:
		return nRed
	}
}

func cWrite(w io.Writer, useColor bool, color []byte, format string, args ...interface{}) {
	if useColor && len(color) > 0 {
		w.Write(color)
	}
	fmt.Fprintf(w, format, args...)
	if useColor && len(color) > 0 {
		w.Write(nReset)
	}
}