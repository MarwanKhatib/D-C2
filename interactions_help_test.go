package main

// This file contains tests for the /help slash command, ensuring that the
// handler responds with an ephemeral help message containing key sections.

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// fakeHelpResponder captures the last interaction and response sent by
// handleSlashHelp so assertions can be made in tests.
type fakeHelpResponder struct {
	lastInteraction *discordgo.Interaction
	lastResponse    *discordgo.InteractionResponse
	calls           int
}

// InteractionRespond records the interaction and response, satisfying the
// helpResponder interface used by handleSlashHelp.
func (f *fakeHelpResponder) InteractionRespond(i *discordgo.Interaction, resp *discordgo.InteractionResponse) error {
	f.calls++
	f.lastInteraction = i
	f.lastResponse = resp
	return nil
}

// TestHandleSlashHelpSendsEphemeralHelpMessage verifies that the handler sends
// a single ephemeral response containing the documented help snippets.
func TestHandleSlashHelpSendsEphemeralHelpMessage(t *testing.T) {
	responder := &fakeHelpResponder{}
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID: "test-interaction",
		},
	}

	handleSlashHelp(responder, interaction)

	if responder.calls != 1 {
		t.Fatalf("expected InteractionRespond to be called once, got %d", responder.calls)
	}
	if responder.lastInteraction != interaction.Interaction {
		t.Fatalf("expected InteractionRespond to receive the original interaction")
	}
	if responder.lastResponse == nil || responder.lastResponse.Data == nil {
		t.Fatalf("expected a non-nil response and data")
	}
	if responder.lastResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Fatalf("unexpected response type: %v", responder.lastResponse.Type)
	}
	if responder.lastResponse.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatalf("expected help message to be ephemeral")
	}
	content := responder.lastResponse.Data.Content
	for _, snippet := range []string{
		"D‑C2 Help",
		"/cmd `<command>`",
		"/screenshot",
		"**Emoji commands**",
		"Send `help` or `/help` anytime",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected help content to contain %q", snippet)
		}
	}
}
