package main

// This file contains the /cmd slash command handler, which runs arbitrary
// shell commands on the controlled device.

import "github.com/bwmarrin/discordgo"

// handleSlashCmd extracts the command string from the interaction and runs it
// via the shared runCommandString helper, replying with ephemeral status.
func handleSlashCmd(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	var cmdStr string
	for _, opt := range data.Options {
		if opt.Name == "command" {
			cmdStr = opt.StringValue()
			break
		}
	}
	if cmdStr == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No command provided.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Running: `" + cmdStr + "`",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	runCommandString(s, i.ChannelID, nil, cmdStr)
}
