package gobot_test

import (
	"fmt"
	"github.com/jlindsey/gobot"
	"regexp"
	"strconv"
)

// Implement a more complex command that adds two numbers together.
type AddCommand struct {
	matcher *regexp.Regexp
}

func (a AddCommand) String() string {
	return `AddCommand{ trigger: "add a b" }`
}

func (a AddCommand) Matches(text string) bool {
	return a.matcher.MatchString(text)
}

func (a AddCommand) Help() string {
	return `*add*: Add two numbers together.
	Add takes two numbers and adds them together.
	ex: @gobot: add 1 2`
}

func (a AddCommand) Run(channel string, text string, out chan *gobot.SlackMessage) error {
	names := a.matcher.SubexpNames()
	matches := a.matcher.FindAllStringSubmatch(text, -1)[0]

	md := map[string]int{}
	for i, n := range matches {
		conv, err := strconv.Atoi(n)
		if err != nil {
			return err
		}
		md[names[i]] = conv
	}

	res := md["a"] + md["b"]
	out <- gobot.NewSlackMessage(channel, fmt.Sprintf("%d + %d = %d", md["a"], md["b"], res))

	return nil
}

func ExampleCommand() {
	b := gobot.NewBot()
	matcher := regexp.MustCompile(`add (?{<a>\d+) (?P<b>\d+)`)
	b.RegisterCommand(AddCommand{matcher})
}
