package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "")
	t.Setenv("DISCORD_GUILD_ID", "")

	path := filepath.Join(t.TempDir(), ".env")
	contents := "# comment\nDISCORD_BOT_TOKEN=token-from-file\nDISCORD_GUILD_ID=123\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("", path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BotToken != "token-from-file" || cfg.GuildID != "123" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestEmbeddedDoesNotOverrideExistingEnv(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "from-env")
	t.Setenv("DISCORD_GUILD_ID", "")

	cfg, err := Load("DISCORD_BOT_TOKEN=from-embed\nDISCORD_GUILD_ID=456\n", "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BotToken != "from-env" {
		t.Fatalf("embedded value overrode env: %q", cfg.BotToken)
	}
	if cfg.GuildID != "456" {
		t.Fatalf("embedded guild not applied: %q", cfg.GuildID)
	}
}

func TestLoadRequiresCredentials(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "")
	t.Setenv("DISCORD_GUILD_ID", "")

	if _, err := Load("", filepath.Join(t.TempDir(), "missing.env")); err == nil {
		t.Fatal("expected an error for missing credentials")
	}
}

func TestAllowedUsers(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "t")
	t.Setenv("DISCORD_GUILD_ID", "1")
	t.Setenv("DISCORD_ALLOWED_USERS", " 111 , 222 ,, ")

	cfg, err := Load("", filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Allowed("111") || !cfg.Allowed("222") {
		t.Fatalf("expected listed users to be allowed: %+v", cfg.AllowedUsers)
	}
	if cfg.Allowed("333") {
		t.Fatal("unlisted user should be denied")
	}

	t.Setenv("DISCORD_ALLOWED_USERS", "")
	cfg, err = Load("", filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Allowed("anyone") {
		t.Fatal("empty allowlist should allow everyone")
	}
}

func TestRuntimeKnobs(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "t")
	t.Setenv("DISCORD_GUILD_ID", "1")
	t.Setenv("DC2_JOB_TIMEOUT", "30s")
	t.Setenv("DC2_MAX_OUTPUT", "4096")
	t.Setenv("DC2_MAX_JOBS", "3")
	t.Setenv("DC2_AUDIT_LOG", "/tmp/audit.jsonl")

	cfg, err := Load("", filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JobTimeout.Seconds() != 30 {
		t.Errorf("JobTimeout = %v", cfg.JobTimeout)
	}
	if cfg.MaxOutput != 4096 || cfg.MaxJobs != 3 {
		t.Errorf("limits = %d/%d", cfg.MaxOutput, cfg.MaxJobs)
	}
	if cfg.ChunkSize != DefaultChunkSize {
		t.Errorf("ChunkSize = %d", cfg.ChunkSize)
	}
	if cfg.AuditLog != "/tmp/audit.jsonl" {
		t.Errorf("AuditLog = %q", cfg.AuditLog)
	}
}

func TestInvalidRuntimeKnobsFallBack(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "t")
	t.Setenv("DISCORD_GUILD_ID", "1")
	t.Setenv("DC2_JOB_TIMEOUT", "-5s")
	t.Setenv("DC2_MAX_OUTPUT", "0")
	t.Setenv("DC2_MAX_JOBS", "not-a-number")

	cfg, err := Load("", filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JobTimeout != DefaultJobTimeout || cfg.MaxOutput != DefaultMaxOutput || cfg.MaxJobs != DefaultMaxJobs {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}
