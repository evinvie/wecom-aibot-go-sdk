package wecomaibot

import (
	"log"
	"os"
)

// Logger 定义 SDK 使用的日志接口。
// 实现此接口即可接入自定义日志框架。
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// defaultLogger 是基于标准库的简单日志实现。
type defaultLogger struct {
	inner *log.Logger
}

// NewDefaultLogger 创建一个输出到 stderr、带 "[wecom-aibot]" 前缀的日志实例。
func NewDefaultLogger() Logger {
	return &defaultLogger{
		inner: log.New(os.Stderr, "[wecom-aibot] ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (l *defaultLogger) Debug(msg string, args ...any) { l.inner.Printf("DEBUG "+msg, args...) }
func (l *defaultLogger) Info(msg string, args ...any)  { l.inner.Printf("INFO  "+msg, args...) }
func (l *defaultLogger) Warn(msg string, args ...any)  { l.inner.Printf("WARN  "+msg, args...) }
func (l *defaultLogger) Error(msg string, args ...any) { l.inner.Printf("ERROR "+msg, args...) }

// nopLogger 丢弃所有日志输出。
type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

// NopLogger 返回一个丢弃所有输出的日志实例。
func NopLogger() Logger { return nopLogger{} }

// 编译期接口检查
var (
	_ Logger = (*defaultLogger)(nil)
	_ Logger = nopLogger{}
)
