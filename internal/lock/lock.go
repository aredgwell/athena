// Package lock implements single-writer lock acquisition and stale lock handling.
package lock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LockMeta holds metadata written into the lock file.
type LockMeta struct {
	PID       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
}

// Manager manages repository mutation locks.
type Manager struct {
	lockDir string
	ttl     time.Duration
}

// NewManager creates a lock manager for the given lock directory and TTL.
func NewManager(lockDir string, ttl time.Duration) *Manager {
	return &Manager{lockDir: lockDir, ttl: ttl}
}

func (m *Manager) lockPath() string {
	return filepath.Join(m.lockDir, "repo.lock")
}

// Acquire attempts to acquire the repository lock. It retries until timeout.
// Returns a release function on success.
func (m *Manager) Acquire(command string, timeout time.Duration) (func() error, error) {
	if err := os.MkdirAll(m.lockDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating lock directory: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for {
		err := m.tryAcquire(command)
		if err == nil {
			release := func() error { return m.Release() }
			return release, nil
		}

		if time.Now().After(deadline) {
			meta, readErr := m.ReadMeta()
			if readErr == nil {
				return nil, fmt.Errorf("lock acquisition timeout: held by PID %d (%s) since %s",
					meta.PID, meta.Command, meta.Timestamp.Format(time.RFC3339))
			}
			return nil, fmt.Errorf("lock acquisition timeout: %w", err)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func (m *Manager) tryAcquire(command string) error {
	path := m.lockPath()

	// Attempt atomic creation (O_EXCL fails if file exists)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("lock file exists: %w", err)
	}
	defer f.Close()

	hostname, _ := os.Hostname()
	meta := LockMeta{
		PID:       os.Getpid(),
		Hostname:  hostname,
		Command:   command,
		Timestamp: time.Now().UTC(),
	}

	return json.NewEncoder(f).Encode(meta)
}

// Release removes the lock file.
func (m *Manager) Release() error {
	return os.Remove(m.lockPath())
}

// ReadMeta reads the current lock file metadata.
func (m *Manager) ReadMeta() (LockMeta, error) {
	data, err := os.ReadFile(m.lockPath())
	if err != nil {
		return LockMeta{}, err
	}
	var meta LockMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return LockMeta{}, err
	}
	return meta, nil
}

// IsStale returns true if the lock file is older than the configured TTL.
func (m *Manager) IsStale() (bool, error) {
	meta, err := m.ReadMeta()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return time.Since(meta.Timestamp) > m.ttl, nil
}

// ForceReap removes a stale lock. Returns an error if the lock is not stale.
func (m *Manager) ForceReap() error {
	stale, err := m.IsStale()
	if err != nil {
		return err
	}
	if !stale {
		return fmt.Errorf("lock is not stale (TTL: %s)", m.ttl)
	}
	return m.Release()
}

// IsLocked returns true if the lock file exists.
func (m *Manager) IsLocked() bool {
	_, err := os.Stat(m.lockPath())
	return err == nil
}
