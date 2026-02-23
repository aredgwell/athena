package cli

import (
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

func runTools(cmd *cobra.Command, args []string) error {
	start := time.Now()
	strict, _ := cmd.Flags().GetBool("strict")

	type toolStatus struct {
		Name     string `json:"name"`
		Status   string `json:"status"`
		Required bool   `json:"required"`
	}

	var tools []toolStatus
	ok := true

	for _, t := range rc.cfg.Tools.Required {
		_, err := exec.LookPath(t)
		status := "available"
		if err != nil {
			status = "missing"
			ok = false
		}
		tools = append(tools, toolStatus{Name: t, Status: status, Required: true})
	}

	for _, t := range rc.cfg.Tools.Recommended {
		_, err := exec.LookPath(t)
		status := "available"
		if err != nil {
			status = "missing"
			if strict {
				ok = false
			}
		}
		tools = append(tools, toolStatus{Name: t, Status: status, Required: false})
	}

	env := NewEnvelope("tools", time.Since(start)).WithData(map[string]any{
		"ok":    ok,
		"tools": tools,
	})
	if !ok {
		env.OK = false
	}

	writeOutput(cmd, env, func(w io.Writer) {
		for _, t := range tools {
			label := "recommended"
			if t.Required {
				label = "required"
			}
			fmt.Fprintf(w, "  [%s] %s (%s)\n", t.Status, t.Name, label)
		}
	})
	return nil
}
