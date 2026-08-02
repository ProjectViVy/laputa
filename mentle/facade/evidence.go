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

	now := time.Now()
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
		excerpt := truncateRunes(memory.Content, budget)
		hash := sha256.Sum256([]byte(memory.Content))
		sourceURI := ""
		if v, ok := memory.Metadata["source_uri"].(string); ok {
			sourceURI = v
		}
		fragments = append(fragments, EvidenceFragment{
			CardID:       memory.ID,
			MaterialRef:  fmt.Sprintf("mem://%s@v%d", memory.ID, memory.Version),
			SourceURI:    sourceURI,
			SourceRev:    fmt.Sprintf("%d", memory.Version),
			Excerpt:      excerpt,
			StartOffset:  0,
			EndOffset:    len([]rune(excerpt)),
			ContentHash:  hex.EncodeToString(hash[:]),
			Validity:     evidenceValidity(memory, now),
			EvidenceRefs: nonNil(memory.Supersedes),
		})
		used += len([]rune(excerpt))
	}
	return fragments, nil
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
