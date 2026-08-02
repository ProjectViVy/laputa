package evolution

import "time"

type EvolutionEvidenceBundle struct {
	BundleID       string       `json:"bundle_id"`
	Trigger        string       `json:"trigger"`
	Outcome        string       `json:"outcome"`
	TraceRef       string       `json:"trace_ref"`
	BlastRadius    string       `json:"blast_radius"`
	EvidenceRefs   []string     `json:"evidence_refs"`
	SourceRevision string       `json:"source_revision"`
	ContentHashes  []string     `json:"content_hashes"`
	Policy         BundlePolicy `json:"policy"`
	CreatedAt      time.Time    `json:"created_at"`
}

type BundlePolicy struct {
	PrivacyLevel       string   `json:"privacy_level"`
	AllowedScopes      []string `json:"allowed_scopes"`
	PublicationAllowed bool     `json:"publication_allowed"`
}

type GeneCandidate struct {
	CandidateID  string         `json:"candidate_id"`
	RunID        string         `json:"run_id"`
	Kind         string         `json:"kind"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Payload      map[string]any `json:"payload"`
	EvidenceRefs []string       `json:"evidence_refs"`
	TraceRef     string         `json:"trace_ref"`
	Confidence   float64        `json:"confidence"`
	CreatedAt    time.Time      `json:"created_at"`
}

type EvolutionProposal struct {
	ProposalID    string        `json:"proposal_id"`
	RunID         string        `json:"run_id"`
	CandidateID   string        `json:"candidate_id"`
	Kind          string        `json:"kind"`
	Status        string        `json:"status"`
	Summary       string        `json:"summary"`
	LeakageReport LeakageReport `json:"leakage_report"`
	Reviewer      string        `json:"reviewer,omitempty"`
	ReviewNote    string        `json:"review_note,omitempty"`
	ReviewedAt    *time.Time    `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

type EvolutionRun struct {
	RunID       string     `json:"run_id"`
	Status      string     `json:"status"`
	BundleID    string     `json:"bundle_id"`
	Provider    string     `json:"provider"`
	Candidates  []string   `json:"candidates,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type EvolutionEvent struct {
	EventID    string    `json:"event_id"`
	RunID      string    `json:"run_id,omitempty"`
	ProposalID string    `json:"proposal_id,omitempty"`
	Type       string    `json:"type"`
	Actor      string    `json:"actor"`
	Detail     string    `json:"detail"`
	Timestamp  time.Time `json:"timestamp"`
}

type PortableSkill struct {
	SkillID    string         `json:"skill_id"`
	ProposalID string         `json:"proposal_id"`
	Name       string         `json:"name"`
	Version    string         `json:"version"`
	Spec       map[string]any `json:"spec"`
	CreatedAt  time.Time      `json:"created_at"`
}

type HostArtifact struct {
	ArtifactID    string    `json:"artifact_id"`
	SkillID       string    `json:"skill_id"`
	Host          string    `json:"host"`
	Status        string    `json:"status"`
	ValidationRef string    `json:"validation_ref"`
	RollbackRef   string    `json:"rollback_ref"`
	CreatedAt     time.Time `json:"created_at"`
}
