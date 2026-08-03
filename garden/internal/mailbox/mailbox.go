package mailbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dashimaki/garden/internal/evolution"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DirectionInbox  = "inbox"
	DirectionOutbox = "outbox"

	StateReceived   = "received"
	StateEvaluating = "evaluating"
	StateApproved   = "approved"
	StateRejected   = "rejected"
	StateQueued     = "queued"
	StateSending    = "sending"
	StateAcked      = "acked"
	StateFailed     = "failed"
	StateDeadLetter = "dead_letter"

	DefaultMaxRetries = 3
)

var (
	ErrNotFound          = errors.New("mailbox item not found")
	ErrIllegalTransition = errors.New("illegal mailbox state transition")
	ErrPrivacyGate       = errors.New("privacy gate blocked outbound item")
)

var legalTransitions = map[string]map[string]map[string]bool{
	DirectionInbox: {
		StateReceived:   {StateEvaluating: true},
		StateEvaluating: {StateApproved: true, StateRejected: true},
	},
	DirectionOutbox: {
		StateQueued:  {StateSending: true, StateDeadLetter: true},
		StateSending: {StateAcked: true, StateFailed: true},
		StateFailed:  {StateQueued: true, StateDeadLetter: true},
	},
}

type Item struct {
	ID           string                   `json:"id"`
	Direction    string                   `json:"direction"`
	State        string                   `json:"state"`
	Payload      map[string]any           `json:"payload"`
	EvidenceRefs []string                 `json:"evidence_refs"`
	Leakage      *evolution.LeakageReport `json:"leakage,omitempty"`
	Reason       string                   `json:"reason,omitempty"`
	RetryCount   int                      `json:"retry_count"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

// Gate screens outbound payloads before they leave the local boundary.
type Gate interface {
	Check(payload map[string]any, evidenceRefs []string) (evolution.LeakageReport, error)
}

// EvolutionGate is the default mechanical privacy gate (ADR-0007 §4).
type EvolutionGate struct{}

func (EvolutionGate) Check(payload map[string]any, evidenceRefs []string) (evolution.LeakageReport, error) {
	return evolution.CheckOutbound(payload, evidenceRefs)
}

// Delivery is the local delivery hook for outbox items; no network transport
// exists in this batch (ADR-0007 §5).
type Delivery func(ctx context.Context, item Item) error

type Store struct {
	db         *sql.DB
	Gate       Gate
	Delivery   Delivery
	MaxRetries int
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS mailbox_items(
 id TEXT PRIMARY KEY, direction TEXT NOT NULL, state TEXT NOT NULL,
 payload_json TEXT NOT NULL, evidence_refs_json TEXT NOT NULL,
 leakage_json TEXT NOT NULL DEFAULT '', reason TEXT NOT NULL DEFAULT '',
 retry_count INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS mailbox_state ON mailbox_items(direction,state,updated_at);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, Gate: EvolutionGate{}, MaxRetries: DefaultMaxRetries}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// AddInbox records an inbound evidence bundle in state received.
func (s *Store) AddInbox(ctx context.Context, payload map[string]any, evidenceRefs []string) (Item, error) {
	return s.add(ctx, DirectionInbox, StateReceived, payload, evidenceRefs)
}

// QueueOutbox records an outbound item in state queued.
func (s *Store) QueueOutbox(ctx context.Context, payload map[string]any, evidenceRefs []string) (Item, error) {
	return s.add(ctx, DirectionOutbox, StateQueued, payload, evidenceRefs)
}

func (s *Store) add(ctx context.Context, direction, state string, payload map[string]any, evidenceRefs []string) (Item, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	if evidenceRefs == nil {
		evidenceRefs = []string{}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Item{}, err
	}
	refsJSON, err := json.Marshal(evidenceRefs)
	if err != nil {
		return Item{}, err
	}
	now := time.Now().UTC()
	item := Item{
		ID:           "mbx_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		Direction:    direction,
		State:        state,
		Payload:      payload,
		EvidenceRefs: evidenceRefs,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO mailbox_items(id,direction,state,payload_json,evidence_refs_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
		item.ID, direction, state, payloadJSON, refsJSON, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return Item{}, err
	}
	return item, nil
}

func (s *Store) Get(ctx context.Context, id string) (Item, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,direction,state,payload_json,evidence_refs_json,leakage_json,reason,retry_count,created_at,updated_at FROM mailbox_items WHERE id=?`, id)
	return scanItem(row)
}

