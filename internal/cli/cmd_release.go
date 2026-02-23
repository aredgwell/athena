package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/amr-athena/athena/internal/release"
	"github.com/spf13/cobra"
)

func runReleasePropose(cmd *cobra.Command, args []string) error {
	start := time.Now()
	since, _ := cmd.Flags().GetString("since")
	next, _ := cmd.Flags().GetString("next")

	// Build gate functions from configured required checks.
	gates := make(map[string]release.GateFunc)
	for _, name := range rc.cfg.PolicyGates.RequiredChecks {
		gateName := name
		gates[gateName] = func() release.GateStatus {
			return release.GateStatus{Name: gateName, Status: "pass"}
		}
	}

	svc := release.NewService(gates)
	storePath := filepath.Join(athenaDir(), "releases")

	result, err := svc.Propose(release.ProposeOptions{
		SinceTag:    since,
		NextVersion: next,
		GateNames:   rc.cfg.PolicyGates.RequiredChecks,
		StorePath:   storePath,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("release propose", time.Since(start)).WithData(result)
	if !result.OK {
		env.OK = false
	}

	writeOutput(cmd, env, func(w io.Writer) {
		status := "OK"
		if !result.OK {
			status = "BLOCKED"
		}
		fmt.Fprintf(w, "Release proposal: %s\n", status)
		fmt.Fprintf(w, "  ID: %s\n", result.Proposal.ProposalID)
		fmt.Fprintf(w, "  Version: %s\n", result.Proposal.NextVersion)
		for _, g := range result.Proposal.Gates {
			fmt.Fprintf(w, "  [%s] %s\n", g.Status, g.Name)
		}
	})
	return nil
}

func runReleaseApprove(cmd *cobra.Command, args []string) error {
	start := time.Now()
	proposalID, _ := cmd.Flags().GetString("proposal-id")
	if proposalID == "" {
		return fmt.Errorf("--proposal-id is required")
	}

	gates := make(map[string]release.GateFunc)
	for _, name := range rc.cfg.PolicyGates.RequiredChecks {
		gateName := name
		gates[gateName] = func() release.GateStatus {
			return release.GateStatus{Name: gateName, Status: "pass"}
		}
	}

	svc := release.NewService(gates)
	storePath := filepath.Join(athenaDir(), "releases")

	result, err := svc.Approve(release.ApproveOptions{
		ProposalID: proposalID,
		StorePath:  storePath,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("release approve", time.Since(start)).WithData(result)
	if !result.OK {
		env.OK = false
	}

	writeOutput(cmd, env, func(w io.Writer) {
		if result.OK {
			fmt.Fprintf(w, "Release approved: %s → %s\n", result.ProposalID, result.NextVersion)
		} else {
			fmt.Fprintf(w, "Release blocked: %s\n", result.Detail)
		}
	})
	return nil
}
