/*Package gobot : the funtime Go chat bot.

The Gobot package comprises a library for implementing a Slack bot in Go.
Implementing packages should acquire a pointer to a bot by calling NewBot,
add commands and listeners with RegisterCommand, and finally call Start when
setup is complete to connect to Slack and start processing messages.*/
package gobot

import (
	"encoding/json"
	"fmt"
	"github.com/Jeffail/gabs"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
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
	socketURL *url.URL
	conn      *websocket.Conn
	selfName  string
	selfID    string
	teamName  string

	commands []Command
	helps    map[string]*help

	sendQueue    chan *SlackMessage
	messageQueue chan *gabs.Container
	commandQueue chan func()
}

// String implements the Stringer interface.
func (b Bot) String() string {
	return fmt.Sprintf("Bot{team: %s, name: %s, id: %s}", b.teamName, b.selfName, b.selfID)
}

// NewBot instantiates and returns a new Bot struct.
func NewBot() *Bot {
	token := getAPITokenOrDie()

	bot := Bot{
		apiToken:     token,
		commands:     make([]Command, 0, 10),
		helps:        make(map[string]*help),
		sendQueue:    make(chan *SlackMessage, messageQueueBufferSize),
		messageQueue: make(chan *gabs.Container, messageQueueBufferSize),
		commandQueue: make(chan func(), 5),
	}

	return &bot
}

/*
Start initiates the Slack RTM sign-on process and connects to the
websocket. It starts the various goroutines and listeners that
comprise the bot's functionality.

This method starts the main run loop for the bot and so blocks
until exit conditions are met (interrupt signal caught, error arrises, etc).
Therefore, this method should not be called by implementing packages until
the bot setup is complete and all commands are registered.
*/
func (b *Bot) Start() {
	Log.Info("Hello! Starting up...")

	b.extractHelps()
	b.callSlackStartRTM()
	b.startSlackWebsocket()
	b.runMainLoop()
}

/*
RegisterCommand adds a new command to the internal commands
registry of the bot. This will allow those commands to be
triggered by messages. See the documentation of the Command
interface for mroe details.
*/
func (b *Bot) RegisterCommand(c Command) {
	Log.Debugf("Registering command: %s", c)
	b.commands = append(b.commands, c)
}

func (b *Bot) runMainLoop() {
	go b.consumeIncomingMessages()

	for {
		select {
		case msg := <-b.messageQueue:
			go b.handleIncomingMessage(msg)
		case invocation := <-b.commandQueue:
			go invocation()
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

func getAPITokenOrDie() string {
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

	socketURL, err := url.Parse(parsedBody.Path("url").Data().(string))
	if err != nil {
		Log.Fatalf("Unable to parse websocket endpoint URI: %s", err)
	}
	b.socketURL = socketURL
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
	Log.Infof("Dailing Slack at %s", b.socketURL.String())
	conn, _, err := websocket.DefaultDialer.Dial(b.socketURL.String(), nil)
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

	msgText := msg.Path("text").Data().(string)
	if !msgPrefix.MatchString(msgText) {
		return
	}
	msgText = msgPrefix.ReplaceAllString(msgText, "")

	if helpTrigger.MatchString(msgText) {
		Log.Debugf("HELP Triggered by %s", msgText)
		go b.printCommandsHelp(msg.Path("channel").Data().(string), msgText)
		return
	}

	for _, cmd := range b.commands {
		if cmd.Matches(msgText) {
			Log.Debugf("%s Triggered by %s", cmd, msgText)
			b.commandQueue <- func() { b.handleCommand(msg, cmd) }
			return
		}
	}
}

func (b *Bot) handleCommand(msg *gabs.Container, cmd Command) {
	channel := msg.Path("channel").Data().(string)
	text := msg.Path("text").Data().(string)

	Log.Debugf("Running %s", cmd)
	err := cmd.Run(channel, text, b.sendQueue)
	if err != nil {
		Log.Errorf("Error running command: %s", err)
	}
}

func (b *Bot) handleOutgoingMessage(msg *SlackMessage) {
	str, err := json.Marshal(msg)
	if err != nil {
		Log.Errorf("Unable to marshal message: %s", msg)
	}

	outgoingSem <- 1
	Log.Debugf("Sending json: %s", str)
	b.conn.WriteMessage(websocket.TextMessage, str)
	<-outgoingSem
}
