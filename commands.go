package gobot

/*
Command provides an interface for a bot command.

Help should return a string for the help text of the implementing
command. This string is parsed by the bot when help is requested
and should be in the following format:

	*name*: A short description. Longer description, including argument details.

The "name" portion will be parsed out and bolded in a list. The first sentence will
be displayed as the short description when listing all commands and should not
contain any line breaks. The remainder will be displayed for the detailed help
text of the command and can contain line breaks and any other formatting you require.

Matches accepts a string of the incoming message text and returns a bool indicating
whether that text should trigger the command. By default the bot only attempts to
trigger commands that are directed at it (ie. the bot is @mentioned at the start of
the message) so this method does not need to check for this.

Run is called by the bot if Matches returns true. It is passed the channel name
that the triggering message occurred in, the text of the message, and a chan
to send any outgoing messages to.

Although not required to satisfy this interface, implementing Commands should also
define a String function to implement the Stringer interface for better logging.
*/
type Command interface {
	Help() string
	Matches(text string) bool
	Run(channel string, text string, out chan *SlackMessage) error
}
