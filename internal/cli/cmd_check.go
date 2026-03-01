package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/aredgwell/athena/internal/validate"
	"github.com/spf13/cobra"
)

func runCheck(cmd *cobra.Command, args []string) error {
	start := time.Now()
	fix, _ := cmd.Flags().GetBool("fix")
	strict, _ := cmd.Flags().GetBool("strict-schema")

	opts := validate.CheckOptions{
		Dir:          aiDir(),
		Fix:          fix,
		StrictSchema: strict,
	}
	if fix {
		opts.BackupDir = filepath.Join(athenaDir(), "backups")
	}

	results, summary, err := validate.Check(opts)
	if err != nil {
		return err
	}

	env := NewEnvelope("check", time.Since(start)).WithData(map[string]any{
		"results": results,
		"summary": summary,
	})
	if summary.Invalid > 0 {
		env.OK = false
	}

	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Scanned %d files: %d valid, %d invalid",
			summary.FilesScanned, summary.Valid, summary.Invalid)
		if summary.Migratable > 0 {
			fmt.Fprintf(w, ", %d migratable", summary.Migratable)
		}
		if summary.Fixed > 0 {
			fmt.Fprintf(w, ", %d fixed", summary.Fixed)
		}
		fmt.Fprintln(w)
		for _, r := range results {
			if !r.Valid {
				fmt.Fprintf(w, "  INVALID: %s\n", r.Path)
				for _, e := range r.Errors {
					fmt.Fprintf(w, "    - %s\n", e)
				}
			}
		}
	})
	return nil
}
