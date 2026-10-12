package hosts

import (
	"context"
	"fmt"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"strings"
)

type ProjectionRequest struct {
	BindingID string         `json:"binding_id"`
	Kinds     []persona.Kind `json:"kinds,omitempty"`
	Path      string         `json:"path"`
}
type FilePlan struct {
	BindingID       string            `json:"binding_id"`
	Path            string            `json:"path"`
	OriginalHash    string            `json:"original_hash"`
	ManagedHash     string            `json:"managed_hash"`
	ProposedContent string            `json:"proposed_content"`
	AdapterVersion  string            `json:"adapter_version"`
	SourceRevisions map[string]uint64 `json:"source_revisions"`
}

func (s *Service) projection(ctx context.Context, id string, kinds []persona.Kind) (string, map[string]uint64, error) {
	b, err := s.binding(ctx, id, false)
	if err != nil {
		return "", nil, err
	}
	if len(kinds) == 0 {
		kinds = []persona.Kind{persona.KindIdentity, persona.KindRelationship, persona.KindRedline, persona.KindUser}
	}
	seen := map[persona.Kind]bool{}
	for _, kind := range kinds {
		if kind < persona.KindIdentity || kind > persona.KindMission || kind == persona.KindWorld || seen[kind] {
			return "", nil, failure("invalid_request", "invalid or tool-only projection kind")
		}
		seen[kind] = true
	}
	authority, err := s.persona(ctx, b.ProfileID)
	if err != nil {
		return "", nil, err
	}
	revisions := map[string]uint64{}
	var body strings.Builder
	for _, kind := range persona.AllKinds {
		if !seen[kind] {
			continue
		}
		doc, err := authority.GetDocument(kind)
		if err != nil {
			return "", nil, err
		}
		if !doc.Exists || !doc.Valid {
			return "", nil, failure("unavailable", "selected Persona document unavailable")
		}
		revisions[kind.String()] = doc.Revision
		fmt.Fprintf(&body, "## %s\n%s\n", kind.FileName(), doc.Content)
	}
	return body.String(), revisions, nil
}
func (s *Service) Preview(ctx context.Context, req ProjectionRequest) (FilePlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validateTarget(ctx, req.Path); err != nil {
		return FilePlan{}, err
	}
	stateBefore, err := s.state(ctx, req.BindingID)
	if err != nil {
		return FilePlan{}, err
	}
	if stateBefore.Mode == nativeMode {
		return FilePlan{}, failure("conflict", "native binding cannot preview host file writes")
	}
	body, revisions, err := s.projection(ctx, req.BindingID, req.Kinds)
	if err != nil {
		return FilePlan{}, err
	}
	file, _, err := readFile(req.Path)
	if err != nil {
		return FilePlan{}, err
	}
	start, end, err := managedRegion(file, req.BindingID)
	if err != nil {
		return FilePlan{}, err
	}
	block := managedBlock(req.BindingID, body)
	proposed := file + block
	state, err := s.state(ctx, req.BindingID)
	if err != nil {
		return FilePlan{}, err
	}
	if state.Path != "" && state.Path != req.Path {
		return FilePlan{}, failure("conflict", "binding already manages a different path")
	}
	if state.Path != "" {
		if err = s.recoverFile(&state, file); err != nil {
			return FilePlan{}, err
		}
	}
	if start >= 0 && (state.WrittenHash == "" || hash(file[start:end]) != state.WrittenHash) {
		return FilePlan{}, failure("conflict", "managed block changed or not owned by this binding")
	}
	state.SavedRevisions = revisions
	if state.State == "" {
		state.State = "inactive"
	}
	if err = s.save(ctx, state); err != nil {
		return FilePlan{}, err
	}
	if start >= 0 {
		proposed = file[:start] + block + file[end:]
	}
	return FilePlan{BindingID: req.BindingID, Path: req.Path, OriginalHash: hash(file), ManagedHash: hash(block), ProposedContent: proposed, AdapterVersion: "garden-projection/v1", SourceRevisions: revisions}, nil
}
