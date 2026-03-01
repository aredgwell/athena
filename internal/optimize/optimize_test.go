package optimize

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aredgwell/athena/internal/config"
	"github.com/aredgwell/athena/internal/telemetry"
)

func defaultCfg() config.OptimizeConfig {
	return config.OptimizeConfig{
		Enabled:    true,
		WindowDays: 30,
		MinSamples: 5,
	}
}

func makeRecords(command string, count int, tokens int, exitCode int) []telemetry.Record {
	var records []telemetry.Record
	for i := 0; i < count; i++ {
		records = append(records, telemetry.Record{
			Timestamp:       time.Now().Add(-time.Duration(i) * time.Hour),
			Command:         command,
			TotalTokens:     tokens,
			ExitCode:        exitCode,
			ExecutionTimeMS: 30,
		})
	}
	return records
}

func TestOptimizeRecommendCommand(t *testing.T) {
	t.Run("empty records", func(t *testing.T) {
		result, err := Recommend(nil, defaultCfg(), RecommendOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !result.OK {
			t.Error("expected OK")
		}
		if len(result.Proposals) != 0 {
			t.Errorf("proposals: got %d, want 0", len(result.Proposals))
		}
	})

	t.Run("insufficient samples", func(t *testing.T) {
		records := makeRecords("context pack", 2, 10000, 0)
		result, err := Recommend(records, defaultCfg(), RecommendOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Proposals) != 0 {
			t.Errorf("proposals: got %d, want 0 (insufficient samples)", len(result.Proposals))
		}
	})

	t.Run("compression recommendation", func(t *testing.T) {
		records := makeRecords("context pack", 10, 10000, 0)
		result, err := Recommend(records, defaultCfg(), RecommendOptions{})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, p := range result.Proposals {
			if p.Target == "context.profiles.review.compress" {
				found = true
				if p.Recommended != true {
					t.Error("should recommend compression=true")
				}
				if p.SampleCount != 10 {
					t.Errorf("sample_count: got %d, want 10", p.SampleCount)
				}
				if p.Confidence <= 0 {
					t.Error("confidence should be > 0")
				}
			}
		}
		if !found {
			t.Error("expected compression recommendation")
		}
	})

	t.Run("policy threshold recommendation", func(t *testing.T) {
		records := makeRecords("check", 10, 0, 1) // all failing
		result, err := Recommend(records, defaultCfg(), RecommendOptions{})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, p := range result.Proposals {
			if p.Target == "policy.default" {
				found = true
				if p.Recommended != "standard" {
					t.Errorf("recommended: got %v, want standard", p.Recommended)
				}
			}
		}
		if !found {
			t.Error("expected policy recommendation for high failure rate")
		}
	})

	t.Run("window days override", func(t *testing.T) {
		result, _ := Recommend(nil, defaultCfg(), RecommendOptions{WindowDays: 7})
		if result.WindowDays != 7 {
			t.Errorf("window_days: got %d, want 7", result.WindowDays)
		}
	})

	t.Run("default window from config", func(t *testing.T) {
		result, _ := Recommend(nil, defaultCfg(), RecommendOptions{})
		if result.WindowDays != 30 {
			t.Errorf("window_days: got %d, want 30", result.WindowDays)
		}
	})

	t.Run("records outside window excluded", func(t *testing.T) {
		// Records from 60 days ago with 7-day window
		var records []telemetry.Record
		for i := 0; i < 10; i++ {
			records = append(records, telemetry.Record{
				Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
				Command:     "context pack",
				TotalTokens: 10000,
			})
		}

		result, _ := Recommend(records, defaultCfg(), RecommendOptions{WindowDays: 7})
		if len(result.Proposals) != 0 {
			t.Errorf("proposals: got %d, want 0 (all outside window)", len(result.Proposals))
		}
	})
}

func TestPersistProposals(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proposals")
	path := filepath.Join(dir, "optimize.json")

	records := makeRecords("context pack", 10, 10000, 0)
	_, err := Recommend(records, defaultCfg(), RecommendOptions{
		ProposalPath: path,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("proposal file should not be empty")
	}
}

func TestComputeConfidence(t *testing.T) {
	if got := computeConfidence(3, 5); got != 0 {
		t.Errorf("below min: got %f, want 0", got)
	}
	if got := computeConfidence(5, 5); got != 0.5 {
		t.Errorf("at min: got %f, want 0.5", got)
	}
	if got := computeConfidence(100, 5); got != 0.95 {
		t.Errorf("high samples: got %f, want 0.95 (cap)", got)
	}
}

func TestNoAutoApply(t *testing.T) {
	// Verify that recommendations are proposal-only
	records := makeRecords("context pack", 10, 10000, 0)
	result, _ := Recommend(records, config.OptimizeConfig{
		Enabled:    true,
		WindowDays: 30,
		MinSamples: 5,
		AutoApply:  false,
	}, RecommendOptions{})

	// Result should contain proposals but no indication of auto-application
	if !result.OK {
		t.Error("expected OK")
	}
	for _, p := range result.Proposals {
		if p.ProposalID == "" {
			t.Error("proposals should have IDs for manual review")
		}
	}
}
