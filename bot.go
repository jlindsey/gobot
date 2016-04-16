/* Gobot: the funtime Go chat bot. */
package gobot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Jeffail/gabs"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	apiTokenEnvKey         = "SLACK_API_TOKEN"
	apiRTMStartEndpoint    = "https://slack.com/api/rtm.start"
	messageQueueBufferSize = 10
)

var (
	done        = make(chan bool)
	outgoingSem = make(chan int, 1)
	msgPrefix   *regexp.Regexp
)

/*
Bot is the primary type of the package, encapsulating the configuration,
connections, and state of the bot.
*/
type Bot struct {
	apiToken  string
	socketUrl *url.URL
	conn      *websocket.Conn
	selfName  string
	selfID    string
	teamName  string

	commands []Command

	sendQueue    chan slackMessage
	messageQueue chan *gabs.Container
	commandQueue chan func()
}

// String implements the Stringer interface.
func (b Bot) String() string {
	return fmt.Sprintf("Bot{team: %s, name: %s, id: %s}", b.teamName, b.selfName, b.selfID)
}

func NewBot() *Bot {
	token := getApiTokenOrDie()

	bot := Bot{
		apiToken:     token,
		sendQueue:    make(chan slackMessage, messageQueueBufferSize),
		messageQueue: make(chan *gabs.Container, messageQueueBufferSize),
		commandQueue: make(chan func(), 5),
	}

	return &bot
}

func (b *Bot) Start() {
	Log.Info("Hello! Starting up...")

	b.callSlackStartRTM()
	b.startSlackWebsocket()
	defer b.Close()
	b.runMainLoop()
}

func (b *Bot) Close() {
}

func (b *Bot) runMainLoop() {
	go b.consumeIncomingMessages()

	for {
		select {
		case msg := <-b.messageQueue:
			go b.handleIncomingMessage(msg)
		case cmdWrapper := <-b.commandQueue:
			go cmdWrapper()
		case msg := <-b.sendQueue:
			go b.handleOutgoingMessage(msg)
		case <-done:
			Log.Info("Closing gracefully")
			b.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			select {
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func getApiTokenOrDie() string {
	token := os.Getenv(apiTokenEnvKey)
	if len(token) == 0 {
		Log.Fatalf("Can't find slack token in env var %s", apiTokenEnvKey)
	}
	return token
}

func (b *Bot) callSlackStartRTM() {
	Log.Info("Calling Slack RTM start")

	postVars := url.Values{}
	postVars.Set("token", b.apiToken)
	postVars.Set("simple_latest", "true")
	postVars.Set("no_unreads", "true")

	resp, err := http.PostForm(apiRTMStartEndpoint, postVars)
	if err != nil {
		Log.Fatalf("Unable to connect to RTM service: %s", err)
	}
	defer resp.Body.Close()

	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Fatalf("Unable to read response body: %s", err)
	}

	parsedBody, err := gabs.ParseJSON(rawBody)
	if err != nil {
		Log.Fatalf("Unable to parse response body: %s", err)
	}

	if !parsedBody.Path("ok").Data().(bool) {
		Log.Fatalf("Bad response from RTM start call: %s", parsedBody)
	}

	socketUrl, err := url.Parse(parsedBody.Path("url").Data().(string))
	if err != nil {
		Log.Fatalf("Unable to parse websocket endpoint URI: %s", err)
	}
	b.socketUrl = socketUrl
	b.teamName = parsedBody.Path("team.name").Data().(string)
	b.selfName = parsedBody.Path("self.name").Data().(string)
	b.selfID = parsedBody.Path("self.id").Data().(string)

	prefixRegStr := `^<@%s>:?\s?`
	msgPrefix, err = regexp.Compile(fmt.Sprintf(prefixRegStr, b.selfID))
	if err != nil {
		Log.Fatalf(`Unable to compile regexp from "%s" for msgPrefix: %s`, prefixRegStr, err)
	}
}

func (b *Bot) startSlackWebsocket() {
	Log.Infof("Dailing Slack at %s", b.socketUrl.String())
	conn, _, err := websocket.DefaultDialer.Dial(b.socketUrl.String(), nil)
	if err != nil {
		Log.Fatalf("Unable to open websocket to Slack: %s", err)
	}

	b.conn = conn
	Log.Infof("Connected to %s as %s!", b.teamName, b.selfName)
}

func (b *Bot) consumeIncomingMessages() {
	defer close(done)

	for {
		msgType, msg, err := b.conn.ReadMessage()
		if err != nil {
			Log.Errorf("Error reading message: %s", err)
			return
		}
		Log.Debugf("Raw incoming message: [%d] %s", msgType, msg)

		if msgType == websocket.TextMessage {
			parsedMsg, err := gabs.ParseJSON(msg)
			if err != nil {
				Log.Errorf("Error parsing message: %s", err)
				continue
			}
			b.messageQueue <- parsedMsg
		}
	}
}

func (b *Bot) handleIncomingMessage(msg *gabs.Container) {
	if !msg.Exists("type") || !msg.Exists("text") || msg.Path("type").Data().(string) != "message" {
		return
	}

	Log.Debugf("New message: %s", msg)

	var msgText string = msg.Path("text").Data().(string)
	if !msgPrefix.MatchString(msgText) {
		return
	}
	msgText = msgPrefix.ReplaceAllString(msgText, "")

	if msgText == "help" {
		Log.Debugf("HELP Triggered by %s", msgText)
		go b.printCommandsHelp(msg.Path("channel").Data().(string))
		return
	}

	for _, cmd := range b.commands {
		if cmd.Matches(msgText) {
			Log.Debugf("Triggered by %s", msgText)
			b.commandQueue <- func() { b.handleCommand(msg, cmd) }
		}
	}
}

func (b *Bot) printCommandsHelp(toChannel string) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintln("_List Of Commands_"))
	buffer.WriteString(fmt.Sprintln("*help*:  Displays this help message."))

	for _, cmd := range b.commands {
		buffer.WriteString(fmt.Sprintln(cmd.Help()))
	}

	b.sendQueue <- NewSlackMessage(toChannel, strings.TrimSpace(buffer.String()))
}

func (b *Bot) handleCommand(triggeringMessage *gabs.Container, cmd Command) {
	err := cmd.Run(triggeringMessage, b.sendQueue)
	if err != nil {
		Log.Errorf("Error running command: %s", err)
	}
}

func (b *Bot) handleOutgoingMessage(msg slackMessage) {
	str, err := json.Marshal(msg)
	if err != nil {
		Log.Errorf("Unable to marshal message: %s", msg)
	}

	outgoingSem <- 1
	Log.Debugf("Sending json: %s", str)
	b.conn.WriteMessage(websocket.TextMessage, str)
	<-outgoingSem
}
