package wecomaibot

import (
	"log"
	"os"
)

// Logger defines the logging interface used by the SDK.
// Implement this interface to integrate with your own logging framework.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// defaultLogger is a simple stdlib-based logger.
type defaultLogger struct {
	inner *log.Logger
}

// NewDefaultLogger creates a logger that writes to stderr with a
// "[wecom-aibot]" prefix.
func NewDefaultLogger() Logger {
	return &defaultLogger{
		inner: log.New(os.Stderr, "[wecom-aibot] ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (l *defaultLogger) Debug(msg string, args ...any) {
	l.inner.Printf("DEBUG "+msg, args...)
}

func (l *defaultLogger) Info(msg string, args ...any) {
	l.inner.Printf("INFO  "+msg, args...)
}

func (l *defaultLogger) Warn(msg string, args ...any) {
	l.inner.Printf("WARN  "+msg, args...)
}

func (l *defaultLogger) Error(msg string, args ...any) {
	l.inner.Printf("ERROR "+msg, args...)
}

// nopLogger discards all log messages.
type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

// NopLogger returns a logger that discards all output.
func NopLogger() Logger { return nopLogger{} }

// compile-time checks
var (
	_ Logger = (*defaultLogger)(nil)
	_ Logger = nopLogger{}
)
