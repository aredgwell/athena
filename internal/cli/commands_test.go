package cli

import (
	"strings"
	"testing"
)

func TestAllCommandsWired(t *testing.T) {
	// All commands from ATHENA.md CLI surface
	commands := []struct {
		args    []string
		contain string
	}{
		{[]string{"init", "--help"}, "init"},
		{[]string{"upgrade", "--help"}, "upgrade"},
		{[]string{"check", "--help"}, "check"},
		{[]string{"index", "--help"}, "index"},
		{[]string{"gc", "--help"}, "gc"},
		{[]string{"tools", "--help"}, "tools"},
		{[]string{"doctor", "--help"}, "doctor"},
		{[]string{"capabilities", "--help"}, "capabilities"},
		{[]string{"report", "--help"}, "report"},
		{[]string{"changelog", "--help"}, "changelog"},
		{[]string{"completion", "--help"}, "completion"},
		{[]string{"policy", "gate", "--help"}, "gate"},
		{[]string{"plan", "--help"}, "plan"},
		{[]string{"apply", "--help"}, "apply"},
		{[]string{"rollback", "--help"}, "rollback"},
		{[]string{"security", "scan", "--help"}, "scan"},
		{[]string{"context", "pack", "--help"}, "pack"},
		{[]string{"context", "mcp", "--help"}, "mcp"},
		{[]string{"context", "budget", "--help"}, "budget"},
		{[]string{"note", "new", "--help"}, "new"},
		{[]string{"note", "close", "--help"}, "close"},
		{[]string{"note", "promote", "--help"}, "promote"},
		{[]string{"note", "list", "--help"}, "list"},
		{[]string{"review", "promotions", "--help"}, "promotions"},
		{[]string{"review", "weekly", "--help"}, "weekly"},
		{[]string{"commit", "lint", "--help"}, "lint"},
		{[]string{"release", "propose", "--help"}, "propose"},
		{[]string{"release", "approve", "--help"}, "approve"},
		{[]string{"hooks", "install", "--help"}, "install"},
		{[]string{"optimize", "recommend", "--help"}, "recommend"},
	}

	for _, tc := range commands {
		name := strings.Join(tc.args[:len(tc.args)-1], " ")
		t.Run(name, func(t *testing.T) {
			out, err := executeCommand(tc.args...)
			if err != nil {
				t.Fatalf("command %v: %v", tc.args, err)
			}
			if !strings.Contains(out, tc.contain) {
				t.Errorf("expected output to contain %q, got: %s", tc.contain, out)
			}
		})
	}
}

func TestCommandFlags(t *testing.T) {
	// Verify key command-specific flags are registered
	checks := []struct {
		cmd  string
		flag string
	}{
		{"init", "force"},
		{"init", "dry-run"},
		{"init", "preset"},
		{"check", "fix"},
		{"check", "strict-schema"},
		{"gc", "days"},
		{"gc", "dry-run"},
		{"changelog", "since"},
		{"changelog", "next"},
	}

	for _, tc := range checks {
		t.Run(tc.cmd+"/"+tc.flag, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tc.cmd})
			if err != nil {
				t.Fatalf("find %s: %v", tc.cmd, err)
			}
			if cmd.Flags().Lookup(tc.flag) == nil {
				t.Errorf("missing flag %q on %s", tc.flag, tc.cmd)
			}
		})
	}
}

func TestSubcommandFlags(t *testing.T) {
	checks := []struct {
		parent string
		sub    string
		flag   string
	}{
		{"policy", "gate", "pr"},
		{"security", "scan", "secrets"},
		{"security", "scan", "report-format"},
		{"context", "pack", "profile"},
		{"context", "pack", "changed"},
		{"context", "pack", "stdout"},
		{"context", "budget", "max-tokens"},
		{"note", "new", "type"},
		{"note", "list", "status"},
		{"commit", "lint", "from"},
		{"release", "propose", "since"},
		{"release", "approve", "proposal-id"},
		{"hooks", "install", "pre-commit"},
		{"optimize", "recommend", "window"},
	}

	for _, tc := range checks {
		name := tc.parent + " " + tc.sub + "/" + tc.flag
		t.Run(name, func(t *testing.T) {
			parentCmd, _, _ := rootCmd.Find([]string{tc.parent})
			subCmd, _, err := parentCmd.Find([]string{tc.sub})
			if err != nil {
				t.Fatalf("find %s %s: %v", tc.parent, tc.sub, err)
			}
			if subCmd.Flags().Lookup(tc.flag) == nil {
				t.Errorf("missing flag %q on %s %s", tc.flag, tc.parent, tc.sub)
			}
		})
	}
}

func TestCompletionCommand(t *testing.T) {
	out, err := executeCommand("completion", "bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "bash") {
		t.Error("expected bash completion output")
	}
}

func TestCommandCount(t *testing.T) {
	// Count top-level commands (should include all wired commands + version)
	cmds := rootCmd.Commands()
	if len(cmds) < 20 {
		names := make([]string, 0, len(cmds))
		for _, c := range cmds {
			names = append(names, c.Name())
		}
		t.Errorf("expected >= 20 top-level commands, got %d: %v", len(cmds), names)
	}
}
