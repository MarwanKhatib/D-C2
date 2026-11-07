package commands

import (
	"fmt"
	"runtime"
	"time"

	"discord-c2/internal/version"

	"github.com/bwmarrin/discordgo"
)

// handleDevInfo reports build metadata and runtime state.
func handleDevInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	running := 0
	if jobManager != nil {
		running = jobManager.Running()
	}
	respondEphemeral(s, i, fmt.Sprintf(
		"🧩 **D‑C2** %s\nOS: `%s/%s`\nGo: `%s`\nUptime: `%s`\nJobs running: `%d`\nChannel: `%s`",
		version.String(), runtime.GOOS, runtime.GOARCH, runtime.Version(),
		time.Since(processStart).Round(time.Second), running, channelID,
	))
}
