// Package logstream forwards application log lines to a Discord channel while
// still writing them to the underlying local writer. Delivery is queued and
// rate-limited so a noisy client cannot flood Discord.
package logstream

import (
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

	queue   = make(chan string, 256)
	worker  sync.Once
	dropped atomic.Uint64
)

const (
	maxLogMessage = 1800
	sendInterval  = 200 * time.Millisecond
	dropReport    = 25 // ticks between drop summaries (~5s)
)

// Write implements io.Writer.
func (w *Writer) Write(p []byte) (int, error) {
	n, err := w.Base.Write(p)
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return n, err
	}

	if !isEnabled() {
		return n, err
	}
	if len(msg) > maxLogMessage {
		msg = msg[:maxLogMessage]
	}

	select {
	case queue <- msg:
	default:
		dropped.Add(1)
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

	worker.Do(func() { go deliver() })
}

// Disable stops forwarding log lines to Discord.
func Disable() {
	mutex.Lock()
	enabled = false
	session = nil
	channelID = ""
	mutex.Unlock()
	drain()
}

func isEnabled() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return enabled
}

func deliver() {
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()
	ticks := 0
	for range ticker.C {
		ticks++
		select {
		case msg := <-queue:
			send("```log\n" + msg + "\n```")
		default:
			if ticks%dropReport == 0 {
				if n := dropped.Swap(0); n > 0 {
					send("```log\n... " + strconv.FormatUint(n, 10) + " log line(s) dropped (rate limit)\n```")
				}
			}
			continue
		}
		if ticks%dropReport == 0 {
			if n := dropped.Swap(0); n > 0 {
				send("```log\n... " + strconv.FormatUint(n, 10) + " log line(s) dropped (rate limit)\n```")
			}
		}
	}
}

func send(content string) {
	mutex.Lock()
	s := session
	ch := channelID
	on := enabled
	mutex.Unlock()
	if on && s != nil && ch != "" {
		_, _ = s.ChannelMessageSend(ch, content)
	}
}

func drain() {
	for {
		select {
		case <-queue:
		default:
			return
		}
	}
}
