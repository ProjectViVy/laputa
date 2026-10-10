package ingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"github.com/ProjectViVy/laputa/laputa/actmem"
	"github.com/ProjectViVy/laputa/laputa/evolution"
)

// CaptureActivity is trusted host-only source metadata. It never represents
// ACTMEM authority and is omitted from transport JSON request decoding.
type CaptureActivity struct {
	Phase    string `json:"phase"`
	UserText string `json:"user_text"`
}

// ActivityProjection contains operation receipts, never a second copy of the
// authoritative Markdown entries. Read their bodies only through ACTMEM.
type ActivityProjection struct {
	Status   string   `json:"status"`
	Revision uint64   `json:"revision,omitempty"`
	Entries  []string `json:"entries,omitempty"`
	Error    string   `json:"error,omitempty"`
}

type activityIntent struct {
	Schema string                `json:"schema"`
	Source CaptureActivity       `json:"source"`
	Before actmem.AppendSnapshot `json:"before"`
	ActivityProjection
}

func initialActivity(value *CaptureActivity) (string, error) {
	if value == nil {
		return "", nil
	}
	if value.Phase != "completed" && value.Phase != "failed" && value.Phase != "canceled" {
		return "", errors.New("invalid terminal activity phase")
	}
	if strings.TrimSpace(value.UserText) == "" {
		return "", errors.New("activity requires admitted user content")
	}
	body := html.EscapeString(strings.ReplaceAll(strings.ReplaceAll(value.UserText, "\r\n", "\n"), "\r", "\n"))
	body = "User said:\n> " + strings.ReplaceAll(body, "\n", "\n> ")
	runes := []rune(body)
	if len(runes) > actmem.RecapItemCapChars {
		body = string(runes[:actmem.RecapItemCapChars])
	}
	encoded, err := json.Marshal(activityIntent{Schema: "garden.capture-activity/v1", Source: CaptureActivity{Phase: value.Phase, UserText: body}, ActivityProjection: ActivityProjection{Status: "pending"}})
	return string(encoded), err
}

func initializeActivityColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(ingestions)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var id, notNull, primary int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&id, &name, &kind, &notNull, &defaultValue, &primary); err != nil {
			_ = rows.Close()
			return err
		}
		found = found || name == "activity_json"
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	if found {
		return nil
	}
	// Existing accepted rows explicitly remain not_requested; no source/body
	// backfill or hidden old-effect mutation accompanies this metadata column.
	_, err = db.Exec(`ALTER TABLE ingestions ADD COLUMN activity_json TEXT NOT NULL DEFAULT ''`)
	return err
}

func (s *Service) loadActivity(ctx context.Context, id string) (activityIntent, uint64, string, string, string, error) {
	var raw, session, event, workspace string
	var seq uint64
	err := s.db.QueryRowContext(ctx, `SELECT rowid,session_id,event_id,workspace,activity_json FROM ingestions WHERE ingestion_id=?`, id).Scan(&seq, &session, &event, &workspace, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	if err != nil {
		return activityIntent{}, 0, "", "", "", err
	}
	if raw == "" {
		return activityIntent{ActivityProjection: ActivityProjection{Status: "not_requested"}}, seq, session, event, workspace, nil
	}
	var intent activityIntent
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&intent); err != nil {
		return intent, 0, "", "", "", err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return intent, 0, "", "", "", errors.New("trailing captured activity intent data")
	}
	switch intent.Status {
	case "pending", "applying", "applied", "unknown":
	default:
		return intent, 0, "", "", "", errors.New("invalid captured activity status")
	}
	if intent.Schema != "garden.capture-activity/v1" {
		return intent, 0, "", "", "", errors.New("unknown captured activity intent schema")
	}
	return intent, seq, session, event, workspace, nil
}

