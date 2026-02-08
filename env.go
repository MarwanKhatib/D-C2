package main

import (
	"bufio"
	_ "embed"
	"os"
	"strings"
)

var myChannelId string

var guildID string

//go:embed .env
var embeddedEnv string

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
