package jobs

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"discord-c2/internal/audit"
)

func newTestManager(t *testing.T, maxJobs int, timeout time.Duration, maxOutput int) (*Manager, *audit.Logger) {
	t.Helper()
	logger, err := audit.Open(filepath.Join(t.TempDir(), "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	return New(maxJobs, timeout, maxOutput, logger), logger
}

func waitFor(t *testing.T, m *Manager, id string) Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if j, ok := m.Get(id); ok && j.Status != StatusRunning {
			return j
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s did not finish", id)
	return Job{}
}

func TestEchoJob(t *testing.T) {
	m, _ := newTestManager(t, 2, 5*time.Second, 64*1024)
	job, err := m.Start("operator", "chan", "echo hello", 0)
	if err != nil {
		t.Fatal(err)
	}
	got := waitFor(t, m, job.ID)
	if got.Status != StatusDone || got.ExitCode != 0 {
		t.Fatalf("unexpected status: %+v", got)
	}
	if !strings.Contains(got.Output, "hello") {
		t.Fatalf("output missing marker: %q", got.Output)
	}
}

func TestTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep")
	}
	m, _ := newTestManager(t, 2, 5*time.Second, 4096)
	job, err := m.Start("operator", "chan", "sleep 5", 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	got := waitFor(t, m, job.ID)
	if got.Status != StatusTimedOut {
		t.Fatalf("expected timeout, got %+v", got)
	}
}

func TestOutputCap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses seq")
	}
	m, _ := newTestManager(t, 2, 5*time.Second, 100)
	job, err := m.Start("operator", "chan", "seq 1 10000", 0)
	if err != nil {
		t.Fatal(err)
	}
	got := waitFor(t, m, job.ID)
	if len(got.Output) > 1200 {
		t.Fatalf("output not bounded: %d bytes", len(got.Output))
	}
	if !strings.Contains(got.Output, "truncated") {
		t.Fatalf("expected truncation marker: %q", got.Output)
	}
}

func TestKill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep")
	}
	m, _ := newTestManager(t, 2, 10*time.Second, 4096)
	job, err := m.Start("operator", "chan", "sleep 5", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Kill(job.ID) {
		t.Fatal("expected kill to succeed")
	}
	got := waitFor(t, m, job.ID)
	if got.Status != StatusKilled {
		t.Fatalf("expected killed, got %+v", got)
	}
}

func TestConcurrencyLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep")
	}
	m, _ := newTestManager(t, 1, 10*time.Second, 4096)
	first, err := m.Start("operator", "chan", "sleep 1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start("operator", "chan", "echo second", 0); err != ErrTooManyJobs {
		t.Fatalf("expected ErrTooManyJobs, got %v", err)
	}
	waitFor(t, m, first.ID)
	if _, err := m.Start("operator", "chan", "echo third", 0); err != nil {
		t.Fatalf("expected slot to free up: %v", err)
	}
}

func TestPersistenceAcrossManagers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.jsonl")

	first, _ := newTestManager(t, 2, 5*time.Second, 4096)
	if err := first.EnablePersistence(path); err != nil {
		t.Fatal(err)
	}
	job, err := first.Start("operator", "chan", "echo persisted", 0)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, first, job.ID)

	second, _ := newTestManager(t, 2, 5*time.Second, 4096)
	if err := second.EnablePersistence(path); err != nil {
		t.Fatal(err)
	}
	got, ok := second.Get(job.ID)
	if !ok {
		t.Fatal("persisted job not loaded")
	}
	if got.Status != StatusDone || !strings.Contains(got.Output, "persisted") {
		t.Fatalf("unexpected persisted job: %+v", got)
	}
	if len(second.List(10)) == 0 {
		t.Fatal("persisted job missing from list")
	}
}

func TestNotifierReceivesFinishedJob(t *testing.T) {
	m, _ := newTestManager(t, 2, 5*time.Second, 4096)
	ch := make(chan Job, 1)
	m.SetNotifier(func(j Job) { ch <- j })
	if _, err := m.Start("operator", "chan", "echo done", 0); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-ch:
		if got.Status != StatusDone {
			t.Fatalf("unexpected status: %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("notifier not called")
	}
}