func (s *Service) saveActivity(ctx context.Context, id string, intent activityIntent) error {
	raw, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE ingestions SET activity_json=? WHERE ingestion_id=?`, string(raw), id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) ActivityProjection(ctx context.Context, id string) (ActivityProjection, error) {
	intent, _, _, _, _, err := s.loadActivity(ctx, id)
	return intent.ActivityProjection, err
}

func (s *Service) projectedPair(intent activityIntent, seq uint64, session, event, workspace string) []evolution.Entry {
	scope := evolution.Scope{SubjectID: s.ProfileID, Kind: evolution.ScopePersonal}
	if workspace != "" {
		scope.Kind = evolution.ScopeWorkspace
		scope.WorkspaceID = workspace
	}
	refs := []evolution.SourceRef{{SourceID: "garden.ingest", RecordID: fmt.Sprint(seq), Scope: scope}}
	return []evolution.Entry{
		{Section: evolution.SectionPulse, Scope: scope, SessionID: session, EventID: event, Body: "Primary run " + intent.Source.Phase, Sources: refs},
		{Section: evolution.SectionRecap, Scope: scope, SessionID: session, EventID: event, Body: intent.Source.UserText, Sources: refs},
	}
}

func (s *Service) finishActivity(ctx context.Context, id string, intent activityIntent, result evolution.ActivityResult) error {
	intent.Status = "applied"
	intent.Error = ""
	intent.Revision = intent.Before.Revision + 1
	intent.Entries = nil
	for _, entry := range result.Entries {
		intent.Entries = append(intent.Entries, entry.ID)
	}
	intent.Source.UserText = "" // operation receipt is not another ACTMEM authority
	receiptCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.saveActivity(receiptCtx, id, intent)
}

func (s *Service) projectActivity(ctx context.Context, id string) error {
	intent, seq, session, event, workspace, err := s.loadActivity(ctx, id)
	if err != nil || intent.Status == "not_requested" || intent.Status == "applied" {
		return err
	}
	if s.Actmem == nil {
		return errors.New("native ACTMEM projection unavailable")
	}
	pair := s.projectedPair(intent, seq, session, event, workspace)
	if intent.Status == "applying" || intent.Status == "unknown" {
		result, found, err := s.Actmem.LookupCaptured(pair)
		if err != nil {
			return err
		}
		if found {
			return s.finishActivity(ctx, id, intent, result)
		}
		current, err := s.Actmem.AppendSnapshot()
		if err != nil {
			return err
		}
		if current != intent.Before {
			intent.Status = "unknown"
			intent.Error = "actmem_recovery_required"
			if err := s.saveActivity(ctx, id, intent); err != nil {
				return err
			}
			return errors.New("captured activity outcome remains unknown")
		}
		// Exact unchanged canonical pre-write head proves this original append
		// did not commit; retry its original event/sources, never a fresh identity.
		intent.Status = "pending"
	}
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		before, err := s.Actmem.AppendSnapshot()
		if err != nil {
			return err
		}
		intent.Before = before
		intent.Status = "applying"
		intent.Error = ""
		if err := s.saveActivity(ctx, id, intent); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		result, err := s.Actmem.AppendCaptured(pair, before)
		if err == nil {
			return s.finishActivity(ctx, id, intent, result)
		}
		if actmem.CodeOf(err) == "actmem_revision_conflict" {
			intent.Status = "pending"
			if err := s.saveActivity(ctx, id, intent); err != nil {
				return err
			}
			continue // explicit no-effect CAS rejection
		}
		intent.Status = "unknown"
		intent.Error = actmem.CodeOf(err)
		receiptCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		saveErr := s.saveActivity(receiptCtx, id, intent)
		cancel()
		if saveErr != nil {
			return saveErr
		}
		return err
	}
	return errors.New("captured activity head remained busy")
}

// drainActivity is serialized by the native ingest worker. An unresolved
// earlier operation fences later appends and watermark admission.
func (s *Service) drainActivity(ctx context.Context) error {
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT ingestion_id FROM ingestions WHERE activity_json<>'' AND json_extract(activity_json,'$.status')<>'applied' ORDER BY rowid`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.projectActivity(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// ArchiveSession is called by the admitted host after terminal delivery has
// drained and producer admission is sealed. It never invents source captures.
func (s *Service) ArchiveSession(ctx context.Context, session string) error {
	if s.Actmem == nil {
		return errors.New("native ACTMEM archive unavailable")
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.activityMu.Lock()
		var pending int
		err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ingestions WHERE session_id=? AND activity_json<>'' AND COALESCE(json_extract(activity_json,'$.status'),'unknown')<>'applied'`, session).Scan(&pending)
		if err == nil && pending == 0 {
			_, err = s.Actmem.FoldSession(session)
		}
		s.activityMu.Unlock()
		if err != nil {
			return err
		}
		if pending == 0 {
			return nil
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-s.ctx.Done():
			timer.Stop()
			return s.ctx.Err()
		case <-timer.C:
		}
	}
}
