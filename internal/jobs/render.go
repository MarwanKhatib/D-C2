package jobs

import (
	"fmt"
	"strings"
	"time"
)

const maxMessageChunk = 1700

// Ack renders the immediate message for a started job.
func Ack(j Job) string {
	return fmt.Sprintf("🆔 job `%s` started: `%s`", j.ID, truncateCommand(j.Command))
}

// ResultHeader renders the summary line for a finished job.
func ResultHeader(j Job) string {
	icon := "✅"
	switch j.Status {
	case StatusFailed:
		icon = "❌"
	case StatusTimedOut:
		icon = "⏱️"
	case StatusKilled:
		icon = "⛔"
	}
	header := fmt.Sprintf("%s job `%s` %s · exit %d · %s", icon, j.ID, j.Status, j.ExitCode, j.Duration().Round(time.Millisecond))
	if j.Err != "" && j.Status != StatusDone {
		header += " · " + j.Err
	}
	return header
}

// ResultChunks renders the job result as one or more Discord-sized messages.
func ResultChunks(j Job, max int) []string {
	if max <= 0 {
		max = maxMessageChunk
	}
	header := ResultHeader(j)
	body := j.Output
	if strings.TrimSpace(body) == "" {
		return []string{header + "\n```text\n<no output>\n```"}
	}

	runes := []rune(body)
	var out []string
	for start := 0; start < len(runes); start += max {
		end := start + max
		if end > len(runes) {
			end = len(runes)
		}
		chunk := header + "\n```text\n" + string(runes[start:end]) + "\n```"
		if start > 0 {
			chunk = fmt.Sprintf("job `%s` (cont.)\n```text\n%s\n```", j.ID, string(runes[start:end]))
		}
		out = append(out, chunk)
	}
	return out
}

func truncateCommand(cmd string) string {
	if len(cmd) <= 120 {
		return cmd
	}
	return cmd[:120] + "…"
}
