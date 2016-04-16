package gobot

import (
	"github.com/op/go-logging"
	"os"
)

var (
	Log    *logging.Logger
	format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} [%{level:.4s}]%{color:reset} %{message}`,
	)
	uncoloredFormat = logging.MustStringFormatter(
		`%{time:15:04:05.000} %{shortfunc} [%{level:.4s}] %{message}`,
	)
)

func init() {
	file, err := os.OpenFile("minebot.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	Log = logging.MustGetLogger("minebot")

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
