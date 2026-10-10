package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ProjectViVy/laputa/mentle/facade"
)

// SemanticUnit is a provenance-preserving index unit derived from a source
// artifact (ADR-0006 §5). Units are index material only: they never write
// Laputa sections or create authority.
type SemanticUnit struct {
	Content        string    `json:"content"`
	SourcePath     string    `json:"source_path"`
	SourceRef      string    `json:"source_ref,omitempty"`
	HeadingPath    []string  `json:"heading_path,omitempty"`
	StartOffset    int       `json:"start_offset"`
	EndOffset      int       `json:"end_offset"`
	ContentHash    string    `json:"content_hash,omitempty"`
	SourceRevision string    `json:"source_revision,omitempty"`
	ObservedAt     time.Time `json:"observed_at,omitempty"`
	Scope          string    `json:"scope,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
}

type IngestUnitsResult struct {
	Created   int      `json:"created"`
	Skipped   int      `json:"skipped"`
	MemoryIDs []string `json:"memory_ids"`
}

// IngestSemanticUnits persists derived index units idempotently. The
// idempotency key is source_path + content_hash, so re-scanning an unchanged
// source never duplicates units.
func (s *Service) IngestSemanticUnits(ctx context.Context, units []SemanticUnit) (IngestUnitsResult, error) {
	if s.memory == nil {
		return IngestUnitsResult{}, facade.ErrUnavailable
	}
	result := IngestUnitsResult{MemoryIDs: []string{}}
	for _, unit := range units {
		unit.Content = strings.TrimSpace(unit.Content)
		if unit.Content == "" || strings.TrimSpace(unit.SourcePath) == "" {
			return result, errors.New("semantic unit requires content and source_path")
		}
		hash := unit.ContentHash
		if hash == "" {
			sum := sha256.Sum256([]byte(unit.Content))
			hash = "sha256:" + hex.EncodeToString(sum[:])
		}
		observed := unit.ObservedAt
		if observed.IsZero() {
			observed = time.Now().UTC()
		}
		metadata := map[string]any{
			"source_path":  unit.SourcePath,
			"content_hash": hash,
			"start_offset": unit.StartOffset,
			"end_offset":   unit.EndOffset,
			"observed_at":  observed.UTC().Format(time.RFC3339),
			"sync_state":   "current",
			"lifecycle":    "ltm",
			"collection":   "sources",
		}
		if unit.SourceRef != "" {
			metadata["source_ref"] = unit.SourceRef
		}
		if len(unit.HeadingPath) > 0 {
			metadata["heading_path"] = append([]string{}, unit.HeadingPath...)
		}
		if unit.SourceRevision != "" {
			metadata["source_revision"] = unit.SourceRevision
		}
		memory, err := s.memory.CreateMemory(ctx, facade.CreateMemoryRequest{
			Content:  unit.Content,
			Kind:     "semantic_unit",
			Scope:    unit.Scope,
			Tags:     unit.Tags,
			Source:   facade.MemorySource{Type: "import", URI: unit.SourcePath, Revision: unit.SourceRevision},
			Metadata: metadata,
		}, "unit:"+unit.SourcePath+":"+hash, hash)
		if err != nil {
			return result, err
		}
		seen := false
		for _, id := range result.MemoryIDs {
			if id == memory.ID {
				seen = true
				break
			}
		}
		if seen {
			result.Skipped++
		} else {
			result.Created++
			result.MemoryIDs = append(result.MemoryIDs, memory.ID)
		}
	}
	return result, nil
}
