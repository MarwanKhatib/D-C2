package preflight

import "testing"

func TestHasPermissions(t *testing.T) {
	if !hasPermissions(requiredPermissions, requiredPermissions) {
		t.Fatal("exact permissions should pass")
	}
	if !hasPermissions(administratorPermission, requiredPermissions) {
		t.Fatal("administrator should pass")
	}
	if hasPermissions(1<<11, requiredPermissions) {
		t.Fatal("send-only should not pass")
	}
}

func TestMissingPermissions(t *testing.T) {
	if got := missingPermissions(administratorPermission); got != "" {
		t.Fatalf("administrator should report no missing bits, got %q", got)
	}
	got := missingPermissions(1 << 11)
	if got == "" || got == "unknown" {
		t.Fatalf("expected missing permission names, got %q", got)
	}
	if want := "View Channels"; !contains(got, want) {
		t.Fatalf("expected %q in %q", want, got)
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		512:             "512 B",
		1024:            "1.0 KiB",
		7 * 1024 * 1024: "7.0 MiB",
		8 * 1024 * 1024: "8.0 MiB",
	}
	for in, want := range cases {
		if got := humanSize(in); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", in, got, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
