package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func initLogLevel() {
	rawLvl := viper.GetString("log_level")
	logrusLvl, err := logrus.ParseLevel(rawLvl)
	if err != nil {
		fmt.Printf("Invalid log level %s\n", rawLvl) //nolint:forbidigo
		cobra.CheckErr(err)
	}

	logrus.SetLevel(logrusLvl)
	logrus.SetFormatter(&LogrusTextFormatter{ForceColors: true})
}

const (
	red    = 31
	yellow = 33
	blue   = 36
	gray   = 37
)

type LogrusTextFormatter struct {
	ForceColors bool
}

func (f *LogrusTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var levelColor int
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	case logrus.InfoLevel:
		levelColor = blue
	default:
		levelColor = blue
	}

	buff := &bytes.Buffer{}
	showColors := f.ForceColors || f.checkIfTerminal(entry.Logger.Out)

	// err is always nil when calling WriteString or WriteByte, so we ignore it (see package docs)
	buff.WriteByte('[')
	buff.WriteString(time.Now().Format("15:04:05"))
	buff.WriteString("][")
	if showColors {
		buff.WriteString(fmt.Sprintf("\x1b[%dm", levelColor))
	}
	buff.WriteString(strings.ToUpper(entry.Level.String()[0:4]))
	if showColors {
		buff.WriteString("\x1b[0m")
	}
	buff.WriteByte(']')
	for key, value := range entry.Data {
		buff.WriteByte('[')
		if showColors {
			buff.WriteString(fmt.Sprintf("\x1b[%dm", levelColor))
		}
		buff.WriteString(key)
		if showColors {
			buff.WriteString("\x1b[0m")
		}
		buff.WriteString(fmt.Sprintf(":%s]", value))
	}
	buff.WriteByte(' ')
	// Remove a single newline if it already exists in the message to keep
	// the behavior of logrus text_formatter the same as the stdlib log package
	entry.Message = strings.TrimSuffix(entry.Message, "\n")

	buff.WriteString(entry.Message)
	buff.WriteByte('\n')
	return buff.Bytes(), nil
}

func (f *LogrusTextFormatter) checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return term.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}
