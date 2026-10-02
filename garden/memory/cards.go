package memory

import "time"

// MemoryCard is the discovery-shape read object returned by backend
// search. It lives in this package so the backend contract and adapters
// do not depend on the agent-facing wire package.
type MemoryCard struct {
	ID             string     `json:"id"`
	Kind           string     `json:"kind"`
	Collection     string     `json:"collection"`
	Scope          string     `json:"scope"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	SourceRef      string     `json:"source_ref"`
	Revision       int        `json:"revision"`
	Status         string     `json:"status"`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to,omitempty"`
	SupersededBy   *string    `json:"superseded_by,omitempty"`
	Tags           []string   `json:"tags"`
	HeatScore      float64    `json:"heat_score"`
	LastActivated  *time.Time `json:"last_activated,omitempty"`
	CandidateScore float64    `json:"candidate_score"`
}

// EvidenceFragment carries scope/revision/status in its envelope so
// Garden can verify backend adapter results (contracts.md section 6). The
// added fields are additive-only on the existing wire.
type EvidenceFragment struct {
	CardID       string   `json:"card_id"`
	MaterialRef  string   `json:"material_ref"`
	SourceURI    string   `json:"source_uri,omitempty"`
	SourceRev    string   `json:"source_rev,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	Revision     uint64   `json:"revision,omitempty"`
	Status       string   `json:"status,omitempty"`
	Excerpt      string   `json:"excerpt"`
	StartOffset  int      `json:"start_offset"`
	EndOffset    int      `json:"end_offset"`
	ContentHash  string   `json:"content_hash"`
	Validity     string   `json:"validity"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}
