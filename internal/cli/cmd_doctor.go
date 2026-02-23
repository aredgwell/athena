package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/amr-athena/athena/internal/doctor"
	"github.com/spf13/cobra"
)

func runDoctor(cmd *cobra.Command, args []string) error {
	start := time.Now()

	opts := doctor.Options{
		ManifestPath: filepath.Join(rc.repoRoot, "athena.toml"),
		AthenaDir:    athenaDir(),
		AIDir:        aiDir(),
		LockDir:      filepath.Join(athenaDir(), "locks"),
		ChecksumPath: filepath.Join(athenaDir(), "checksums.json"),
		PolicyLevel:  rc.policy,
		Tools:        rc.cfg.Tools,
	}

	result := doctor.Run(opts, doctor.ExecRunner{})

	env := NewEnvelope("doctor", time.Since(start)).WithData(result)
	if !result.OK {
		env.OK = false
	}

	writeOutput(cmd, env, func(w io.Writer) {
		status := "PASS"
		if !result.OK {
			status = "FAIL"
		}
		fmt.Fprintf(w, "Doctor: %s\n", status)
		for _, c := range result.Checks {
			fmt.Fprintf(w, "  [%s] %s", c.Status, c.Name)
			if c.Detail != "" {
				fmt.Fprintf(w, ": %s", c.Detail)
			}
			fmt.Fprintln(w)
		}
	})
	return nil
}
