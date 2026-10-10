package facade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type EvidenceFragment struct {
	CardID       string   `json:"card_id"`
	MaterialRef  string   `json:"material_ref"`
	SourceURI    string   `json:"source_uri,omitempty"`
	SourceRev    string   `json:"source_rev,omitempty"`
	Excerpt      string   `json:"excerpt"`
	StartOffset  int      `json:"start_offset"`
	EndOffset    int      `json:"end_offset"`
	ContentHash  string   `json:"content_hash"`
	Validity     string   `json:"validity"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type EvidenceQuery struct {
	CardIDs       []string `json:"card_ids"`
	PerItemBudget int      `json:"per_item_budget,omitempty"`
	TotalBudget   int      `json:"total_budget,omitempty"`
}

const (
	defaultPerItemBudget = 800
	defaultTotalBudget   = 4000
)

func (s *Service) ReadEvidence(ctx context.Context, q EvidenceQuery) ([]EvidenceFragment, error) {
	if s.Catalog == nil {
		return nil, ErrUnavailable
	}
	if len(q.CardIDs) == 0 {
		return []EvidenceFragment{}, nil
	}
	perItem := q.PerItemBudget
	if perItem <= 0 {
		perItem = defaultPerItemBudget
	}
	total := q.TotalBudget
	if total <= 0 {
		total = defaultTotalBudget
	}

	fragments := make([]EvidenceFragment, 0, len(q.CardIDs))
	used := 0
	for _, id := range q.CardIDs {
		if used >= total {
			break
		}
		memory, err := s.GetMemory(ctx, id)
		if err != nil {
			continue
		}
		budget := perItem
		if remaining := total - used; remaining < budget {
			budget = remaining
		}
		fragment := RenderMemoryEvidence(memory, budget)
		fragments = append(fragments, fragment)
		used += len([]rune(fragment.Excerpt))
	}
	return fragments, nil
}

// RenderMemoryEvidence derives bounded evidence from one already-read record.
// The caller must admit its scope, status and revision before projection. This
// pure renderer performs no second authority read and grants no read access.
func RenderMemoryEvidence(memory Memory, budget int) EvidenceFragment {
	if budget <= 0 {
		budget = defaultPerItemBudget
	}
	var excerpt string
	startOffset, endOffset := 0, 0
	var hash [sha256.Size]byte
	if start, end, ok := provenanceOffsets(memory.Metadata); ok && start >= 0 && end > start && end <= len(memory.Content) {
		segment := memory.Content[start:end]
		excerpt = truncateRunes(segment, budget)
		startOffset = start
		endOffset = start + len([]byte(excerpt))
		hash = sha256.Sum256([]byte(excerpt))
	} else {
		excerpt = truncateRunes(memory.Content, budget)
		endOffset = len([]rune(excerpt))
		hash = sha256.Sum256([]byte(memory.Content))
	}
	sourceURI := memory.Source.URI
	if sourceURI == "" {
		if v, ok := memory.Metadata["source_uri"].(string); ok {
			sourceURI = v
		}
	}
	sourceRev := memory.Source.Revision
	if sourceRev == "" {
		sourceRev = fmt.Sprintf("%d", memory.Version)
	}
	return EvidenceFragment{
		CardID:       memory.ID,
		MaterialRef:  fmt.Sprintf("mem://%s@v%d", memory.ID, memory.Version),
		SourceURI:    sourceURI,
		SourceRev:    sourceRev,
		Excerpt:      excerpt,
		StartOffset:  startOffset,
		EndOffset:    endOffset,
		ContentHash:  hex.EncodeToString(hash[:]),
		Validity:     evidenceValidity(memory, time.Now()),
		EvidenceRefs: nonNil(memory.Supersedes),
	}
}

func provenanceOffsets(md map[string]any) (int, int, bool) {
	startRaw, okStart := md["start_offset"]
	endRaw, okEnd := md["end_offset"]
	if !okStart || !okEnd {
		return 0, 0, false
	}
	start, okS := toInt(startRaw)
	end, okE := toInt(endRaw)
	if !okS || !okE {
		return 0, 0, false
	}
	return start, end, true
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func evidenceValidity(m Memory, now time.Time) string {
	switch {
	case m.Status == "deleted":
		return "retracted"
	case m.SupersededBy != nil:
		return "superseded"
	case m.ValidTo != nil && m.ValidTo.Before(now):
		return "expired"
	case m.Status == "disputed":
		return "disputed"
	default:
		return "active"
	}
}
