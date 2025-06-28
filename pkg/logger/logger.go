package logger

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"time"
)

// Level представляет уровень логирования
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String возвращает строковое представление уровня логирования
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
}

type logger struct {
	out    io.Writer
	level  Level
	prefix string
}

func NewLogger(out io.Writer, level Level, prefix string) Logger {
	return &logger{
		out:    out,
		level:  level,
		prefix: prefix,
	}
}

// Debug логирует сообщение с уровнем Debug
func (l *logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info логирует сообщение с уровнем Info
func (l *logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn логирует сообщение с уровнем Warn
func (l *logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error логирует сообщение с уровнем Error
func (l *logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal логирует сообщение с уровнем Fatal и завершает программу
func (l *logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
	os.Exit(1)
}

// DefaultLogger возвращает логгер по умолчанию
func DefaultLogger() Logger {
	return NewLogger(os.Stdout, LevelInfo, "GophKeeper")
}

func (l *logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}

	msg := fmt.Sprintf(
		"%s [%s] %s:%d %s: %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		level.String(),
		path.Base(file),
		line,
		l.prefix,
		fmt.Sprintf(format, args...),
	)

	fmt.Fprint(l.out, msg)
}
