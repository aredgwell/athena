// Package doctor implements repository diagnostics aggregation.
package doctor

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/amr-athena/athena/internal/config"
)

// CommandRunner abstracts external command execution for testability.
type CommandRunner interface {
	LookPath(name string) (string, error)
}

// ExecRunner is the real implementation.
type ExecRunner struct{}

func (ExecRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// CheckResult holds the outcome of a single diagnostic check.
type CheckResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "pass", "warn", "fail"
	Detail string `json:"detail,omitempty"`
}

// DiagResult holds the aggregate doctor outcome.
type DiagResult struct {
	OK     bool          `json:"ok"`
	Checks []CheckResult `json:"checks"`
}

// Options controls doctor behavior.
type Options struct {
	ManifestPath string
	AthenaDir    string // .athena/
	AIDir        string // .ai/
	LockDir      string
	ChecksumPath string
	PolicyLevel  config.PolicyLevel
	Tools        config.ToolsConfig
}

// Run executes all diagnostic checks.
func Run(opts Options, runner CommandRunner) *DiagResult {
	result := &DiagResult{OK: true}

	checkManifest(result, opts)
	checkManagedPaths(result, opts)
	checkTooling(result, opts, runner)
	checkLockHealth(result, opts)
	checkChecksumFile(result, opts)

	return result
}

func checkManifest(result *DiagResult, opts Options) {
	if opts.ManifestPath == "" {
		result.Checks = append(result.Checks, CheckResult{
			Name: "manifest", Status: "warn", Detail: "no manifest path configured",
		})
		return
	}

	_, err := os.Stat(opts.ManifestPath)
	if err != nil {
		result.Checks = append(result.Checks, CheckResult{
			Name: "manifest", Status: "fail", Detail: "athena.toml not found",
		})
		result.OK = false
		return
	}

	// Try parsing
	_, err = config.Load(opts.ManifestPath)
	if err != nil {
		result.Checks = append(result.Checks, CheckResult{
			Name: "manifest", Status: "fail", Detail: "manifest parse error: " + err.Error(),
		})
		result.OK = false
		return
	}

	result.Checks = append(result.Checks, CheckResult{
		Name: "manifest", Status: "pass",
	})
}

func checkManagedPaths(result *DiagResult, opts Options) {
	paths := map[string]string{
		"athena_dir": opts.AthenaDir,
		"ai_dir":     opts.AIDir,
	}

	for name, path := range paths {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			result.Checks = append(result.Checks, CheckResult{
				Name: name, Status: "warn", Detail: path + " not found",
			})
			continue
		}
		if !info.IsDir() {
			result.Checks = append(result.Checks, CheckResult{
				Name: name, Status: "fail", Detail: path + " is not a directory",
			})
			result.OK = false
			continue
		}
		result.Checks = append(result.Checks, CheckResult{
			Name: name, Status: "pass",
		})
	}
}

func checkTooling(result *DiagResult, opts Options, runner CommandRunner) {
	// Check required tools
	for _, tool := range opts.Tools.Required {
		_, err := runner.LookPath(tool)
		if err != nil {
			result.Checks = append(result.Checks, CheckResult{
				Name: "tooling", Status: "fail", Detail: tool + " missing (required)",
			})
			result.OK = false
		}
	}

	// Check recommended tools
	for _, tool := range opts.Tools.Recommended {
		_, err := runner.LookPath(tool)
		if err != nil {
			status := "warn"
			if opts.PolicyLevel == config.PolicyStrict {
				status = "fail"
				result.OK = false
			}
			result.Checks = append(result.Checks, CheckResult{
				Name: "tooling", Status: status, Detail: tool + " missing",
			})
		}
	}
}

func checkLockHealth(result *DiagResult, opts Options) {
	if opts.LockDir == "" {
		return
	}

	lockFile := filepath.Join(opts.LockDir, "athena.lock")
	_, err := os.Stat(lockFile)
	if err != nil {
		// No active lock is fine
		result.Checks = append(result.Checks, CheckResult{
			Name: "lock", Status: "pass", Detail: "no active lock",
		})
		return
	}

	result.Checks = append(result.Checks, CheckResult{
		Name: "lock", Status: "warn", Detail: "active lock found",
	})
}

func checkChecksumFile(result *DiagResult, opts Options) {
	if opts.ChecksumPath == "" {
		return
	}

	_, err := os.Stat(opts.ChecksumPath)
	if err != nil {
		result.Checks = append(result.Checks, CheckResult{
			Name: "checksums", Status: "warn", Detail: "checksums.json not found",
		})
		return
	}

	result.Checks = append(result.Checks, CheckResult{
		Name: "checksums", Status: "pass",
	})
}
