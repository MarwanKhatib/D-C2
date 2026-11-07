// Package winsec exposes Windows firewall and Defender controls. Every helper
// is safe to call on non-Windows hosts and reports "unsupported" instead.
package winsec

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Truncate limits s to max bytes.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// FirewallStatus summarizes the firewall state across all profiles.
func FirewallStatus() (string, error) {
	if runtime.GOOS != "windows" {
		return "unsupported", nil
	}
	out, err := exec.Command("netsh", "advfirewall", "show", "allprofiles").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("netsh advfirewall failed: %w; output=%s", err, Truncate(string(out), 200))
	}

	enabled, disabled := false, false
	for _, raw := range strings.Split(strings.ToLower(string(out)), "\n") {
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

// DefenderStatus reports Defender real-time protection state.
func DefenderStatus() (string, error) {
	if runtime.GOOS != "windows" {
		return "unsupported", nil
	}
	script := `try { (Get-MpComputerStatus).RealTimeProtectionEnabled } catch { "" }`
	out, err := exec.Command("powershell", "-NoLogo", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return "", fmt.Errorf("Get-MpComputerStatus failed: %w; output=%s", err, Truncate(string(out), 200))
	}
	switch val := strings.TrimSpace(strings.ToLower(string(out))); val {
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

// SetFirewall turns the Windows firewall on or off for all profiles. It returns
// the reported status and the raw netsh output.
func SetFirewall(mode string) (status, output string, err error) {
	out, err := exec.Command("netsh", "advfirewall", "set", "allprofiles", "state", mode).CombinedOutput()
	if err != nil {
		return "", Truncate(string(out), 400), fmt.Errorf("set firewall %s: %w; output=%s", mode, err, Truncate(string(out), 200))
	}
	status, _ = FirewallStatus()
	return status, Truncate(string(out), 400), nil
}

// SetDefenderRealtime toggles Defender real-time protection.
func SetDefenderRealtime(mode string) (string, error) {
	disabled := "false"
	if mode == "off" {
		disabled = "true"
	}
	script := `try { Set-MpPreference -DisableRealtimeMonitoring $` + disabled + ` -ErrorAction Stop; "ok" } catch { $_.Exception.Message }`
	out, err := exec.Command("powershell", "-NoLogo", "-NonInteractive", "-Command", script).CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil || !strings.EqualFold(result, "ok") {
		if result == "" && err != nil {
			result = err.Error()
		}
		return result, fmt.Errorf("set Defender real-time %s: %s", mode, Truncate(result, 200))
	}
	status, _ := DefenderStatus()
	return status, nil
}

// ExclusionExists reports whether path is already a Defender exclusion.
func ExclusionExists(path string) (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}
	abs, _ := filepath.Abs(path)
	want := strings.ToLower(abs)
	script := `try { (Get-MpPreference).ExclusionPath -join "` + "`n" + `" } catch { "" }`
	out, err := exec.Command("powershell", "-NoLogo", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return false, fmt.Errorf("Get-MpPreference ExclusionPath failed: %w; output=%s", err, Truncate(string(out), 200))
	}
	for _, raw := range strings.Split(strings.ToLower(string(out)), "\n") {
		if line := strings.TrimSpace(raw); line != "" && strings.EqualFold(line, want) {
			return true, nil
		}
	}
	return false, nil
}

// AddExclusion adds path to the Defender exclusion list.
func AddExclusion(path string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	abs, _ := filepath.Abs(path)
	script := fmt.Sprintf(`try { Add-MpPreference -ExclusionPath "%s" -ErrorAction Stop; "ok" } catch { $_.Exception.Message }`, abs)
	out, err := exec.Command("powershell", "-NoLogo", "-NonInteractive", "-Command", script).CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil || !strings.EqualFold(result, "ok") {
		if result == "" && err != nil {
			result = err.Error()
		}
		return fmt.Errorf("Add-MpPreference failed for %q: %s", abs, Truncate(result, 200))
	}
	return nil
}

// EnsureCurrentExeExcluded guarantees the running executable is excluded from
// Defender scans.
func EnsureCurrentExeExcluded() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.Abs(exe)

	exists, err := ExclusionExists(exe)
	if err != nil {
		log.Printf("Defender exclusion check failed for %s: %v", exe, err)
		return err
	}
	if exists {
		log.Printf("Defender exclusion already present for %s", exe)
		return nil
	}
	if err := AddExclusion(exe); err != nil {
		log.Printf("Failed to add Defender exclusion for %s: %v", exe, err)
		return err
	}
	log.Printf("Added Defender exclusion for current executable: %s", exe)
	return nil
}
