package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/amr-athena/athena/internal/config"
	"github.com/amr-athena/athena/internal/scaffold"
	"github.com/amr-athena/athena/internal/templates"
	"github.com/spf13/cobra"
)

func runInit(cmd *cobra.Command, args []string) error {
	start := time.Now()
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	preset, _ := cmd.Flags().GetString("preset")

	cfg := config.ConfigForPreset(preset)
	files := templates.ManagedFiles(cfg)

	summary, err := scaffold.Init(files, scaffold.InitOptions{
		RepoRoot: rc.repoRoot,
		DryRun:   dryRun,
		Force:    force,
		Version:  version,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("init", time.Since(start)).WithData(summary)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Initialized: %d written, %d skipped", summary.Written, summary.Skipped)
		if dryRun {
			fmt.Fprint(w, " (dry-run)")
		}
		fmt.Fprintln(w)
		for _, a := range summary.Actions {
			fmt.Fprintf(w, "  [%s] %s", a.Action, a.Path)
			if a.Reason != "" {
				fmt.Fprintf(w, " (%s)", a.Reason)
			}
			fmt.Fprintln(w)
		}
	})
	return nil
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	start := time.Now()
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	files := templates.ManagedFiles(rc.cfg)

	summary, err := scaffold.Upgrade(files, scaffold.UpgradeOptions{
		RepoRoot: rc.repoRoot,
		DryRun:   dryRun,
		Version:  version,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("upgrade", time.Since(start)).WithData(summary)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Upgraded: %d written, %d overwritten, %d skipped",
			summary.Written, summary.Overwritten, summary.Skipped)
		if dryRun {
			fmt.Fprint(w, " (dry-run)")
		}
		fmt.Fprintln(w)
	})
	return nil
}
