// Package audit writes a local JSON-lines audit trail for command execution.
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is one auditable action.
type Event struct {
	Time        time.Time `json:"time"`
	Event       string    `json:"event"`
	JobID       string    `json:"job_id,omitempty"`
	Operator    string    `json:"operator,omitempty"`
	ChannelID   string    `json:"channel_id,omitempty"`
	Command     string    `json:"command,omitempty"`
	Status      string    `json:"status,omitempty"`
	ExitCode    int       `json:"exit_code,omitempty"`
	DurationMS  int64     `json:"duration_ms,omitempty"`
	OutputBytes int       `json:"output_bytes,omitempty"`
}

// Logger appends events to a JSONL file. A nil Logger is a no-op, so callers do
// not need nil checks.
type Logger struct {
	mu   sync.Mutex
	file *os.File
}

// Open creates or appends to path. An empty path returns (nil, nil), disabling
// auditing.
func Open(path string) (*Logger, error) {
	if path == "" {
		return nil, nil
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

// Write appends one event. Errors are swallowed: auditing must never break
// command execution.
func (l *Logger) Write(e Event) {
	if l == nil || l.file == nil {
		return
	}
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	e.Time = e.Time.UTC()
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.file.Write(append(line, '\n'))
}

// Close closes the underlying file.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}
