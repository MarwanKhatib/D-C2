// Package approvals implements a two-person rule for command execution.
package approvals

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("request not found or already handled")
	ErrExpired      = errors.New("request expired")
	ErrSelfApproval = errors.New("requester cannot approve their own command")
	ErrNotApprover  = errors.New("user is not an approver")
)

// Request is a pending command awaiting a second operator.
type Request struct {
	ID        string
	Command   string
	Requester string
	ChannelID string
	Created   time.Time
}

// Manager stores pending requests in memory.
type Manager struct {
	mu    sync.Mutex
	reqs  map[string]Request
	ttl   time.Duration
	nowFn func() time.Time
}

// New creates a manager with the given request lifetime.
func New(ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &Manager{reqs: make(map[string]Request), ttl: ttl, nowFn: time.Now}
}

// Create registers a pending command and returns it.
func (m *Manager) Create(command, requester, channelID string) Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()

	req := Request{
		ID:        newID(),
		Command:   command,
		Requester: requester,
		ChannelID: channelID,
		Created:   m.nowFn(),
	}
	m.reqs[req.ID] = req
	return req
}

// Get returns a pending request.
func (m *Manager) Get(id string) (Request, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()
	req, ok := m.reqs[id]
	return req, ok
}

// Approve removes and returns a request. The approver must not be the
// requester, and must be in approvers when that set is non-empty.
func (m *Manager) Approve(id, approver string, approvers map[string]bool) (Request, error) {
	return m.take(id, approver, approvers, true)
}

// Deny removes and returns a request without executing it.
func (m *Manager) Deny(id, approver string, approvers map[string]bool) (Request, error) {
	return m.take(id, approver, approvers, false)
}

func (m *Manager) take(id, approver string, approvers map[string]bool, checkSelf bool) (Request, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()

	req, ok := m.reqs[id]
	if !ok {
		return Request{}, ErrNotFound
	}
	if len(approvers) > 0 && !approvers[approver] {
		return Request{}, ErrNotApprover
	}
	if checkSelf && approver == req.Requester {
		return Request{}, ErrSelfApproval
	}
	delete(m.reqs, id)
	return req, nil
}

func (m *Manager) pruneLocked() {
	cutoff := m.nowFn().Add(-m.ttl)
	for id, req := range m.reqs {
		if req.Created.Before(cutoff) {
			delete(m.reqs, id)
		}
	}
}

func newID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405")))
	}
	return hex.EncodeToString(b[:])
}
