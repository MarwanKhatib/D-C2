package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoggerWritesJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "audit.jsonl")
	l, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	l.Write(Event{Event: "job.start", JobID: "abc", Operator: "1", Command: "id"})
	l.Write(Event{Event: "job.finish", JobID: "abc", Status: "done", ExitCode: 0})
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var events []Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("invalid JSON line: %v", err)
		}
		events = append(events, e)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].JobID != "abc" || events[1].Status != "done" {
		t.Fatalf("unexpected events: %+v", events)
	}
	if events[0].Time.IsZero() {
		t.Fatal("expected a timestamp on each event")
	}
}

func TestNilLoggerIsNoOp(t *testing.T) {
	var l *Logger
	l.Write(Event{Event: "noop"})
	if err := l.Close(); err != nil {
		t.Fatalf("nil close: %v", err)
	}
}

func TestEmptyPathDisables(t *testing.T) {
	l, err := Open("")
	if err != nil || l != nil {
		t.Fatalf("expected nil logger, got %v, %v", l, err)
	}
}
