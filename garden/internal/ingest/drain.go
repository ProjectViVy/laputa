package ingest

import (
	"context"
	"errors"
	"time"

	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/mentle/facade"
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
		if s.CheckIdentity != nil {
			var id string
			if err := s.db.QueryRowContext(ctx, `SELECT ingestion_id FROM ingestions WHERE event_id=? AND content_hash=?`, entry.EventID, entry.ContentHash).Scan(&id); err != nil {
				return drained, err
			}
			identity, err := s.admitted(ctx, id)
			if err != nil || identity.Scope != entry.Scope {
				s.fail(ctx, id, errors.New("capture binding revoked or scope changed"))
				_ = s.Spool.MarkDrained(ctx, entry.EventID, entry.ContentHash)
				continue
			}
		}
		if _, decErr := memory.DecodeScope(entry.Scope); decErr != nil {
			// A spooled entry whose recorded scope cannot be classified
			// must not be re-routed into another scope: fail it visibly and
			// drain the entry so it does not retry forever.
			now := time.Now().UTC().Format(time.RFC3339Nano)
			_, _ = s.db.ExecContext(ctx, `UPDATE ingestions SET status='failed',error=?,updated_at=? WHERE event_id=? AND status='spooled'`, "unclassified spool scope", now, entry.EventID)
			_ = s.Spool.MarkDrained(ctx, entry.EventID, entry.ContentHash)
			continue
		}
		session, event := entry.SessionID, entry.EventID
		var identity Identity
		if s.CheckIdentity != nil {
			var id string
			_ = s.db.QueryRowContext(ctx, `SELECT ingestion_id FROM ingestions WHERE event_id=?`, entry.EventID).Scan(&id)
			identity, _ = s.identity(ctx, id)
			session, event = identity.HostSessionID, identity.HostEventID
		}
		memory, err := s.memory.CreateMemory(ctx, facade.CreateMemoryRequest{
			Content:  entry.Content,
			Kind:     entry.Kind,
			Scope:    entry.Scope,
			Source:   facade.MemorySource{Type: "session", SessionID: session, EventID: event},
			Metadata: map[string]any{"content_hash": entry.ContentHash, "lifecycle": "stm", "collection": "working", "capture_identity": identity},
		}, "session:"+entry.EventID, entry.ContentHash)
		if err != nil {
			return drained, err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := s.db.ExecContext(ctx, `UPDATE ingestions SET status='completed',memory_id=?,error=NULL,updated_at=? WHERE event_id=? AND status='spooled'`, memory.ID, now, entry.EventID); err != nil {
			return drained, err
		}
		// Persist the canonical result before draining the spool. A failed mark
		// leaves a retryable entry rather than losing recovery on a DB error.
		if err := s.Spool.MarkDrained(ctx, entry.EventID, entry.ContentHash); err != nil {
			return drained, err
		}
		drained++
	}
	return drained, nil
}
