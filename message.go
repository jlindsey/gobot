package gobot

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

var (
	msgID uint32
)

/*
SlackMessage encapsulates a single (outgoing) Slack message payload.
Incoming messages are handled by the gabs library for flexibility.

Slack messages must contain a sequentially-incrementing ID field
to ensure Slack displays the messages in proper order even if they
are sent or received out of order. Gobot maintains a uint32 counter
and atomically generates the next ID within the NewSlackMessage method,
so this type should never be used on its own and should always be
acquired via that method.
*/
type SlackMessage struct {
	id      uint32
	Channel string
	Text    string
}

/*
MarshalJSON satisfies the json.Marshaler interface.
This is needed to include the unexported ID field as well as include
the "type" (which for us is always "message").
*/
func (s SlackMessage) MarshalJSON() ([]byte, error) {
	Log.Debugf("Marshaling Slack Message: %s", s)
	return json.Marshal(map[string]interface{}{
		"id":      s.id,
		"type":    "message",
		"channel": s.Channel,
		"text":    s.Text,
	})
}

// String implements the Stringer interface.
func (s SlackMessage) String() string {
	return fmt.Sprintf("slackMessage{ID: %d Channel: %s, Text: %s}", s.id, s.Channel, s.Text)
}

// NewSlackMessage returns a new SlackMessage with the next atomically-incremented message ID.
func NewSlackMessage(channel string, text string) *SlackMessage {
	Log.Debugf("New slack message: %s, %s", channel, text)
	nextID := atomic.AddUint32(&msgID, 1)
	return &SlackMessage{nextID, channel, text}
}
