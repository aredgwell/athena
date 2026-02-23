// Package release implements release proposal and approval gate orchestration.
package release

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// GateStatus represents the outcome of a single release gate.
type GateStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "pass", "fail", "skip"
	Detail string `json:"detail,omitempty"`
}

// Proposal is a release proposal artifact.
type Proposal struct {
	ProposalID  string       `json:"proposal_id"`
	NextVersion string       `json:"next_version"`
	SinceTag    string       `json:"since_tag,omitempty"`
	Gates       []GateStatus `json:"gates"`
	CreatedAt   time.Time    `json:"created_at"`
	Fingerprint string       `json:"fingerprint"`
}

// GateFunc runs a single gate check. Returns a GateStatus.
type GateFunc func() GateStatus

// ProposeOptions controls release propose behavior.
type ProposeOptions struct {
	SinceTag    string
	NextVersion string
	GateNames   []string
	StorePath   string // directory for proposal artifacts
}

// ProposeResult holds the outcome of a release propose operation.
type ProposeResult struct {
	OK       bool     `json:"ok"`
	Proposal Proposal `json:"proposal"`
}

// ApproveOptions controls release approve behavior.
type ApproveOptions struct {
	ProposalID string
	StorePath  string
}

// ApproveResult holds the outcome of a release approve operation.
type ApproveResult struct {
	OK          bool   `json:"ok"`
	ProposalID  string `json:"proposal_id"`
	NextVersion string `json:"next_version"`
	Stale       bool   `json:"stale,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

// Service orchestrates release propose/approve operations.
type Service struct {
	gates map[string]GateFunc
}

// NewService creates a release service with the given gate registry.
func NewService(gates map[string]GateFunc) *Service {
	return &Service{gates: gates}
}

// Propose generates a release proposal by running all configured gates.
func (s *Service) Propose(opts ProposeOptions) (*ProposeResult, error) {
	proposalID := fmt.Sprintf("relprop_%s_%s",
		time.Now().UTC().Format("20060102"),
		opts.NextVersion,
	)

	var gateResults []GateStatus
	allPass := true

	for _, name := range opts.GateNames {
		fn, ok := s.gates[name]
		if !ok {
			gateResults = append(gateResults, GateStatus{
				Name:   name,
				Status: "skip",
				Detail: "gate not registered",
			})
			continue
		}
		gs := fn()
		gateResults = append(gateResults, gs)
		if gs.Status == "fail" {
			allPass = false
		}
	}

	fingerprint := computeFingerprint(opts.NextVersion, gateResults)

	proposal := Proposal{
		ProposalID:  proposalID,
		NextVersion: opts.NextVersion,
		SinceTag:    opts.SinceTag,
		Gates:       gateResults,
		CreatedAt:   time.Now().UTC(),
		Fingerprint: fingerprint,
	}

	result := &ProposeResult{
		OK:       allPass,
		Proposal: proposal,
	}

	// Persist proposal artifact
	if opts.StorePath != "" {
		if err := saveProposal(proposal, opts.StorePath); err != nil {
			return result, err
		}
	}

	return result, nil
}

// Approve validates and executes a release proposal with staleness checking.
func (s *Service) Approve(opts ApproveOptions) (*ApproveResult, error) {
	proposal, err := loadProposal(opts.ProposalID, opts.StorePath)
	if err != nil {
		return nil, fmt.Errorf("loading proposal %s: %w", opts.ProposalID, err)
	}

	// Re-run gates to check for staleness
	var currentResults []GateStatus
	for _, gate := range proposal.Gates {
		fn, ok := s.gates[gate.Name]
		if !ok {
			currentResults = append(currentResults, GateStatus{
				Name:   gate.Name,
				Status: gate.Status,
			})
			continue
		}
		currentResults = append(currentResults, fn())
	}

	currentFingerprint := computeFingerprint(proposal.NextVersion, currentResults)

	if currentFingerprint != proposal.Fingerprint {
		return &ApproveResult{
			OK:          false,
			ProposalID:  opts.ProposalID,
			NextVersion: proposal.NextVersion,
			Stale:       true,
			Detail:      "gate results have changed since proposal; re-run release propose",
		}, nil
	}

	// Check all gates pass
	for _, gs := range currentResults {
		if gs.Status == "fail" {
			return &ApproveResult{
				OK:          false,
				ProposalID:  opts.ProposalID,
				NextVersion: proposal.NextVersion,
				Detail:      fmt.Sprintf("gate %s failed", gs.Name),
			}, nil
		}
	}

	return &ApproveResult{
		OK:          true,
		ProposalID:  opts.ProposalID,
		NextVersion: proposal.NextVersion,
	}, nil
}

func computeFingerprint(version string, gates []GateStatus) string {
	data, _ := json.Marshal(struct {
		Version string       `json:"version"`
		Gates   []GateStatus `json:"gates"`
	}{version, gates})
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h[:8])
}

func saveProposal(p Proposal, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, p.ProposalID+".json")
	return os.WriteFile(path, data, 0o644)
}

func loadProposal(id, dir string) (*Proposal, error) {
	path := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Proposal
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
