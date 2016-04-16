package gobot

import (
	"fmt"
	"sync/atomic"
)

var msgID int32 = 0

type slackMessage struct {
	ID      int32  `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func (s slackMessage) String() string {
	return fmt.Sprintf("slackMessage{ID: %d, Type: %s, Channel: %s, Text: %s}",
		s.ID, s.Type, s.Channel, s.Text)
}

func NewSlackMessage(channel string, text string) slackMessage {
	nextID := atomic.AddInt32(&msgID, 1)
	return slackMessage{nextID, "message", channel, text}
}
