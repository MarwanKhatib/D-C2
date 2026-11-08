package commands

// This file implements the optional two-person approval rule.

import (
	"fmt"
	"strings"
	"time"

	"discord-c2/internal/approvals"
	"discord-c2/internal/audit"
	"discord-c2/internal/jobs"

	"github.com/bwmarrin/discordgo"
)

const (
	approvePrefix = "d2c_approve:"
	denyPrefix    = "d2c_deny:"
	approvalTTL   = 10 * time.Minute
)

var (
	approvalMgr     *approvals.Manager
	requireApproval bool
	approvers       map[string]bool
	approvalAudit   *audit.Logger
)

// ConfigureApprovals enables the two-person rule when required is true.
func ConfigureApprovals(required bool, allowed map[string]bool, log *audit.Logger) {
	requireApproval = required
	approvers = allowed
	approvalAudit = log
	if required {
		approvalMgr = approvals.New(approvalTTL)
	}
}

// ApprovalRequired reports whether commands must be approved.
func ApprovalRequired() bool { return requireApproval }

func startApproval(s *discordgo.Session, i *discordgo.InteractionCreate, operator, channelID, cmd string) {
	if approvalMgr == nil {
		respondEphemeral(s, i, "Approval manager is not configured.")
		return
	}
	req := approvalMgr.Create(cmd, operator, channelID)
	auditApproval("approval.request", req, operator, "pending")
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    approvalText(req),
			Components: approvalComponents(req),
		},
	})
}

// RequestApprovalMessage posts an approval prompt for a message shortcut.
func RequestApprovalMessage(s *discordgo.Session, operator, channelID, cmd string) error {
	if approvalMgr == nil {
		return fmt.Errorf("approval manager is not configured")
	}
	req := approvalMgr.Create(cmd, operator, channelID)
	auditApproval("approval.request", req, operator, "pending")
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:    approvalText(req),
		Components: approvalComponents(req),
	})
	return err
}

func auditApproval(event string, req approvals.Request, actor, status string) {
	approvalAudit.Write(audit.Event{ // nil-safe
		Event:     event,
		JobID:     req.ID,
		Operator:  actor,
		ChannelID: req.ChannelID,
		Command:   req.Command,
		Status:    status,
	})
}

func approvalText(req approvals.Request) string {
	return fmt.Sprintf(
		"🔐 **Approval required**\nRequester: <@%s>\nCommand: `%s`\nRequest: `%s` (expires in %s)",
		req.Requester, req.Command, req.ID, approvalTTL,
	)
}

func approvalComponents(req approvals.Request) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				CustomID: approvePrefix + req.ID,
				Label:    "Approve",
				Style:    discordgo.SuccessButton,
			},
			discordgo.Button{
				CustomID: denyPrefix + req.ID,
				Label:    "Deny",
				Style:    discordgo.DangerButton,
			},
		}},
	}
}

func handleApproval(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData, approve bool) {
	if approvalMgr == nil || jobManager == nil {
		respondEphemeral(s, i, "Approval is not configured.")
		return
	}

	id := strings.TrimPrefix(data.CustomID, approvePrefix)
	if !approve {
		id = strings.TrimPrefix(data.CustomID, denyPrefix)
	}
	approver := interactionUserID(i)

	var (
		req approvals.Request
		err error
	)
	if approve {
		req, err = approvalMgr.Approve(id, approver, approvers)
	} else {
		req, err = approvalMgr.Deny(id, approver, approvers)
	}
	if err != nil {
		respondEphemeral(s, i, "❌ "+err.Error())
		return
	}

	if !approve {
		auditApproval("approval.deny", req, approver, "denied")
		respondEphemeral(s, i, "⛔ denied by <@"+approver+">")
		s.ChannelMessageSend(req.ChannelID, "⛔ command denied: `"+req.Command+"`")
		return
	}

	auditApproval("approval.approve", req, approver, "approved")
	job, err := jobManager.Start(req.Requester, req.ChannelID, req.Command, 0)
	if err != nil {
		respondEphemeral(s, i, "❌ "+err.Error())
		return
	}
	respondEphemeral(s, i, "✅ approved by <@"+approver+">")
	s.ChannelMessageSend(req.ChannelID, jobs.Ack(job))
}
