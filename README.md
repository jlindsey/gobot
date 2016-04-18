# gobot
--
    import "github.com/jlindsey/gobot"

Gobot: the funtime Go chat bot.

The Gobot package comprises a library for implementing a Slack bot in Go.
Implementing packages should acquire a pointer to a bot by calling NewBot, add
commands and listeners with RegisterCommand, and finally call Start when setup
is complete to connect to Slack and start processing messages.

## Usage

```go
var (
	// Log handle. See: https://github.com/op/go-logging
	Log *logging.Logger
)
```

#### func  StartCLI

```go
func StartCLI()
```
StartCLI launches the bot in a shell environment.

Packages implementing Gobot as a compiled binary launched via a CLI should call
this method to properly handle interrupts and stdin/out redirection.

#### type Bot

```go
type Bot struct {
}
```

Bot is the primary type of the package, encapsulating the configuration,
connections, and state of the bot.

#### func  NewBot

```go
func NewBot() *Bot
```
NewBot instantiates and returns a new Bot struct.

#### func (*Bot) RegisterCommand

```go
func (b *Bot) RegisterCommand(c Command)
```
RegisterCommand adds a new command to the internal commands registry of the bot.
This will allow those commands to be triggered by messages. See the
documentation of the Command interface for mroe details.

#### func (*Bot) Start

```go
func (b *Bot) Start()
```
Start initiates the Slack RTM sign-on process and connects to the websocket. It
starts the various goroutines and listeners that comprise the bot's
functionality.

This method starts the main run loop for the bot and so blocks until exit
conditions are met (interrupt signal caught, error arrises, etc). Therefore,
this method should not be called by implementing packages until the bot setup is
complete and all commands are registered.

#### func (Bot) String

```go
func (b Bot) String() string
```
String implements the Stringer interface.

#### type Command

```go
type Command interface {
	Help() string
	Matches(text string) bool
	Run(channel string, text string, out chan *SlackMessage) error
}
```

Command provides an interface for a bot command.

Help should return a string for the help text of the implementing command. This
string is parsed by the bot when help is requested and should be in the
following format:

    *name*: A short description. Longer description, including argument details.

The "name" portion will be parsed out and bolded in a list. The first sentence
will be displayed as the short description when listing all commands and should
not contain any line breaks. The remainder will be displayed for the detailed
help text of the command and can contain line breaks and any other formatting
you require.

Matches accepts a string of the incoming message text and returns a bool
indicating whether that text should trigger the command. By default the bot only
attempts to trigger commands that are directed at it (ie. the bot is @mentioned
at the start of the message) so this method does not need to check for this.

Run is called by the bot if Matches returns true. It is passed the channel name
that the triggering message occurred in, the text of the message, and a chan to
send any outgoing messages to.

Although not required to satisfy this interface, implementing Commands should
also define a String function to implement the Stringer interface for better
logging.

#### type SlackMessage

```go
type SlackMessage struct {
	Channel string
	Text    string
}
```

SlackMessage encapsulates a single (outgoing) Slack message payload. Incoming
messages are handled by the gabs library for flexibility.

Slack messages must contain a sequentially-incrementing ID field to ensure Slack
displays the messages in proper order even if they are sent or received out of
order. Gobot maintains a uint32 counter and atomically generates the next ID
within the NewSlackMessage method, so this type should never be used on its own
and should always be acquired via that method.

#### func  NewSlackMessage

```go
func NewSlackMessage(channel string, text string) *SlackMessage
```
Returns a new SlackMessage with the next atomically-incremented message ID.

#### func (SlackMessage) MarshalJSON

```go
func (s SlackMessage) MarshalJSON() ([]byte, error)
```
MarshalJSON() satisfies the json.Marshaler interface. This is needed to include
the unexported ID field as well as include the "type" (which for us is always
"message").

#### func (SlackMessage) String

```go
func (s SlackMessage) String() string
```
String implements the Stringer interface.
