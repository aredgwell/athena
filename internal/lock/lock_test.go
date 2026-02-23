package lock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "locks")
	return NewManager(dir, 15*time.Minute)
}

func TestAcquireAndRelease(t *testing.T) {
	m := newTestManager(t)

	release, err := m.Acquire("test-cmd", 1*time.Second)
	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !m.IsLocked() {
		t.Error("expected lock to be held")
	}

	meta, err := m.ReadMeta()
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if meta.Command != "test-cmd" {
		t.Errorf("command: got %s, want test-cmd", meta.Command)
	}
	if meta.PID != os.Getpid() {
		t.Errorf("PID: got %d, want %d", meta.PID, os.Getpid())
	}

	if err := release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}

	if m.IsLocked() {
		t.Error("expected lock to be released")
	}
}

func TestAcquireBlocking(t *testing.T) {
	m := newTestManager(t)

	release, err := m.Acquire("first", 1*time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	// Second acquire should timeout
	_, err = m.Acquire("second", 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error on second acquire")
	}
}

func TestAcquireWaitsForRelease(t *testing.T) {
	m := newTestManager(t)

	release, err := m.Acquire("first", 1*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Release after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		release()
	}()

	// Second acquire should succeed after release
	release2, err := m.Acquire("second", 1*time.Second)
	if err != nil {
		t.Fatalf("second acquire should succeed: %v", err)
	}
	release2()
}

func TestIsStale(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "locks")
	m := NewManager(dir, 100*time.Millisecond)

	release, err := m.Acquire("test", 1*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	stale, err := m.IsStale()
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("lock should not be stale immediately")
	}

	time.Sleep(150 * time.Millisecond)

	stale, err = m.IsStale()
	if err != nil {
		t.Fatal(err)
	}
	if !stale {
		t.Error("lock should be stale after TTL")
	}
}

func TestForceReap(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "locks")
	m := NewManager(dir, 100*time.Millisecond)

	release, err := m.Acquire("test", 1*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	// Not stale yet
	if err := m.ForceReap(); err == nil {
		t.Error("expected error when reaping non-stale lock")
	}

	time.Sleep(150 * time.Millisecond)

	if err := m.ForceReap(); err != nil {
		t.Fatalf("force reap failed: %v", err)
	}

	if m.IsLocked() {
		t.Error("lock should be removed after reap")
	}
}

func TestIsLockedNoFile(t *testing.T) {
	m := newTestManager(t)
	if m.IsLocked() {
		t.Error("expected no lock when file does not exist")
	}
}

func TestIsStaleNoFile(t *testing.T) {
	m := newTestManager(t)
	stale, err := m.IsStale()
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("no lock file should not be stale")
	}
}
