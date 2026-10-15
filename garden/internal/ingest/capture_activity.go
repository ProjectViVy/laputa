package ingest

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math"
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
	Status         string                     `json:"status"`
	Revision       uint64                     `json:"revision,omitempty"`
	Entries        []string                   `json:"entries,omitempty"`
	Error          string                     `json:"error,omitempty"`
	LegacyConsumer *LegacyConsumerDisposition `json:"legacy_consumer,omitempty"`
}

// LegacyConsumerDisposition closes only the incompatible consumer. The original
// effect Status/Before/source/receipt remain independent, including uncertainty.
type LegacyConsumerDisposition struct {
	State          string              `json:"state"`
	Reason         string              `json:"reason"`
	Binding        actmem.RecapBinding `json:"binding"`
	PolicyRevision uint64              `json:"policy_revision"`
	PolicyDigest   string              `json:"policy_digest"`
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
	intent, err := decodeActivity(raw)
	return intent, seq, session, event, workspace, err
}

func decodeActivity(raw string) (activityIntent, error) {
	if raw == "" {
		return activityIntent{ActivityProjection: ActivityProjection{Status: "not_requested"}}, nil
	}
	var intent activityIntent
	if err := evolution.DecodeStrictJSON([]byte(raw), &intent); err != nil {
		return intent, err
	}
	switch intent.Status {
	case "pending", "applying", "applied", "unknown":
	default:
		return intent, errors.New("invalid captured activity status")
	}
	if intent.Schema != "garden.capture-activity/v1" {
		return intent, errors.New("unknown captured activity intent schema")
	}
	if c := intent.LegacyConsumer; c != nil {
		if c.State != "policy_closed" || c.Reason != "recap_v3" || c.Binding.SubjectID == "" || c.Binding.DomainID == "" || c.PolicyRevision == 0 || !validSnapshotHash(c.PolicyDigest) {
			return intent, errors.New("invalid legacy consumer disposition")
		}
	}
	return intent, nil
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
	if intent.LegacyConsumer != nil {
		_, err := s.needsLegacyActivity(ctx, intent)
		return err
	}
	pair := s.projectedPair(intent, seq, session, event, workspace)
	// A healthy legacy snapshot retains the existing path. Only an incompatible
	// path consults the trusted positive C1 policy; generic errors never skip.
	if _, err := s.Actmem.AppendSnapshot(); err != nil {
		closed, closeErr := s.closeLegacyConsumer(ctx, id, intent, pair)
		if closed || closeErr != nil {
			return closeErr
		}
		return err
	}
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
		if closed, closeErr := s.closeLegacyConsumer(ctx, id, intent, pair); closed || closeErr != nil {
			return closeErr
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
	rows, err := s.activityRows(ctx, "WHERE activity_json<>''")
	if err != nil {
		return err
	}
	for _, row := range rows {
		pending, err := s.needsLegacyActivity(ctx, row.intent)
		if err != nil {
			return err
		}
		if !pending {
			continue
		}
		if err := s.projectActivity(ctx, row.id); err != nil {
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
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	rows, err := s.activityRows(ctx, "WHERE session_id=?", session)
	if err != nil {
		return err
	}
	for _, row := range rows {
		pending, err := s.needsLegacyActivity(ctx, row.intent)
		if err != nil {
			return err
		}
		if pending {
			if err := s.projectActivity(ctx, row.id); err != nil {
				return err
			}
		}
	}
	if _, err := s.Actmem.AppendSnapshot(); err != nil {
		head, policyErr := s.legacyPolicy(ctx)
		if policyErr != nil {
			return policyErr
		}
		if head.Revision > 0 {
			return nil
		} // Policy no-op; no capsule/fold is claimed.
		return err
	}
	_, err = s.Actmem.FoldSession(session)
	return err
}

func validSnapshotHash(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == 32 && value == hex.EncodeToString(raw)
}

func (s *Service) legacyPolicy(ctx context.Context) (actmem.RecapHead, error) {
	if s.Actmem == nil || s.RecapBinding == (actmem.RecapBinding{}) {
		return actmem.RecapHead{}, nil
	}
	head, err := s.Actmem.ReadRecap(ctx, s.RecapBinding)
	if actmem.CodeOf(err) == "recap_legacy_nonempty" {
		return actmem.RecapHead{}, nil
	}
	return head, err
}

func (s *Service) closeLegacyConsumer(ctx context.Context, id string, intent activityIntent, pair []evolution.Entry) (bool, error) {
	head, err := s.legacyPolicy(ctx)
	if err != nil || head.Revision == 0 {
		return false, err
	}
	if (intent.Status == "applying" || intent.Status == "unknown") && validSnapshotHash(intent.Before.SHA256) && intent.Before.Revision < math.MaxUint64 {
		result, found, lookupErr := s.Actmem.LookupCaptured(pair)
		if lookupErr == nil && found {
			return true, s.finishActivity(ctx, id, intent, result)
		}
		if lookupErr != nil && actmem.CodeOf(lookupErr) != "actmem_capture_conflict" && actmem.CodeOf(lookupErr) != "actmem_recovery_required" {
			return false, lookupErr
		}
		// Missing/partial/conflicting proof never establishes a negative effect.
		// Close only the consumer and retain every original uncertainty field.
	}
	intent.LegacyConsumer = &LegacyConsumerDisposition{State: "policy_closed", Reason: "recap_v3", Binding: head.Binding, PolicyRevision: head.Revision, PolicyDigest: head.Digest}
	return true, s.saveActivity(ctx, id, intent)
}

func (s *Service) needsLegacyActivity(ctx context.Context, intent activityIntent) (bool, error) {
	if intent.Status == "not_requested" || intent.Status == "applied" {
		return false, nil
	}
	c := intent.LegacyConsumer
	if c == nil {
		return true, nil
	}
	head, err := s.legacyPolicy(ctx)
	if err != nil {
		return true, err
	}
	if head.Revision == 0 || c.Binding != head.Binding {
		return true, errors.New("legacy consumer policy is not confirmed by the current owner")
	}
	proof, err := s.Actmem.ReadRecapRevision(ctx, s.RecapBinding, c.PolicyRevision)
	if err != nil {
		return true, err
	}
	if proof.Digest != c.PolicyDigest {
		return true, errors.New("legacy consumer policy proof differs")
	}
	return false, nil
}

type activityRow struct {
	id, status string
	seq        uint64
	intent     activityIntent
}

// Read one ordered SQLite snapshot, release rows, then enter owner IO. Strict
// native decoding and needsLegacyActivity are shared by every recovery/fence path;
// SQL JSON shortcuts cannot hide malformed or unknown consumer dispositions.
func (s *Service) activityRows(ctx context.Context, where string, args ...any) ([]activityRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT rowid,ingestion_id,status,activity_json FROM ingestions `+where+` ORDER BY rowid`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []activityRow
	for rows.Next() {
		var row activityRow
		var raw string
		if err := rows.Scan(&row.seq, &row.id, &row.status, &raw); err != nil {
			return nil, err
		}
		row.intent, err = decodeActivity(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
