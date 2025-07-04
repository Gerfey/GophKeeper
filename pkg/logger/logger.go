package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

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
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

type logger struct {
	out    io.Writer
	level  Level
	prefix string
}

const (
	CallerSkipFrames = 2
)

func NewLogger(out io.Writer, level Level, prefix string) Logger {
	return &logger{
		out:    out,
		level:  level,
		prefix: prefix,
	}
}

func (l *logger) Debugf(format string, args ...any) {
	l.logf(LevelDebug, format, args...)
}

func (l *logger) Infof(format string, args ...any) {
	l.logf(LevelInfo, format, args...)
}

func (l *logger) Warnf(format string, args ...any) {
	l.logf(LevelWarn, format, args...)
}

func (l *logger) Errorf(format string, args ...any) {
	l.logf(LevelError, format, args...)
}

func (l *logger) Fatalf(format string, args ...any) {
	l.logf(LevelFatal, format, args...)
	os.Exit(1)
}

func DefaultLogger() Logger {
	return NewLogger(os.Stdout, LevelInfo, "GophKeeper")
}

func (l *logger) logf(level Level, format string, args ...any) {
	if level < l.level {
		return
	}

	_, file, line, ok := runtime.Caller(CallerSkipFrames)
	if !ok {
		file = "???"
		line = 0
	}

	msg := fmt.Sprintf(
		"%s [%s] %s:%d: %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		level.String(),
		filepath.Base(file),
		line,
		fmt.Sprintf(format, args...),
	)

	_, _ = fmt.Fprint(l.out, msg)
}
