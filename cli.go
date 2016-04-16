package gobot

import (
	"os"
	"os/signal"
)

var (
	interrupt = make(chan os.Signal)
)

func init() {
	signal.Notify(interrupt, os.Interrupt)
}

func StartCLI() {
	go handleInterrupt()
	bot := NewBot()
	bot.Start()
}

func handleInterrupt() {
	select {
	case sig := <-interrupt:
		Log.Info("Interrupt: %s", sig)
		close(done)
	}
}
