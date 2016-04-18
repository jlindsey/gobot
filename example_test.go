package gobot_test

import (
	"github.com/jlindsey/gobot"
)

// Implement a simple Command
type PingCommand struct{}

func (p PingCommand) String() string {
	return `PingCommand{ trigger: "ping" }`
}

func (p PingCommand) Matches(text string) bool {
	return text == "ping"
}

func (p PingCommand) Help() string {
	return "*ping*: A simple response command to test connectivity"
}

func (p PingCommand) Run(channel string, text string, out chan *gobot.SlackMessage) error {
	out <- gobot.NewSlackMessage(channel, "Pong!")
	return nil
}

// Main entry point
func Example() {
	bot := gobot.NewBot()
	bot.RegisterCommand(PingCommand{})
	bot.Start()
}
