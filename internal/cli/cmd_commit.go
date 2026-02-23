package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/amr-athena/athena/internal/commitlint"
	"github.com/spf13/cobra"
)

func runCommitLint(cmd *cobra.Command, args []string) error {
	start := time.Now()
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")

	messages, err := gitLog(from, to)
	if err != nil {
		return err
	}

	opts := commitlint.LintOptions{
		ValidTypes:   rc.cfg.ConventionalCommits.Types,
		RequireScope: rc.cfg.ConventionalCommits.RequireScope,
	}
	if len(opts.ValidTypes) == 0 {
		opts.ValidTypes = commitlint.DefaultTypes()
	}

	summary := commitlint.LintAll(messages, opts)

	env := NewEnvelope("commit lint", time.Since(start)).WithData(summary)
	if summary.Invalid > 0 {
		env.OK = false
	}

	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Linted %d commits: %d valid, %d invalid\n",
			summary.Total, summary.Valid, summary.Invalid)
		for _, r := range summary.Results {
			if !r.Valid {
				fmt.Fprintf(w, "  INVALID: %s\n", r.Message)
				for _, e := range r.Errors {
					fmt.Fprintf(w, "    - %s\n", e)
				}
			}
		}
	})
	return nil
}
