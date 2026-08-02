package facade

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type RetrievalQuery struct {
	Text  string
	Wing  string
	Room  string
	Limit int
}

type RetrievalHit struct {
	ID       string            `json:"id"`
	Content  string            `json:"content"`
	Wing     string            `json:"wing"`
	Room     string            `json:"room"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Score    float64           `json:"score"`
	Channels []string          `json:"channels"`
}

type GraphFact struct {
	Predicate  string  `json:"predicate"`
	Object     string  `json:"object"`
	ValidFrom  string  `json:"valid_from,omitempty"`
	ValidTo    string  `json:"valid_to,omitempty"`
	Confidence float64 `json:"confidence"`
}

type TimelineEvent struct {
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	ValidFrom string `json:"valid_from,omitempty"`
	ValidTo   string `json:"valid_to,omitempty"`
}

func (s *Service) Retrieve(ctx context.Context, query RetrievalQuery) ([]RetrievalHit, error) {
	if s.Hybrid == nil {
		return nil, ErrUnavailable
	}
	if query.Text == "" {
		return nil, errors.New("retrieval query is required")
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	results, err := s.Hybrid.SearchScored(ctx, query.Text, query.Wing, query.Room, query.Limit*4)
	if err != nil {
		return nil, err
	}
	hits := make([]RetrievalHit, 0, len(results))
	seen := map[string]bool{}
	for _, result := range results {
		d := result.Drawer
		if canonicalID, version, ok := physicalCanonicalID(d.ID); ok && s.Catalog != nil {
			memory, getErr := s.GetMemory(ctx, canonicalID)
			if getErr != nil || memory.Version != version || seen[canonicalID] {
				continue
			}
			seen[canonicalID] = true
			d.ID = canonicalID
			d.Content = memory.Content
		}
		hits = append(hits, RetrievalHit{ID: d.ID, Content: d.Content, Wing: d.Wing, Room: d.Room, Metadata: d.Metadata, Score: result.Score, Channels: result.Channels})
		if len(hits) >= query.Limit {
			break
		}
	}
	return hits, nil
}

func physicalCanonicalID(id string) (string, int, bool) {
	at := strings.LastIndex(id, "@v")
	if at < 0 || !strings.HasPrefix(id, "mem_") {
		return "", 0, false
	}
	var version int
	if _, err := fmt.Sscanf(id[at+2:], "%d", &version); err != nil {
		return "", 0, false
	}
	return id[:at], version, true
}

func (s *Service) QueryEntity(ctx context.Context, entity, asOf, direction string) ([]GraphFact, error) {
	if s.KG == nil {
		return nil, ErrUnavailable
	}
	results, err := s.KG.QueryEntity(entity, asOf, direction)
	if err != nil {
		return nil, err
	}
	facts := make([]GraphFact, 0, len(results))
	for _, result := range results {
		facts = append(facts, GraphFact{Predicate: result.Predicate, Object: result.Object, ValidFrom: result.ValidFrom, ValidTo: result.ValidTo, Confidence: result.Confidence})
	}
	return facts, nil
}

func (s *Service) Timeline(ctx context.Context, entity string) ([]TimelineEvent, error) {
	if s.KG == nil {
		return nil, ErrUnavailable
	}
	results, err := s.KG.Timeline(entity)
	if err != nil {
		return nil, err
	}
	events := make([]TimelineEvent, 0, len(results))
	for _, result := range results {
		events = append(events, TimelineEvent{Predicate: result.Predicate, Object: result.Object, ValidFrom: result.ValidFrom, ValidTo: result.ValidTo})
	}
	return events, nil
}
