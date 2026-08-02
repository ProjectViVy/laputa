package facade

import (
	"context"
	"math"
	"time"
)

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

type CardQuery struct {
	Text       string   `json:"text"`
	Scope      string   `json:"scope,omitempty"`
	Collection string   `json:"collection,omitempty"`
	Status     []string `json:"status,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Cursor     string   `json:"cursor,omitempty"`
}

type CardPage struct {
	Cards      []MemoryCard `json:"cards"`
	NextCursor *string      `json:"next_cursor,omitempty"`
}

const summaryBudget = 200

func (s *Service) SearchCards(ctx context.Context, q CardQuery) (CardPage, error) {
	if s.Hybrid == nil {
		return CardPage{}, ErrUnavailable
	}
	if q.Text == "" {
		return CardPage{}, nil
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	allowed := map[string]bool{}
	if len(q.Status) == 0 {
		allowed["active"] = true
	} else {
		for _, st := range q.Status {
			allowed[st] = true
		}
	}

	results, err := s.Hybrid.SearchScored(ctx, q.Text, "", "", q.Limit*4)
	if err != nil {
		return CardPage{}, err
	}

	now := time.Now()
	cards := make([]MemoryCard, 0, q.Limit)
	seen := map[string]bool{}
	for _, result := range results {
		d := result.Drawer
		canonicalID, version, ok := physicalCanonicalID(d.ID)
		if !ok || s.Catalog == nil {
			continue
		}
		if seen[canonicalID] {
			continue
		}
		memory, getErr := s.GetMemory(ctx, canonicalID)
		if getErr != nil || memory.Version != version {
			continue
		}
		if !allowed[memory.Status] {
			continue
		}
		if memory.ValidTo != nil && memory.ValidTo.Before(now) {
			continue
		}
		if memory.SupersededBy != nil {
			continue
		}
		if q.Scope != "" && memory.Scope != q.Scope {
			continue
		}
		collection := ""
		if v, ok := memory.Metadata["collection"].(string); ok {
			collection = v
		} else {
			collection = d.Wing
		}
		if q.Collection != "" && collection != q.Collection {
			continue
		}
		seen[canonicalID] = true
		cards = append(cards, MemoryCard{
			ID:             memory.ID,
			Kind:           memory.Kind,
			Collection:     collection,
			Scope:          memory.Scope,
			Title:          cardTitle(memory),
			Summary:        truncateRunes(memory.Content, summaryBudget),
			SourceRef:      memory.Source.Type,
			Revision:       memory.Version,
			Status:         memory.Status,
			ValidFrom:      memory.ValidFrom,
			ValidTo:        memory.ValidTo,
			SupersededBy:   memory.SupersededBy,
			Tags:           nonNil(memory.Tags),
			HeatScore:      heatScore(memory.UpdatedAt, now),
			CandidateScore: math.Min(1, result.Score*60),
		})
		if len(cards) >= q.Limit {
			break
		}
	}
	return CardPage{Cards: cards}, nil
}

func cardTitle(m Memory) string {
	if v, ok := m.Metadata["title"].(string); ok && v != "" {
		return v
	}
	return truncateRunes(m.Content, 60)
}

func heatScore(updated time.Time, now time.Time) float64 {
	age := now.Sub(updated).Hours()
	if age < 0 {
		age = 0
	}
	return math.Round(math.Exp(-age/168)*1000) / 1000
}

func truncateRunes(value string, max int) string {
	r := []rune(value)
	if len(r) <= max {
		return value
	}
	if max <= 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}
