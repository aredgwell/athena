package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/amr-athena/athena/internal/security"
	"github.com/spf13/cobra"
)

func runSecurityScan(cmd *cobra.Command, args []string) error {
	start := time.Now()
	secrets, _ := cmd.Flags().GetBool("secrets")
	workflows, _ := cmd.Flags().GetBool("workflows")
	reportFmt, _ := cmd.Flags().GetString("report-format")

	svc := security.NewService(rc.cfg.Security, security.ExecRunner{})
	result, err := svc.Scan(security.ScanOptions{
		Secrets:      secrets,
		Workflows:    workflows,
		ReportFormat: reportFmt,
		ReportDir:    rc.cfg.Security.ReportDir,
		PolicyLevel:  rc.policy,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("security scan", time.Since(start)).WithData(result)
	if !result.OK {
		env.OK = false
	}

	writeOutput(cmd, env, func(w io.Writer) {
		status := "PASS"
		if !result.OK {
			status = "FAIL"
		}
		fmt.Fprintf(w, "Security scan: %s\n", status)
		for name, tr := range result.Tools {
			fmt.Fprintf(w, "  [%s] %s", tr.Status, name)
			if tr.Findings > 0 {
				fmt.Fprintf(w, " (%d findings)", tr.Findings)
			}
			fmt.Fprintln(w)
		}
	})
	return nil
}
