// Package logstream forwards application log lines to a Discord channel while
// still writing them to the underlying local writer.
package logstream

import (
	"io"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// Writer forwards log lines to Discord when streaming is enabled and then
// delegates to the wrapped writer.
type Writer struct {
	Base io.Writer
}

var (
	enabled   bool
	session   *discordgo.Session
	channelID string
	mutex     sync.Mutex
)

const maxLogMessage = 1800

// Write implements io.Writer.
func (w *Writer) Write(p []byte) (int, error) {
	n, err := w.Base.Write(p)
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return n, err
	}

	mutex.Lock()
	on := enabled
	s := session
	ch := channelID
	mutex.Unlock()

	if on && s != nil && ch != "" {
		if len(msg) > maxLogMessage {
			msg = msg[:maxLogMessage]
		}
		go s.ChannelMessageSend(ch, "```log\n"+msg+"\n```")
	}
	return n, err
}

// Enable directs future log lines to the given channel.
func Enable(s *discordgo.Session, id string) {
	mutex.Lock()
	enabled = true
	session = s
	channelID = id
	mutex.Unlock()
}

// Disable stops forwarding log lines to Discord.
func Disable() {
	mutex.Lock()
	enabled = false
	session = nil
	channelID = ""
	mutex.Unlock()
}
