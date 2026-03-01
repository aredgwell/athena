package cli

import (
	mcppkg "github.com/aredgwell/athena/internal/mcp"
	"github.com/spf13/cobra"
)

func runMCP(cmd *cobra.Command, args []string) error {
	mcppkg.Version = version
	return mcppkg.Run(cmd.Context(), rc.repoRoot)
}
