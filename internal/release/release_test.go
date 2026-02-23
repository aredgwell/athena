package release

import (
	"os"
	"path/filepath"
	"testing"
)

func passGate(name string) GateFunc {
	return func() GateStatus {
		return GateStatus{Name: name, Status: "pass"}
	}
}

func failGate(name, detail string) GateFunc {
	return func() GateStatus {
		return GateStatus{Name: name, Status: "fail", Detail: detail}
	}
}

func TestReleaseProposeCommand(t *testing.T) {
	t.Run("all gates pass", func(t *testing.T) {
		svc := NewService(map[string]GateFunc{
			"commit_lint":   passGate("commit_lint"),
			"check":         passGate("check"),
			"security_scan": passGate("security_scan"),
		})

		result, err := svc.Propose(ProposeOptions{
			SinceTag:    "v1.3.0",
			NextVersion: "1.4.0",
			GateNames:   []string{"commit_lint", "check", "security_scan"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.OK {
			t.Error("expected OK when all gates pass")
		}
		if result.Proposal.NextVersion != "1.4.0" {
			t.Errorf("version: got %s, want 1.4.0", result.Proposal.NextVersion)
		}
		if len(result.Proposal.Gates) != 3 {
			t.Errorf("gates: got %d, want 3", len(result.Proposal.Gates))
		}
		for _, g := range result.Proposal.Gates {
			if g.Status != "pass" {
				t.Errorf("gate %s: got %s, want pass", g.Name, g.Status)
			}
		}
	})

	t.Run("gate failure", func(t *testing.T) {
		svc := NewService(map[string]GateFunc{
			"check":         passGate("check"),
			"security_scan": failGate("security_scan", "secrets found"),
		})

		result, err := svc.Propose(ProposeOptions{
			NextVersion: "1.4.0",
			GateNames:   []string{"check", "security_scan"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.OK {
			t.Error("expected not OK when gate fails")
		}
	})

	t.Run("unknown gate skipped", func(t *testing.T) {
		svc := NewService(map[string]GateFunc{})

		result, err := svc.Propose(ProposeOptions{
			NextVersion: "1.0.0",
			GateNames:   []string{"nonexistent"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Proposal.Gates[0].Status != "skip" {
			t.Errorf("status: got %s, want skip", result.Proposal.Gates[0].Status)
		}
	})

	t.Run("persists artifact", func(t *testing.T) {
		storePath := filepath.Join(t.TempDir(), "proposals")
		svc := NewService(map[string]GateFunc{
			"check": passGate("check"),
		})

		result, err := svc.Propose(ProposeOptions{
			NextVersion: "1.0.0",
			GateNames:   []string{"check"},
			StorePath:   storePath,
		})
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(storePath, result.Proposal.ProposalID+".json")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("proposal artifact not found: %v", err)
		}
	})
}

func TestReleaseApproveCommand(t *testing.T) {
	t.Run("approve passing proposal", func(t *testing.T) {
		storePath := filepath.Join(t.TempDir(), "proposals")
		gates := map[string]GateFunc{
			"check":       passGate("check"),
			"commit_lint": passGate("commit_lint"),
		}
		svc := NewService(gates)

		// Propose
		propResult, err := svc.Propose(ProposeOptions{
			NextVersion: "2.0.0",
			GateNames:   []string{"check", "commit_lint"},
			StorePath:   storePath,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Approve
		appResult, err := svc.Approve(ApproveOptions{
			ProposalID: propResult.Proposal.ProposalID,
			StorePath:  storePath,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !appResult.OK {
			t.Errorf("expected OK: %s", appResult.Detail)
		}
		if appResult.NextVersion != "2.0.0" {
			t.Errorf("version: got %s", appResult.NextVersion)
		}
	})

	t.Run("stale proposal detected", func(t *testing.T) {
		storePath := filepath.Join(t.TempDir(), "proposals")

		// Create with passing gates
		svc := NewService(map[string]GateFunc{
			"check": passGate("check"),
		})

		propResult, err := svc.Propose(ProposeOptions{
			NextVersion: "2.0.0",
			GateNames:   []string{"check"},
			StorePath:   storePath,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Now gates have changed (check now fails)
		svc2 := NewService(map[string]GateFunc{
			"check": failGate("check", "new failure"),
		})

		appResult, err := svc2.Approve(ApproveOptions{
			ProposalID: propResult.Proposal.ProposalID,
			StorePath:  storePath,
		})
		if err != nil {
			t.Fatal(err)
		}
		if appResult.OK {
			t.Error("expected not OK for stale proposal")
		}
		if !appResult.Stale {
			t.Error("expected stale=true")
		}
	})

	t.Run("missing proposal", func(t *testing.T) {
		svc := NewService(map[string]GateFunc{})

		_, err := svc.Approve(ApproveOptions{
			ProposalID: "nonexistent",
			StorePath:  t.TempDir(),
		})
		if err == nil {
			t.Error("expected error for missing proposal")
		}
	})

	t.Run("failing gate on approve", func(t *testing.T) {
		storePath := filepath.Join(t.TempDir(), "proposals")

		// Propose with a failing gate (result.OK will be false but proposal is saved)
		failGates := map[string]GateFunc{
			"check": failGate("check", "failing"),
		}
		svc := NewService(failGates)

		propResult, err := svc.Propose(ProposeOptions{
			NextVersion: "2.0.0",
			GateNames:   []string{"check"},
			StorePath:   storePath,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Approve with same failing gates (not stale, but fails)
		appResult, err := svc.Approve(ApproveOptions{
			ProposalID: propResult.Proposal.ProposalID,
			StorePath:  storePath,
		})
		if err != nil {
			t.Fatal(err)
		}
		if appResult.OK {
			t.Error("expected not OK when gates fail")
		}
		if appResult.Stale {
			t.Error("should not be stale (same results)")
		}
	})
}

func TestFingerprint(t *testing.T) {
	gates := []GateStatus{
		{Name: "check", Status: "pass"},
	}
	fp1 := computeFingerprint("1.0.0", gates)
	fp2 := computeFingerprint("1.0.0", gates)

	if fp1 != fp2 {
		t.Error("fingerprint should be deterministic")
	}

	fp3 := computeFingerprint("2.0.0", gates)
	if fp1 == fp3 {
		t.Error("different versions should produce different fingerprints")
	}
}