// List returns items for a direction, optionally filtered by state.
func (s *Store) List(ctx context.Context, direction, state string) ([]Item, error) {
	query := `SELECT id,direction,state,payload_json,evidence_refs_json,leakage_json,reason,retry_count,created_at,updated_at FROM mailbox_items WHERE direction=?`
	args := []any{direction}
	if state != "" {
		query += ` AND state=?`
		args = append(args, state)
	}
	query += ` ORDER BY updated_at DESC, id DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListDeadLetter returns dead-lettered items across both directions.
func (s *Store) ListDeadLetter(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,direction,state,payload_json,evidence_refs_json,leakage_json,reason,retry_count,created_at,updated_at FROM mailbox_items WHERE state=? ORDER BY updated_at DESC, id DESC`, StateDeadLetter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Transition moves an item to a legally reachable state.
func (s *Store) Transition(ctx context.Context, id, to, reason string) (Item, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return Item{}, err
	}
	if !legalTransitions[item.Direction][item.State][to] {
		return Item{}, fmt.Errorf("%w: %s %s -> %s", ErrIllegalTransition, item.Direction, item.State, to)
	}
	return s.update(ctx, item, to, reason)
}

// Review approves or rejects an inbox item, advancing received items to
// evaluating first. Approval is always an explicit action (ADR-0007 §3).
func (s *Store) Review(ctx context.Context, id, decision, reason string) (Item, error) {
	if decision != StateApproved && decision != StateRejected {
		return Item{}, fmt.Errorf("%w: %s", evolution.ErrInvalidDecision, decision)
	}
	item, err := s.Get(ctx, id)
	if err != nil {
		return Item{}, err
	}
	if item.Direction != DirectionInbox {
		return Item{}, fmt.Errorf("%w: review applies to inbox items", ErrIllegalTransition)
	}
	if item.State == StateReceived {
		if item, err = s.update(ctx, item, StateEvaluating, "auto-evaluate on review"); err != nil {
			return Item{}, err
		}
	}
	if !legalTransitions[item.Direction][item.State][decision] {
		return Item{}, fmt.Errorf("%w: %s %s -> %s", ErrIllegalTransition, item.Direction, item.State, decision)
	}
	return s.update(ctx, item, decision, reason)
}

// Dispatch runs the privacy gate and delivers a queued outbox item. Gate
// violations dead-letter the item with the leakage report attached.
func (s *Store) Dispatch(ctx context.Context, id string) (Item, error) {
	item, err := s.Transition(ctx, id, StateSending, "dispatch")
	if err != nil {
		return Item{}, err
	}
	report, gateErr := s.Gate.Check(item.Payload, item.EvidenceRefs)
	if gateErr != nil {
		item.Leakage = &report
		return s.deadLetter(ctx, item, "privacy_gate")
	}
	if s.Delivery == nil {
		return s.update(ctx, item, StateAcked, "local delivery (no transport)")
	}
	if deliveryErr := s.Delivery(ctx, item); deliveryErr != nil {
		item.RetryCount++
		if item.RetryCount > s.MaxRetries {
			return s.deadLetter(ctx, item, "delivery_failed: retries exhausted")
		}
		item, err = s.update(ctx, item, StateFailed, deliveryErr.Error())
		if err != nil {
			return Item{}, err
		}
		return s.update(ctx, item, StateQueued, fmt.Sprintf("retry %d", item.RetryCount))
	}
	return s.update(ctx, item, StateAcked, "delivered")
}

func (s *Store) deadLetter(ctx context.Context, item Item, reason string) (Item, error) {
	item.State = StateDeadLetter
	item.Reason = reason
	item.UpdatedAt = time.Now().UTC()
	leakageJSON := ""
	if item.Leakage != nil {
		b, err := json.Marshal(item.Leakage)
		if err != nil {
			return Item{}, err
		}
		leakageJSON = string(b)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE mailbox_items SET state=?,reason=?,leakage_json=?,retry_count=?,updated_at=? WHERE id=?`,
		item.State, item.Reason, leakageJSON, item.RetryCount, item.UpdatedAt.Format(time.RFC3339Nano), item.ID)
	return item, err
}

func (s *Store) update(ctx context.Context, item Item, to, reason string) (Item, error) {
	item.State = to
	item.Reason = reason
	item.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE mailbox_items SET state=?,reason=?,retry_count=?,updated_at=? WHERE id=?`,
		to, reason, item.RetryCount, item.UpdatedAt.Format(time.RFC3339Nano), item.ID)
	return item, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(row scanner) (Item, error) {
	var item Item
	var payloadJSON, refsJSON, leakageJSON, reason, created, updated string
	err := row.Scan(&item.ID, &item.Direction, &item.State, &payloadJSON, &refsJSON, &leakageJSON, &reason, &item.RetryCount, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
		return Item{}, err
	}
	if err := json.Unmarshal([]byte(refsJSON), &item.EvidenceRefs); err != nil {
		return Item{}, err
	}
	if leakageJSON != "" {
		var report evolution.LeakageReport
		if err := json.Unmarshal([]byte(leakageJSON), &report); err != nil {
			return Item{}, err
		}
		item.Leakage = &report
	}
	item.Reason = reason
	item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return item, nil
}
