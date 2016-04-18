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

/*
StartCLI launches the bot in a shell environment.

Packages implementing Gobot as a compiled binary launched via a CLI should
call this method to properly handle interrupts and stdin/out redirection.
*/
func StartCLI() {
	go handleInterrupt()
}

func handleInterrupt() {
	select {
	case sig := <-interrupt:
		Log.Info("Interrupt: %s", sig)
		close(done)
	}
}
