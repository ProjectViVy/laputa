package ingest

import (
	"context"
	"time"

	"github.com/dashimaki/mentle/facade"
)

func (s *Service) DrainSpool(ctx context.Context) (int, error) {
	if s.Spool == nil || s.memory == nil {
		return 0, nil
	}
	pending, err := s.Spool.Pending(ctx)
	if err != nil {
		return 0, err
	}
	drained := 0
	for _, entry := range pending {
		_, err := s.memory.CreateMemory(ctx, facade.CreateMemoryRequest{
			Content:  entry.Content,
			Kind:     entry.Kind,
			Source:   facade.MemorySource{Type: "session", SessionID: entry.SessionID, EventID: entry.EventID},
			Metadata: map[string]any{"content_hash": entry.ContentHash, "lifecycle": "stm", "collection": "working"},
		}, "session:"+entry.EventID, entry.ContentHash)
		if err != nil {
			break
		}
		if err := s.Spool.MarkDrained(ctx, entry.EventID, entry.ContentHash); err != nil {
			break
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='completed',error=NULL,updated_at=? WHERE event_id=? AND status='spooled'`, now, entry.EventID)
		drained++
	}
	return drained, nil
}
