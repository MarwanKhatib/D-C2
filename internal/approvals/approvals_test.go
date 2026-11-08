package approvals

import (
	"testing"
	"time"
)

func TestCreateAndApprove(t *testing.T) {
	m := New(time.Minute)
	req := m.Create("id", "alice", "chan")
	if req.ID == "" {
		t.Fatal("empty request id")
	}

	got, err := m.Approve(req.ID, "bob", map[string]bool{"bob": true})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if got.Command != "id" || got.Requester != "alice" {
		t.Fatalf("unexpected request: %+v", got)
	}
	if _, err := m.Approve(req.ID, "bob", nil); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound on second approve, got %v", err)
	}
}

func TestSelfApprovalRejected(t *testing.T) {
	m := New(time.Minute)
	req := m.Create("id", "alice", "chan")
	if _, err := m.Approve(req.ID, "alice", nil); err != ErrSelfApproval {
		t.Fatalf("expected ErrSelfApproval, got %v", err)
	}
	if _, ok := m.Get(req.ID); !ok {
		t.Fatal("request should remain pending after a rejected approval")
	}
}

func TestNotApproverRejected(t *testing.T) {
	m := New(time.Minute)
	req := m.Create("id", "alice", "chan")
	if _, err := m.Approve(req.ID, "mallory", map[string]bool{"bob": true}); err != ErrNotApprover {
		t.Fatalf("expected ErrNotApprover, got %v", err)
	}
}

func TestExpiry(t *testing.T) {
	m := New(time.Minute)
	now := time.Now()
	m.nowFn = func() time.Time { return now }
	req := m.Create("id", "alice", "chan")

	m.nowFn = func() time.Time { return now.Add(2 * time.Minute) }
	if _, ok := m.Get(req.ID); ok {
		t.Fatal("expected request to be pruned")
	}
	if _, err := m.Approve(req.ID, "bob", nil); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeny(t *testing.T) {
	m := New(time.Minute)
	req := m.Create("id", "alice", "chan")
	if _, err := m.Deny(req.ID, "bob", nil); err != nil {
		t.Fatalf("deny: %v", err)
	}
	if _, ok := m.Get(req.ID); ok {
		t.Fatal("request should be removed after deny")
	}
}
