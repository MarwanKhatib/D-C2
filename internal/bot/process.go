package bot

import "discord-c2/internal/process"

// restartClient re-execs the current binary and returns only if it fails.
func restartClient() error { return process.Restart() }
