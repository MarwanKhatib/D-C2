// Package config loads runtime configuration from an optional embedded .env
// payload and an on-disk .env file.
package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Defaults for optional runtime knobs.
const (
	DefaultJobTimeout = 120 * time.Second
	DefaultMaxOutput  = 200_000
	DefaultMaxJobs    = 8
	DefaultAuditLog   = "dc2-audit.jsonl"
	DefaultChunkSize  = 7 * 1024 * 1024
)

// Config is the runtime configuration required to start the client.
type Config struct {
	// BotToken is the Discord bot token used for the "Bot " auth scheme.
	BotToken string
	// GuildID is the Discord guild in which device channels live.
	GuildID string
	// AllowedUsers limits control to these Discord user IDs. Empty means any
	// guild member who can post in the device channel may control the host.
	AllowedUsers map[string]bool
	// JobTimeout bounds a single asynchronous command.
	JobTimeout time.Duration
	// MaxOutput is the maximum captured bytes per command (stdout+stderr).
	MaxOutput int
	// MaxJobs is the maximum number of concurrent asynchronous commands.
	MaxJobs int
	// AuditLog is the JSONL audit path. Empty disables auditing.
	AuditLog string
	// ChunkSize is the split size for downloads larger than Discord's limit.
	ChunkSize int64
}

// Allowed reports whether userID may control this device. An empty allowlist
// permits everyone, preserving the original behavior.
func (c Config) Allowed(userID string) bool {
	return len(c.AllowedUsers) == 0 || c.AllowedUsers[userID]
}

// Load applies key=value pairs from embedded (when non-empty) and then from the
// file at path, and returns the resulting configuration. Values already present
// in the process environment are not overwritten by embedded values.
func Load(embedded, path string) (Config, error) {
	if embedded != "" {
		applyEnv(embedded, false)
	}
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		applyEnvFromScanner(bufio.NewScanner(f))
	}

	cfg := Config{
		BotToken:     os.Getenv("DISCORD_BOT_TOKEN"),
		GuildID:      os.Getenv("DISCORD_GUILD_ID"),
		AllowedUsers: parseAllowedUsers(os.Getenv("DISCORD_ALLOWED_USERS")),
		JobTimeout:   envDuration("DC2_JOB_TIMEOUT", DefaultJobTimeout),
		MaxOutput:    envInt("DC2_MAX_OUTPUT", DefaultMaxOutput),
		MaxJobs:      envInt("DC2_MAX_JOBS", DefaultMaxJobs),
		AuditLog:     envString("DC2_AUDIT_LOG", defaultAuditPath()),
		ChunkSize:    int64(envInt("DC2_CHUNK_SIZE", DefaultChunkSize)),
	}
	if cfg.BotToken == "" || cfg.GuildID == "" {
		return cfg, errors.New("missing DISCORD_BOT_TOKEN or DISCORD_GUILD_ID")
	}
	return cfg, nil
}

func envString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(v)
	}
	return fallback
}

func defaultAuditPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "dc2", "audit.jsonl")
	}
	return DefaultAuditLog
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func applyEnv(contents string, overwrite bool) {
	applyEnvFromScannerWithMode(bufio.NewScanner(strings.NewReader(contents)), overwrite)
}

// parseAllowedUsers turns "id1, id2" into a set. Empty IDs are ignored.
func parseAllowedUsers(raw string) map[string]bool {
	var users map[string]bool
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if users == nil {
			users = make(map[string]bool)
		}
		users[id] = true
	}
	return users
}

func applyEnvFromScanner(scanner *bufio.Scanner) {
	applyEnvFromScannerWithMode(scanner, true)
}

func applyEnvFromScannerWithMode(scanner *bufio.Scanner, overwrite bool) {
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		if !overwrite && os.Getenv(key) != "" {
			continue
		}
		os.Setenv(key, strings.TrimSpace(parts[1]))
	}
}
