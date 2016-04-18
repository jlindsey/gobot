package gobot

import (
	"github.com/op/go-logging"
	"os"
)

var (
	// Log handle. See: https://github.com/op/go-logging
	Log    *logging.Logger
	format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} [%{level:.4s}]%{color:reset} %{message}`,
	)
	uncoloredFormat = logging.MustStringFormatter(
		`%{time:15:04:05.000} %{shortfunc} [%{level:.4s}] %{message}`,
	)
)

const (
	logfile    = "gobot.log"
	loggerName = "gobot"
)

func init() {
	file, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	Log = logging.MustGetLogger(loggerName)

	bufferStdout := logging.NewLogBackend(os.Stdout, "", 0)
	bufferStdoutFormatted := logging.NewBackendFormatter(bufferStdout, format)
	bufferStdoutLeveled := logging.AddModuleLevel(bufferStdoutFormatted)
	bufferStdoutLeveled.SetLevel(logging.DEBUG, "")

	bufferFile := logging.NewLogBackend(file, "", 0)
	bufferFileFormatted := logging.NewBackendFormatter(bufferFile, uncoloredFormat)
	bufferFileLeveled := logging.AddModuleLevel(bufferFileFormatted)
	bufferFileLeveled.SetLevel(logging.DEBUG, "")

	logging.SetBackend(bufferStdoutLeveled, bufferFileLeveled)
}
