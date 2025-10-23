package common

import (
	"fmt"
	"log"
	"os"
)

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	prefix string
	logger *log.Logger
}

// NewLogger 创建日志器
func NewLogger(prefix string) Logger {
	return &DefaultLogger{
		prefix: prefix,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	l.log("DEBUG", msg, args...)
}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	l.log("INFO", msg, args...)
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	l.log("WARN", msg, args...)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	l.log("ERROR", msg, args...)
}

func (l *DefaultLogger) Fatal(msg string, args ...interface{}) {
	l.log("FATAL", msg, args...)
	os.Exit(1)
}

func (l *DefaultLogger) log(level string, msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	l.logger.Printf("[%s] [%s] %s", l.prefix, level, formatted)
}

