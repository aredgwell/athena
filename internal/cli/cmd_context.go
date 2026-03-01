package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/aredgwell/athena/internal/context"
	"github.com/spf13/cobra"
)

func runContextPack(cmd *cobra.Command, args []string) error {
	start := time.Now()
	profile, _ := cmd.Flags().GetString("profile")
	changed, _ := cmd.Flags().GetBool("changed")
	stdout, _ := cmd.Flags().GetBool("stdout")
	output, _ := cmd.Flags().GetString("output")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	svc := context.NewService(rc.cfg.Context, context.ExecRunner{})
	result, err := svc.Pack(context.PackOptions{
		Profile:     profile,
		Changed:     changed,
		Stdout:      stdout,
		OutputPath:  output,
		DryRun:      dryRun,
		PolicyLevel: rc.policy,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("context pack", time.Since(start)).WithData(result)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Profile: %s\n", result.Profile)
		if result.DryRun {
			fmt.Fprintf(w, "Args: %v (dry-run)\n", result.Args)
		} else if result.OutputPath != "" {
			fmt.Fprintf(w, "Output: %s\n", result.OutputPath)
		}
	})
	return nil
}

func runContextMCP(cmd *cobra.Command, args []string) error {
	start := time.Now()
	stdio, _ := cmd.Flags().GetBool("stdio")

	svc := context.NewService(rc.cfg.Context, context.ExecRunner{})
	result, err := svc.MCP(context.MCPOptions{Stdio: stdio})
	if err != nil {
		return err
	}

	env := NewEnvelope("context mcp", time.Since(start)).WithData(result)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "MCP started: %v\n", result.Started)
	})
	return nil
}

func runContextBudget(cmd *cobra.Command, args []string) error {
	start := time.Now()
	profile, _ := cmd.Flags().GetString("profile")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")

	svc := context.NewService(rc.cfg.Context, context.ExecRunner{})
	result, err := svc.Budget(context.BudgetOptions{
		Profile:     profile,
		MaxTokens:   maxTokens,
		PolicyLevel: rc.policy,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("context budget", time.Since(start)).WithData(result)
	if !result.WithinBudget {
		env.AddWarning("context budget exceeded")
	}

	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Profile: %s\n", result.Profile)
		fmt.Fprintf(w, "Estimated tokens: %d\n", result.EstimatedTokens)
		if result.MaxTokens > 0 {
			fmt.Fprintf(w, "Max tokens: %d\n", result.MaxTokens)
			fmt.Fprintf(w, "Within budget: %v\n", result.WithinBudget)
		}
	})
	return nil
}
