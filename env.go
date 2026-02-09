package main

// This file loads configuration from an embedded or on-disk .env file and
// exposes process-wide identifiers used by the Discord client.

import (
	"bufio"
	_ "embed"
	"os"
	"strings"
)

// myChannelId stores the Discord text channel ID associated with this device.
var myChannelId string

// guildID holds the Discord guild in which device channels and commands live.
var guildID string

// embeddedEnv optionally contains environment configuration baked into the
// binary at build time via go:embed.
//go:embed .env
var embeddedEnv string

// loadEnvFile reads key=value pairs either from the embedded .env contents or
// from the file at path and applies them to the process environment.
func loadEnvFile(path string) {
	if embeddedEnv != "" {
		scanner := bufio.NewScanner(strings.NewReader(embeddedEnv))
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
			value := strings.TrimSpace(parts[1])
			if len(key) == 0 {
				continue
			}
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
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
		value := strings.TrimSpace(parts[1])
		if len(key) == 0 {
			continue
		}
		os.Setenv(key, value)
	}
}
