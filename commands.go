package gobot

import (
	"github.com/Jeffail/gabs"
)

type Command interface {
	Help() string
	Matches(string) bool
	Run(*gabs.Container, chan slackMessage) error
}
