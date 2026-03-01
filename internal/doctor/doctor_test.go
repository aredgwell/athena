package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aredgwell/athena/internal/config"
)

type mockRunner struct {
	available map[string]bool
}

func newMockRunner() *mockRunner {
	return &mockRunner{available: make(map[string]bool)}
}

func (m *mockRunner) LookPath(name string) (string, error) {
	if avail, ok := m.available[name]; ok && avail {
		return "/usr/local/bin/" + name, nil
	}
	return "", fmt.Errorf("not found: %s", name)
}

func TestDoctorAllPass(t *testing.T) {
	dir := t.TempDir()
	athenaDir := filepath.Join(dir, ".athena")
	aiDir := filepath.Join(dir, ".ai")
	os.MkdirAll(athenaDir, 0o755)
	os.MkdirAll(aiDir, 0o755)

	// Write a valid manifest
	manifest := filepath.Join(dir, "athena.toml")
	os.WriteFile(manifest, []byte("version = 2\n"), 0o644)

	checksumPath := filepath.Join(athenaDir, "checksums.json")
	os.WriteFile(checksumPath, []byte("{}"), 0o644)

	runner := newMockRunner()
	runner.available["git"] = true
	runner.available["repomix"] = true

	result := Run(Options{
		ManifestPath: manifest,
		AthenaDir:    athenaDir,
		AIDir:        aiDir,
		ChecksumPath: checksumPath,
		Tools:        config.ToolsConfig{Required: []string{"git"}, Recommended: []string{"repomix"}},
	}, runner)

	if !result.OK {
		t.Errorf("expected OK, checks: %+v", result.Checks)
	}

	found := map[string]bool{}
	for _, c := range result.Checks {
		found[c.Name] = true
		if c.Status == "fail" {
			t.Errorf("check %s failed: %s", c.Name, c.Detail)
		}
	}

	if !found["manifest"] {
		t.Error("missing manifest check")
	}
}

func TestDoctorManifestMissing(t *testing.T) {
	runner := newMockRunner()

	result := Run(Options{
		ManifestPath: filepath.Join(t.TempDir(), "nonexistent.toml"),
	}, runner)

	if result.OK {
		t.Error("expected not OK when manifest missing")
	}

	for _, c := range result.Checks {
		if c.Name == "manifest" && c.Status != "fail" {
			t.Errorf("manifest check: got %s, want fail", c.Status)
		}
	}
}

func TestDoctorManifestParseError(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "athena.toml")
	os.WriteFile(manifest, []byte("not valid {{{{ toml"), 0o644)

	runner := newMockRunner()
	result := Run(Options{ManifestPath: manifest}, runner)

	if result.OK {
		t.Error("expected not OK when manifest parse fails")
	}
}

func TestDoctorManagedPathsMissing(t *testing.T) {
	runner := newMockRunner()

	result := Run(Options{
		AthenaDir: filepath.Join(t.TempDir(), "missing-athena"),
		AIDir:     filepath.Join(t.TempDir(), "missing-ai"),
	}, runner)

	warnCount := 0
	for _, c := range result.Checks {
		if c.Status == "warn" {
			warnCount++
		}
	}
	if warnCount < 2 {
		t.Errorf("expected at least 2 warnings for missing paths, got %d", warnCount)
	}
}

func TestDoctorRequiredToolMissing(t *testing.T) {
	runner := newMockRunner()
	// git not available

	result := Run(Options{
		Tools: config.ToolsConfig{Required: []string{"git"}},
	}, runner)

	if result.OK {
		t.Error("expected not OK when required tool missing")
	}
}

func TestDoctorRecommendedToolMissingStandard(t *testing.T) {
	runner := newMockRunner()

	result := Run(Options{
		PolicyLevel: config.PolicyStandard,
		Tools:       config.ToolsConfig{Recommended: []string{"repomix"}},
	}, runner)

	// Standard policy: missing recommended is warn, not fail
	if !result.OK {
		t.Error("expected OK under standard policy for missing recommended tool")
	}

	found := false
	for _, c := range result.Checks {
		if c.Detail == "repomix missing" && c.Status == "warn" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for missing repomix")
	}
}

func TestDoctorRecommendedToolMissingStrict(t *testing.T) {
	runner := newMockRunner()

	result := Run(Options{
		PolicyLevel: config.PolicyStrict,
		Tools:       config.ToolsConfig{Recommended: []string{"repomix"}},
	}, runner)

	if result.OK {
		t.Error("expected not OK under strict policy for missing recommended tool")
	}
}

func TestDoctorLockHealth(t *testing.T) {
	t.Run("no lock", func(t *testing.T) {
		lockDir := filepath.Join(t.TempDir(), "locks")
		os.MkdirAll(lockDir, 0o755)

		runner := newMockRunner()
		result := Run(Options{LockDir: lockDir}, runner)

		for _, c := range result.Checks {
			if c.Name == "lock" && c.Status != "pass" {
				t.Errorf("lock: got %s, want pass", c.Status)
			}
		}
	})

	t.Run("active lock", func(t *testing.T) {
		lockDir := filepath.Join(t.TempDir(), "locks")
		os.MkdirAll(lockDir, 0o755)
		os.WriteFile(filepath.Join(lockDir, "athena.lock"), []byte("{}"), 0o644)

		runner := newMockRunner()
		result := Run(Options{LockDir: lockDir}, runner)

		for _, c := range result.Checks {
			if c.Name == "lock" && c.Status != "warn" {
				t.Errorf("lock: got %s, want warn", c.Status)
			}
		}
	})
}

func TestDoctorChecksumFile(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "checksums.json")
		os.WriteFile(path, []byte("{}"), 0o644)

		runner := newMockRunner()
		result := Run(Options{ChecksumPath: path}, runner)

		for _, c := range result.Checks {
			if c.Name == "checksums" && c.Status != "pass" {
				t.Errorf("checksums: got %s, want pass", c.Status)
			}
		}
	})

	t.Run("missing", func(t *testing.T) {
		runner := newMockRunner()
		result := Run(Options{ChecksumPath: filepath.Join(t.TempDir(), "missing.json")}, runner)

		for _, c := range result.Checks {
			if c.Name == "checksums" && c.Status != "warn" {
				t.Errorf("checksums: got %s, want warn", c.Status)
			}
		}
	})
}
