package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aredgwell/athena/internal/config"
	atherr "github.com/aredgwell/athena/internal/errors"
	"github.com/aredgwell/athena/internal/lock"
	"github.com/spf13/cobra"
)

// runContext holds resolved runtime state for command execution.
type runContext struct {
	cfg       config.Config
	cfgLoaded bool
	policy    config.PolicyLevel
	format    string
	verbose   bool
	debug     bool
	quiet     bool
	repoRoot  string
}

// rc is the global runtime context, populated by PersistentPreRunE.
var rc *runContext

// aiDir returns the .ai notes directory.
func aiDir() string { return filepath.Join(rc.repoRoot, ".ai") }

// athenaDir returns the .athena config directory.
func athenaDir() string { return filepath.Join(rc.repoRoot, ".athena") }

// initRunContext resolves flags, loads config, and populates rc.
func initRunContext(cmd *cobra.Command) error {
	rc = &runContext{}

	// Resolve flags
	rc.verbose, _ = cmd.Flags().GetBool("verbose")
	rc.debug, _ = cmd.Flags().GetBool("debug")
	rc.quiet, _ = cmd.Flags().GetBool("quiet")
	rc.format, _ = cmd.Flags().GetString("format")

	// Resolve repo root
	root, err := resolveRepoRoot()
	if err != nil {
		// Fall back to cwd for commands that don't need a repo (init, capabilities)
		root, _ = os.Getwd()
	}
	rc.repoRoot = root

	// Load config
	cfgPath := filepath.Join(root, "athena.toml")
	cfg, loadErr := config.Load(cfgPath)
	if loadErr != nil {
		// Config not found — use defaults (needed for init, capabilities, etc.)
		cfg = config.Default()
	} else {
		rc.cfgLoaded = true
	}
	rc.cfg = cfg

	// Resolve policy (CLI flag overrides config)
	policyFlag, _ := cmd.Flags().GetString("policy")
	rc.policy = config.ResolvePolicy(policyFlag, cfg.Policy.Default)

	// Resolve format
	formatFlag, _ := cmd.Flags().GetString("format")
	rc.format = config.ResolveFormat(formatFlag, "text")

	return nil
}

// testRepoRoot overrides repo root resolution for testing.
var testRepoRoot string

// resolveRepoRoot walks up from cwd looking for .athena/ or .git/.
func resolveRepoRoot() (string, error) {
	if testRepoRoot != "" {
		return testRepoRoot, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "athena.toml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".athena")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in an athena-managed repository")
		}
		dir = parent
	}
}

// writeOutput writes JSON or text output based on the format flag.
func writeOutput(cmd *cobra.Command, env *Envelope, textFn func(w io.Writer)) {
	if rc.format == "json" {
		env.WriteJSON(cmd.OutOrStdout())
	} else {
		textFn(cmd.OutOrStdout())
	}
}

// withLock acquires the mutation lock, runs fn, then releases.
func withLock(cmd *cobra.Command, command string, fn func() error) error {
	lockDir := filepath.Join(athenaDir(), "locks")
	ttl, err := rc.cfg.Lock.TTLDuration()
	if err != nil {
		ttl = 15 * time.Minute
	}

	timeout, _ := cmd.Flags().GetDuration("lock-timeout")
	if timeout == 0 {
		timeout = ttl
	}

	mgr := lock.NewManager(lockDir, ttl)
	release, err := mgr.Acquire(command, timeout)
	if err != nil {
		return atherr.New(atherr.ExecPlanRequired,
			"lock acquisition failed: "+err.Error(),
			"Wait for the lock to release or increase --lock-timeout.",
		)
	}
	defer release()
	return fn()
}

// gitLogFn is the function used to retrieve git commit messages.
// Tests can replace this to avoid running real git commands.
var gitLogFn = gitLogExec

// gitLog returns commit messages in the given ref range.
func gitLog(from, to string) ([]string, error) {
	return gitLogFn(from, to)
}

func gitLogExec(from, to string) ([]string, error) {
	args := []string{"log", "--format=%s%n%b%n---COMMIT-BOUNDARY---"}
	if from != "" && to != "" {
		args = append(args, from+".."+to)
	} else if from != "" {
		args = append(args, from+"..HEAD")
	}

	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	raw := strings.Split(string(out), "---COMMIT-BOUNDARY---")
	var messages []string
	for _, msg := range raw {
		msg = strings.TrimSpace(msg)
		if msg != "" {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

// ExitCodeForError maps an error to an exit code per the spec (0-4).
func ExitCodeForError(err error) int {
	if err == nil {
		return 0
	}
	athErr, ok := err.(*atherr.AthenaError)
	if !ok {
		return 1
	}
	switch {
	case strings.HasPrefix(athErr.Code, atherr.NSConfig):
		return 2
	case strings.Contains(athErr.Code, "EXEC"):
		return 4
	default:
		return 1
	}
}
