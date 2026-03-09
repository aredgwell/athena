package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/aredgwell/athena/internal/policy"
	"github.com/spf13/cobra"
)

func runPolicyGate(cmd *cobra.Command, args []string) error {
	start := time.Now()
	pr, _ := cmd.Flags().GetString("pr")

	gate := policy.NewGate(rc.cfg.PolicyGates, policy.PassingChecks(rc.cfg.PolicyGates.RequiredChecks))
	result, err := gate.Evaluate(policy.GateOptions{
		TargetRef:   pr,
		PolicyLevel: rc.policy,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("policy gate", time.Since(start)).WithData(result)
	if !result.OK {
		env.OK = false
	}

	writeOutput(cmd, env, func(w io.Writer) {
		status := "PASS"
		if !result.OK {
			status = "FAIL"
		}
		fmt.Fprintf(w, "Policy gate: %s (ref: %s)\n", status, result.TargetRef)
		for _, p := range result.Passed {
			fmt.Fprintf(w, "  [pass] %s\n", p)
		}
		for _, f := range result.Failures {
			fmt.Fprintf(w, "  [%s] %s: %s\n", f.Severity, f.PolicyID, f.Summary)
		}
	})
	return nil
}
