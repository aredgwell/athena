package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amr-athena/athena/internal/config"
)

func TestHooksInstallCommand(t *testing.T) {
	t.Run("creates config and hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git", "hooks")
		os.MkdirAll(gitDir, 0o755)

		result, err := Install(
			config.HooksConfig{PreCommit: true},
			InstallOptions{RepoRoot: dir},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Created) != 2 {
			t.Errorf("created: got %d, want 2", len(result.Created))
		}

		// Verify config file
		data, err := os.ReadFile(filepath.Join(dir, ".pre-commit-config.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "athena-check") {
			t.Error("config should contain athena-check hook")
		}

		// Verify hook script
		hookData, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "pre-commit"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(hookData), "athena check") {
			t.Error("hook should run athena check")
		}
	})

	t.Run("dry run", func(t *testing.T) {
		dir := t.TempDir()
		result, err := Install(
			config.HooksConfig{PreCommit: true},
			InstallOptions{RepoRoot: dir, DryRun: true},
		)
		if err != nil {
			t.Fatal(err)
		}
		if !result.DryRun {
			t.Error("expected dry_run=true")
		}
		// Config file should not exist
		_, err = os.Stat(filepath.Join(dir, ".pre-commit-config.yaml"))
		if !os.IsNotExist(err) {
			t.Error("dry run should not create files")
		}
	})

	t.Run("config only", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git", "hooks")
		os.MkdirAll(gitDir, 0o755)

		result, err := Install(
			config.HooksConfig{PreCommit: true},
			InstallOptions{RepoRoot: dir, ConfigOnly: true},
		)
		if err != nil {
			t.Fatal(err)
		}
		// Config should exist
		if _, err := os.Stat(filepath.Join(dir, ".pre-commit-config.yaml")); err != nil {
			t.Error("config should exist")
		}
		// Hook should not exist
		if _, err := os.Stat(filepath.Join(dir, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
			t.Error("config-only should not install hook")
		}
		if result.HookPath != "" {
			t.Error("hook_path should be empty in config-only mode")
		}
	})

	t.Run("updates existing", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git", "hooks")
		os.MkdirAll(gitDir, 0o755)

		// Write old content
		os.WriteFile(filepath.Join(dir, ".pre-commit-config.yaml"), []byte("old config"), 0o644)
		os.WriteFile(filepath.Join(dir, ".git", "hooks", "pre-commit"), []byte("old hook"), 0o755)

		result, err := Install(
			config.HooksConfig{PreCommit: true},
			InstallOptions{RepoRoot: dir},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Updated) != 2 {
			t.Errorf("updated: got %d, want 2", len(result.Updated))
		}
	})

	t.Run("pre-commit disabled", func(t *testing.T) {
		dir := t.TempDir()
		result, err := Install(
			config.HooksConfig{PreCommit: false},
			InstallOptions{RepoRoot: dir},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Created) != 0 {
			t.Error("should not create anything when hooks disabled")
		}
	})

	t.Run("override via options", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git", "hooks")
		os.MkdirAll(gitDir, 0o755)

		// Config says disabled, but options override
		result, err := Install(
			config.HooksConfig{PreCommit: false},
			InstallOptions{RepoRoot: dir, PreCommit: true},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Created) != 2 {
			t.Errorf("created: got %d, want 2 (option override)", len(result.Created))
		}
	})
}

func TestHookScriptIsExecutable(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0o755)

	Install(config.HooksConfig{PreCommit: true}, InstallOptions{RepoRoot: dir})

	info, err := os.Stat(filepath.Join(dir, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("hook script should be executable")
	}
}
