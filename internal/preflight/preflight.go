// Package preflight validates configuration and Discord access before the
// client connects. It never prints the bot token.
package preflight

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"discord-c2/internal/audit"
	"discord-c2/internal/config"
	"discord-c2/internal/transfer"
)

const apiBase = "https://discord.com/api/v10"

// Required bot permissions: View Channel, Add Reactions, Manage Channels,
// Send Messages, Attach Files, Read History, Manage Webhooks.
const requiredPermissions int64 = 1<<10 | 1<<6 | 1<<4 | 1<<11 | 1<<15 | 1<<16 | 1<<29

const administratorPermission int64 = 1 << 3

type discordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type discordGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
}

// Run performs local and remote checks and returns an error if any hard check
// fails. Warnings do not fail the run.
func Run(cfg config.Config) error {
	fmt.Println("D-C2 preflight")
	failures := 0

	ok := func(format string, args ...any) { fmt.Printf("[ok]   "+format+"\n", args...) }
	warn := func(format string, args ...any) { fmt.Printf("[warn] "+format+"\n", args...) }
	fail := func(format string, args ...any) {
		fmt.Printf("[fail] "+format+"\n", args...)
		failures++
	}

	client := &http.Client{Timeout: 15 * time.Second}

	var me discordUser
	if err := getJSON(client, cfg.BotToken, "/users/@me", &me); err != nil {
		fail("Discord API/token: %v", err)
	} else {
		ok("token valid: %s (%s)", me.Username, me.ID)
	}

	if me.ID != "" {
		var guilds []discordGuild
		if err := getJSON(client, cfg.BotToken, "/users/@me/guilds", &guilds); err != nil {
			fail("guild list: %v", err)
		} else {
			var found *discordGuild
			for i := range guilds {
				if guilds[i].ID == cfg.GuildID {
					found = &guilds[i]
					break
				}
			}
			switch {
			case found == nil:
				fail("bot is not in guild %s", cfg.GuildID)
			default:
				perms, err := strconv.ParseInt(found.Permissions, 10, 64)
				if err != nil {
					fail("guild permissions: %v", err)
				} else if hasPermissions(perms, requiredPermissions) {
					ok("guild: %s (%s), permissions sufficient", found.Name, found.ID)
				} else {
					fail("guild %s missing permissions: %s", found.Name, missingPermissions(perms))
				}
			}
		}
	}

	if cfg.AuditLog == "" {
		warn("audit log disabled (DC2_AUDIT_LOG is empty)")
	} else if l, err := audit.Open(cfg.AuditLog); err != nil {
		fail("audit log %s: %v", cfg.AuditLog, err)
	} else {
		_ = l.Close()
		ok("audit log writable: %s", cfg.AuditLog)
	}

	ok("chunk size: %s", humanSize(transfer.NormalizeChunkSize(cfg.ChunkSize)))
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		warn("ffmpeg not found; media commands will be unavailable")
	} else {
		ok("ffmpeg found in PATH")
	}

	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(".env"); err == nil && fi.Mode().Perm()&0o077 != 0 {
			warn(".env is mode %04o; use chmod 600", fi.Mode().Perm())
		} else if err == nil {
			ok(".env permissions are restrictive")
		}
	}

	warn("Message Content Intent cannot be verified via API; enable it in the portal")

	if failures > 0 {
		return fmt.Errorf("%d check(s) failed", failures)
	}
	return nil
}

func getJSON(client *http.Client, token, path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, apiBase+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("User-Agent", "D-C2-preflight")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func hasPermissions(perms, required int64) bool {
	return perms&administratorPermission != 0 || perms&required == required
}

func missingPermissions(perms int64) string {
	if perms&administratorPermission != 0 {
		return ""
	}
	names := []struct {
		bit  int64
		name string
	}{
		{1 << 10, "View Channels"},
		{1 << 6, "Add Reactions"},
		{1 << 4, "Manage Channels"},
		{1 << 11, "Send Messages"},
		{1 << 15, "Attach Files"},
		{1 << 16, "Read Message History"},
		{1 << 29, "Manage Webhooks"},
	}
	out := ""
	for _, n := range names {
		if perms&n.bit == 0 {
			if out != "" {
				out += ", "
			}
			out += n.name
		}
	}
	if out == "" {
		return "unknown"
	}
	return out
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
