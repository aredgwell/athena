// Package optimize implements bounded recommendation generation from telemetry.
package optimize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aredgwell/athena/internal/config"
	"github.com/aredgwell/athena/internal/telemetry"
)

// Proposal is a single optimization recommendation.
type Proposal struct {
	ProposalID              string      `json:"proposal_id"`
	Target                  string      `json:"target"`
	Current                 interface{} `json:"current"`
	Recommended             interface{} `json:"recommended"`
	ProjectedTokenReduction float64     `json:"projected_token_reduction"`
	Confidence              float64     `json:"confidence"`
	SampleCount             int         `json:"sample_count"`
}

// RecommendOptions controls optimization analysis.
type RecommendOptions struct {
	WindowDays   int
	MinSamples   int
	ProposalPath string
}

// RecommendResult holds the outcome of the optimize recommend command.
type RecommendResult struct {
	OK         bool       `json:"ok"`
	WindowDays int        `json:"window_days"`
	Proposals  []Proposal `json:"proposals"`
}

// Recommend analyzes telemetry records and generates bounded optimization proposals.
func Recommend(records []telemetry.Record, cfg config.OptimizeConfig, opts RecommendOptions) (*RecommendResult, error) {
	windowDays := opts.WindowDays
	if windowDays == 0 {
		windowDays = cfg.WindowDays
	}
	if windowDays == 0 {
		windowDays = 30
	}

	minSamples := opts.MinSamples
	if minSamples == 0 {
		minSamples = cfg.MinSamples
	}
	if minSamples == 0 {
		minSamples = 5
	}

	// Filter records within window
	cutoff := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour)
	var windowed []telemetry.Record
	for _, r := range records {
		if r.Timestamp.After(cutoff) {
			windowed = append(windowed, r)
		}
	}

	var proposals []Proposal

	// Analyze context compression efficiency
	if p := analyzeCompression(windowed, minSamples); p != nil {
		proposals = append(proposals, *p)
	}

	// Analyze GC window effectiveness
	if p := analyzeGCWindow(windowed, minSamples); p != nil {
		proposals = append(proposals, *p)
	}

	// Analyze policy threshold tuning
	if p := analyzePolicyThresholds(windowed, minSamples); p != nil {
		proposals = append(proposals, *p)
	}

	result := &RecommendResult{
		OK:         true,
		WindowDays: windowDays,
		Proposals:  proposals,
	}

	// Persist proposals if path configured
	proposalPath := opts.ProposalPath
	if proposalPath == "" {
		proposalPath = cfg.ProposalPath
	}
	if proposalPath != "" && len(proposals) > 0 {
		if err := saveProposals(result, proposalPath); err != nil {
			return result, err
		}
	}

	return result, nil
}

func analyzeCompression(records []telemetry.Record, minSamples int) *Proposal {
	var contextPacks []telemetry.Record
	for _, r := range records {
		if r.Command == "context pack" {
			contextPacks = append(contextPacks, r)
		}
	}

	if len(contextPacks) < minSamples {
		return nil
	}

	// Compute average token usage
	var totalTokens int
	for _, r := range contextPacks {
		totalTokens += r.TotalTokens
	}
	avgTokens := totalTokens / len(contextPacks)

	// If avg tokens exceed a threshold, recommend compression change
	if avgTokens > 5000 {
		return &Proposal{
			ProposalID:              fmt.Sprintf("opt_%s_compress", time.Now().Format("20060102")),
			Target:                  "context.profiles.review.compress",
			Current:                 false,
			Recommended:             true,
			ProjectedTokenReduction: 0.18,
			Confidence:              computeConfidence(len(contextPacks), minSamples),
			SampleCount:             len(contextPacks),
		}
	}

	return nil
}

func analyzeGCWindow(records []telemetry.Record, minSamples int) *Proposal {
	var gcRuns []telemetry.Record
	for _, r := range records {
		if r.Command == "gc" {
			gcRuns = append(gcRuns, r)
		}
	}

	if len(gcRuns) < minSamples {
		return nil
	}

	// If GC runs are frequent but mark few notes, suggest widening window
	var avgTime int64
	for _, r := range gcRuns {
		avgTime += r.ExecutionTimeMS
	}
	avgTime /= int64(len(gcRuns))

	if avgTime < 50 && len(gcRuns) > 10 {
		return &Proposal{
			ProposalID:              fmt.Sprintf("opt_%s_gcwin", time.Now().Format("20060102")),
			Target:                  "gc.days",
			Current:                 45,
			Recommended:             60,
			ProjectedTokenReduction: 0.05,
			Confidence:              computeConfidence(len(gcRuns), minSamples),
			SampleCount:             len(gcRuns),
		}
	}

	return nil
}

func analyzePolicyThresholds(records []telemetry.Record, minSamples int) *Proposal {
	var policyFailures int
	var totalChecks int

	for _, r := range records {
		if r.Command == "check" || r.Command == "policy gate" {
			totalChecks++
			if r.ExitCode != 0 {
				policyFailures++
			}
		}
	}

	if totalChecks < minSamples {
		return nil
	}

	failRate := float64(policyFailures) / float64(totalChecks)

	// If failure rate is very high, suggest policy adjustment
	if failRate > 0.5 {
		return &Proposal{
			ProposalID:              fmt.Sprintf("opt_%s_policy", time.Now().Format("20060102")),
			Target:                  "policy.default",
			Current:                 "strict",
			Recommended:             "standard",
			ProjectedTokenReduction: 0.10,
			Confidence:              computeConfidence(totalChecks, minSamples),
			SampleCount:             totalChecks,
		}
	}

	return nil
}

func computeConfidence(samples, minSamples int) float64 {
	if samples < minSamples {
		return 0
	}
	// Confidence grows with sample count, caps at 0.95
	confidence := 0.5 + float64(samples-minSamples)*0.05
	if confidence > 0.95 {
		confidence = 0.95
	}
	return confidence
}

func saveProposals(result *RecommendResult, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
