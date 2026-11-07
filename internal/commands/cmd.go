package commands

// This file contains the /cmd slash command handler, which starts an
// asynchronous shell command job.

import "github.com/bwmarrin/discordgo"

// handleSlashCmd extracts the command string and queues it as a job.
func handleSlashCmd(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	var cmdStr string
	for _, opt := range data.Options {
		if opt.Name == "command" {
			cmdStr = opt.StringValue()
			break
		}
	}
	if cmdStr == "" {
		respondEphemeral(s, i, "No command provided.")
		return
	}
	startJob(s, i, interactionUserID(i), i.ChannelID, cmdStr)
}
