package gobot

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var (
	helpTrigger *regexp.Regexp
	helpParser  *regexp.Regexp
)

func init() {
	helpTrigger = regexp.MustCompile(`(?i)^help(?:\s(?P<cmd_name>.*))?$`)
	helpParser = regexp.MustCompile(`(?ims)^\*(?P<name>\w+)\*:\s+(?P<short>.*?)(?:[.?!]\s?)(?P<long>.*)?$`)
}

type help struct {
	name  string
	short string
	long  string
}

func extractMatchesIntoMap(str string) map[string]string {
	if !strings.HasSuffix(str, ".") {
		str = str + "."
	}

	names := helpParser.SubexpNames()[1:]
	matches := helpParser.FindStringSubmatch(str)

	Log.Debugf("Matches: %#v", matches)

	md := map[string]string{}

	for i, s := range matches[1:] {
		md[names[i]] = s
	}

	return md
}

func parseHelpText(str string) (*help, error) {
	md := extractMatchesIntoMap(str)

	h := &help{}
	h.name = strings.TrimSpace(md["name"])
	h.short = strings.TrimSpace(md["short"])
	h.long = strings.TrimSpace(md["long"])

	if h.name == "" {
		return nil, fmt.Errorf("Unable to parse name from help text: %s", str)
	}

	if h.short == "" {
		return nil, fmt.Errorf("Unable to parse short description from help text: %s", str)
	}

	return h, nil
}

func (b *Bot) extractHelps() {
	for _, cmd := range b.commands {
		h, err := parseHelpText(cmd.Help())
		if err != nil {
			Log.Error(err)
			continue
		}

		b.helps[h.name] = h
	}
}

func (b *Bot) printCommandsHelp(toChannel string, trigger string) {
	var buffer bytes.Buffer

	if trigger == "help" {
		buffer.WriteString(fmt.Sprintln("_List Of Commands_"))
		buffer.WriteString(fmt.Sprintln("*help*:  Displays this help message."))

		for _, h := range b.helps {
			buffer.WriteString(fmt.Sprintf("*%s*: %s\n", h.name, h.short))
		}

		b.sendQueue <- NewSlackMessage(toChannel, strings.TrimSpace(buffer.String()))
	} else {
		matches := helpTrigger.FindAllStringSubmatch(trigger, -1)[0]
		name := matches[1]

		h, ok := b.helps[name]

		if ok {
			b.sendQueue <- NewSlackMessage(toChannel, fmt.Sprintf("_%s_\n\n%s\n%s",
				strings.ToTitle(h.name), h.short, h.long))
			return
		}

		b.sendQueue <- NewSlackMessage(toChannel, fmt.Sprintf("Sorry, there's no command called %s.", name))
	}
}
