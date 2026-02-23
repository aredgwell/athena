package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/amr-athena/athena/internal/capabilities"
	"github.com/spf13/cobra"
)

func runCapabilities(cmd *cobra.Command, args []string) error {
	start := time.Now()
	payload := capabilities.Get()
	env := NewEnvelope("capabilities", time.Since(start)).WithData(payload)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Commands (%d):\n", len(payload.Commands))
		for _, c := range payload.Commands {
			fmt.Fprintf(w, "  %s\n", c)
		}
		fmt.Fprintf(w, "Schema versions: manifest=%d frontmatter=%d telemetry=%d\n",
			payload.SchemaVersions.Manifest,
			payload.SchemaVersions.Frontmatter,
			payload.SchemaVersions.Telemetry)
	})
	return nil
}
