package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/aredgwell/athena/internal/hooks"
	"github.com/spf13/cobra"
)

func runHooksInstall(cmd *cobra.Command, args []string) error {
	start := time.Now()
	preCommit, _ := cmd.Flags().GetBool("pre-commit")

	result, err := hooks.Install(rc.cfg.Hooks, hooks.InstallOptions{
		RepoRoot:  rc.repoRoot,
		PreCommit: preCommit,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("hooks install", time.Since(start)).WithData(result)
	writeOutput(cmd, env, func(w io.Writer) {
		for _, p := range result.Created {
			fmt.Fprintf(w, "  Created: %s\n", p)
		}
		for _, p := range result.Updated {
			fmt.Fprintf(w, "  Updated: %s\n", p)
		}
		if len(result.Created) == 0 && len(result.Updated) == 0 {
			fmt.Fprintln(w, "Hooks up to date.")
		}
	})
	return nil
}
