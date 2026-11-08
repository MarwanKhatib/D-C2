package jobs

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	maxStoreBytes = 8 * 1024 * 1024
	maxStoreJobs  = 200
)

// jobStore persists finished jobs as JSON lines so /jobs survives restarts.
type jobStore struct {
	path string
	mu   sync.Mutex
}

func (s *jobStore) append(j Job) {
	if s == nil || s.path == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	data, err := json.Marshal(j)
	if err == nil {
		_, _ = f.Write(append(data, '\n'))
	}
	_ = f.Close()

	if fi, err := os.Stat(s.path); err == nil && fi.Size() > maxStoreBytes {
		s.compactLocked()
	}
}

func (s *jobStore) load(limit int) []Job {
	if s == nil || s.path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLast(limit)
}

func (s *jobStore) compactLocked() {
	jobs := s.readLast(maxStoreJobs)
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, j := range jobs {
		_ = enc.Encode(j)
	}
	_ = w.Flush()
	_ = f.Close()
	_ = os.Rename(tmp, s.path)
}

func (s *jobStore) readLast(limit int) []Job {
	f, err := os.Open(s.path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var all []Job
	sc := bufio.NewScanner(f)
	// Job output can be large; allow up to 1 MiB lines.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var j Job
		if err := json.Unmarshal(sc.Bytes(), &j); err != nil {
			continue
		}
		all = append(all, j)
	}
	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all
}
