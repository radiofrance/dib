package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pterm/pterm"
)

type LogLevel int

type Logger struct {
	Writer io.Writer
	Level  LogLevel
}

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

var (
	logger        atomic.Value
	defaultLevel  = "info"
	defaultLogger = Logger{
		Writer: os.Stderr,
		Level:  LogLevelInfo,
	}

	// loggerMutex syncs all loggers, so that they don't print at the exact same time.
	loggerMutex sync.Mutex
)

func init() {
	logger.Store(defaultLogger)
}

func SetLevel(level *string) {
	if level == nil ||
		*level == "" ||
		*level == defaultLevel {
		return
	}

	log, ok := logger.Load().(Logger)
	if !ok {
		panic("invalid logger")
	}

	switch *level {
	case "debug":
		log.Level = LogLevelDebug
	case "info":
		log.Level = LogLevelInfo
	case "warning":
		log.Level = LogLevelWarn
	case "error":
		log.Level = LogLevelError
	case "fatal":
		log.Level = LogLevelFatal
	default:
		Fatalf("%q is not a valid log level", *level)
	}

	logger.Store(log)
	Infof("Log level set to %s", log.Level)
}

// LogLevelStyle returns the style of the prefix for each log level.
func (l LogLevel) LogLevelStyle() pterm.Style {
	switch l {
	case LogLevelDebug:
		return pterm.Style{pterm.FgBlack, pterm.BgGray}
	case LogLevelInfo:
		return pterm.Style{pterm.FgBlack, pterm.BgCyan}
	case LogLevelWarn:
		return pterm.Style{pterm.FgBlack, pterm.BgYellow}
	case LogLevelError:
		return pterm.Style{pterm.FgBlack, pterm.BgLightRed}
	case LogLevelFatal:
		return pterm.Style{pterm.FgBlack, pterm.BgLightRed}
	default:
		return pterm.Style{pterm.FgDefault, pterm.BgDefault}
	}
}

// MessageStyle returns the style of the message for each log level.
func (l LogLevel) MessageStyle() pterm.Style {
	switch l {
	case LogLevelDebug:
		return pterm.Style{pterm.FgGray}
	case LogLevelInfo:
		return pterm.Style{pterm.FgLightCyan}
	case LogLevelWarn:
		return pterm.Style{pterm.FgYellow}
	case LogLevelError:
		return pterm.Style{pterm.FgLightRed}
	case LogLevelFatal:
		return pterm.Style{pterm.FgLightRed}
	default:
		return pterm.Style{pterm.FgDefault, pterm.BgDefault}
	}
}

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	}
	return "Unknown"
}

// CanPrint checks if the logger can print a specific log level.
func (l Logger) CanPrint(level LogLevel) bool {
	return l.Level <= level
}

func (l Logger) log(level LogLevel, msg string) {
	if !l.CanPrint(level) {
		return
	}

	line := pterm.Gray(time.Now().Format("15:04:05")) + " "
	line += level.LogLevelStyle().Sprintf(" %-5s ", level.String()) + " "
	line += level.MessageStyle().Sprint(msg)

	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	_, _ = l.Writer.Write([]byte(line + "\n"))
}

func Get() Logger {
	l, ok := logger.Load().(Logger)
	if !ok {
		panic("invalid logger")
	}
	return l
}

func Debugf(msg string, args ...any) {
	l, ok := logger.Load().(Logger)
	if !ok {
		panic("invalid logger")
	}
	l.log(LogLevelDebug, fmt.Sprintf(msg, args...))
}

func Infof(msg string, args ...any) {
	l, ok := logger.Load().(Logger)
	if !ok {
		panic("invalid logger")
	}
	l.log(LogLevelInfo, fmt.Sprintf(msg, args...))
}

func Warnf(msg string, args ...any) {
	l, ok := logger.Load().(Logger)
	if !ok {
		panic("invalid logger")
	}
	l.log(LogLevelWarn, fmt.Sprintf(msg, args...))
}

func Errorf(msg string, args ...any) {
	l, ok := logger.Load().(Logger)
	if !ok {
		panic("invalid logger")
	}
	l.log(LogLevelError, fmt.Sprintf(msg, args...))
}

func Fatalf(msg string, args ...any) {
	l, ok := logger.Load().(Logger)
	if !ok {
		panic("invalid logger")
	}
	l.log(LogLevelFatal, fmt.Sprintf(msg, args...))
	if l.CanPrint(LogLevelFatal) {
		os.Exit(1)
	}
}
