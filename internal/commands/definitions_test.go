package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestDefinitionsAreUniqueAndDocumented(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		seen := make(map[string]bool)
		for _, cmd := range Definitions(goos) {
			if cmd.Name == "" {
				t.Fatalf("%s: command with empty name", goos)
			}
			if cmd.Description == "" {
				t.Errorf("%s: %s has empty description", goos, cmd.Name)
			}
			if seen[cmd.Name] {
				t.Errorf("%s: duplicate command %q", goos, cmd.Name)
			}
			seen[cmd.Name] = true
		}
	}
}

func TestWindowsGroupIsOSFiltered(t *testing.T) {
	if hasCommand(Definitions("linux"), GroupWin) {
		t.Fatal("linux definitions must not include the Windows group")
	}
	if !hasCommand(Definitions("windows"), GroupWin) {
		t.Fatal("windows definitions must include the Windows group")
	}
}

func TestGroupsExposeSubcommands(t *testing.T) {
	for _, cmd := range Definitions("windows") {
		switch cmd.Name {
		case GroupMedia, GroupFS, GroupSys, GroupWin, GroupDev, "jobs":
			if len(cmd.Options) == 0 {
				t.Errorf("%s: group has no subcommands", cmd.Name)
			}
			subs := make(map[string]bool)
			for _, opt := range cmd.Options {
				if opt.Type != discordgo.ApplicationCommandOptionSubCommand {
					t.Errorf("%s/%s: expected subcommand, got type %v", cmd.Name, opt.Name, opt.Type)
				}
				if subs[opt.Name] {
					t.Errorf("%s: duplicate subcommand %q", cmd.Name, opt.Name)
				}
				subs[opt.Name] = true
			}
		}
	}
}

func TestLegacyNamesAreNotRegistered(t *testing.T) {
	legacy := LegacyNames()
	for _, cmd := range Definitions("windows") {
		if legacy[cmd.Name] {
			t.Errorf("legacy name %q is still registered", cmd.Name)
		}
	}
}

func hasCommand(cmds []*discordgo.ApplicationCommand, name string) bool {
	for _, cmd := range cmds {
		if cmd.Name == name {
			return true
		}
	}
	return false
}
