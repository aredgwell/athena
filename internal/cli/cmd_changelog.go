package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/aredgwell/athena/internal/changelog"
	"github.com/spf13/cobra"
)

func runChangelog(cmd *cobra.Command, args []string) error {
	start := time.Now()
	since, _ := cmd.Flags().GetString("since")
	next, _ := cmd.Flags().GetString("next")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	commits, err := gitLog(since, "")
	if err != nil {
		return err
	}

	result, err := changelog.Generate(changelog.Options{
		Commits:     commits,
		NextVersion: next,
		DryRun:      dryRun,
		OutputPath:  rc.cfg.Changelog.Path,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("changelog", time.Since(start)).WithData(result)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Version: %s, %d entries\n", result.Version, result.EntryCount)
		if dryRun {
			fmt.Fprintln(w, result.Markdown)
		}
	})
	return nil
}
