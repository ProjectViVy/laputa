package evolution

import (
	"context"
	"errors"
	"time"
)

var ErrProviderUnavailable = errors.New("evolution: provider unavailable")

type EvolverProvider interface {
	Name() string
	StartRun(ctx context.Context, bundle EvolutionEvidenceBundle) (string, error)
	PollRun(ctx context.Context, runID string) (RunStatus, error)
	Candidates(ctx context.Context, runID string) ([]GeneCandidate, error)
}

type RunStatus struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type ProviderLimits struct {
	WorkingDir      string        `json:"working_dir"`
	MaxEvidenceRefs int           `json:"max_evidence_refs"`
	AllowedEnvVars  []string      `json:"allowed_env_vars"`
	NetworkAccess   bool          `json:"network_access"`
	HubAccess       bool          `json:"hub_access"`
	Timeout         time.Duration `json:"timeout"`
	MaxOutputBytes  int64         `json:"max_output_bytes"`
	MaxLifetime     time.Duration `json:"max_lifetime"`
}

func DefaultProviderLimits() ProviderLimits {
	return ProviderLimits{
		MaxEvidenceRefs: maxEvidenceRefs,
		NetworkAccess:   false,
		HubAccess:       false,
		Timeout:         5 * time.Minute,
		MaxOutputBytes:  1 << 20,
		MaxLifetime:     10 * time.Minute,
	}
}

type NoopProvider struct{}

func (NoopProvider) Name() string { return "noop" }

func (NoopProvider) StartRun(context.Context, EvolutionEvidenceBundle) (string, error) {
	return "", ErrProviderUnavailable
}

func (NoopProvider) PollRun(context.Context, string) (RunStatus, error) {
	return RunStatus{}, ErrProviderUnavailable
}

func (NoopProvider) Candidates(context.Context, string) ([]GeneCandidate, error) {
	return nil, ErrProviderUnavailable
}
