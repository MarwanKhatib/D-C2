package main

import (
	"bufio"
	"os"
	"strings"
)

// myChannelId stores the active control channel ID for this client instance.
var myChannelId string

// guildID stores the target Discord server (guild) ID loaded from configuration.
var guildID string

// loadEnvFile reads simple KEY=VALUE lines from the given file and sets them as environment variables.
func loadEnvFile(path string) {
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
