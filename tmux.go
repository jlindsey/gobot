package gobot

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const tmuxServerName = "minecraft"

var commandSem = make(chan int, 1)

func randHash() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	c := md5.Sum(b)
	return fmt.Sprintf("%x", c), nil
}

func getDelimiters(hash string) (string, string) {
	begin := fmt.Sprintf("### %s ###", hash)
	end := fmt.Sprintf("###/ %s ###", hash)

	return begin, end
}

func commandsForOperation(keys string) (hash string, commands [][]string, err error) {
	hash, err = randHash()
	if err != nil {
		return
	}

	begin, end := getDelimiters(hash)

	commands = [][]string{
		[]string{"-L", tmuxServerName, "send-keys", begin, "Enter"},
		[]string{"-L", tmuxServerName, "send-keys", keys, "Enter"},
		[]string{"-L", tmuxServerName, "send-keys", end, "Enter"},
		[]string{"-L", tmuxServerName, "capture-pane"},
		[]string{"-L", tmuxServerName, "show-buffer"},
	}

	return
}

func parseOutput(str string, hash string) (s string, err error) {
	begin, end := getDelimiters(hash)

	start_i := int64(strings.Index(str, begin))
	if start_i == -1 {
		err = errors.New("Unable to find start delimiter in tmux output")
		return
	}

	start_i = start_i + int64(len(begin))
	end_i := int64(strings.Index(str, end))
	if end_i == -1 {
		err = errors.New("Unable to find end delimiter in tmux output")
		return
	}
	length := end_i - start_i

	b := make([]byte, length)
	r := strings.NewReader(str)
	r.Seek(start_i, 0)
	i, err := r.Read(b)

	s = strings.SplitN(string(b[:i]), "\n", 4)[3]
	return
}

/*
TmuxSendKeysAndCapture runs a command inside a detached tmux client,
and captures the buffer output. The input string is not properly sanitized
so use this at your own risk.
*/
func TmuxSendKeysAndCapture(keys string) (str string, err error) {
	var (
		hash     string
		commands [][]string
		buf      bytes.Buffer
	)

	commandSem <- 1
	hash, commands, err = commandsForOperation(keys)
	if err != nil {
		return
	}

	for i, args := range commands {
		cmd := exec.Command("tmux", args...)

		if (i + 1) == len(commands) {
			cmd.Stdout = &buf
		}

		err = cmd.Run()
		if err != nil {
			err = errors.New(fmt.Sprintf("Error running command: %s", err))
			return
		}
	}
	<-commandSem

	str, err = parseOutput(buf.String(), hash)
	return
}
