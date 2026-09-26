// Package personactx is Garden's narrow, session-frozen consumer of Laputa
// Persona Markdown. It stores only the six bounded projections allowed by
// the architecture; activity memory and the tool-only seventh document have
// no field or route in this package.
package personactx

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dashimaki/garden/internal/sqliteconn"
	"github.com/dashimaki/laputa/persona"
	"github.com/rivo/uniseg"
)

var ErrSessionNotFound = errors.New("personactx: frozen session not found")

// Section is a closed set. No arbitrary authority kind can be placed in a
// FrozenCore value by using this type.
type Section uint8

const (
	SectionIdentity Section = iota
	SectionRelationship
	SectionRedline
	SectionUser
	SectionDream
	SectionDark
)

var sections = [...]Section{
	SectionIdentity,
	SectionRelationship,
	SectionRedline,
	SectionUser,
	SectionDream,
	SectionDark,
}

func (s Section) String() string {
	switch s {
	case SectionIdentity:
		return "identity"
	case SectionRelationship:
		return "relationship"
	case SectionRedline:
		return "redline"
	case SectionUser:
		return "user"
	case SectionDream:
		return "dream"
	case SectionDark:
		return "dark"
	default:
		return "unknown"
	}
}

func (s Section) personaKind() persona.Kind {
	switch s {
	case SectionIdentity:
		return persona.KindIdentity
	case SectionRelationship:
		return persona.KindRelationship
	case SectionRedline:
		return persona.KindRedline
	case SectionUser:
		return persona.KindUser
	case SectionDream:
		return persona.KindDream
	case SectionDark:
		return persona.KindDark
	default:
		return persona.Kind(-1)
	}
}

func frozenLimit(section Section) int {
	limit, _ := section.personaKind().FrozenLimit()
	return limit
}

type FrozenSection struct {
	Section        Section `json:"section"`
	Content        string  `json:"content"`
	SourceRevision uint64  `json:"source_revision"`
	SourceHash     string  `json:"source_hash"`
}

// FrozenCore has exactly six slots and is immutable by convention after
// capture. Garden recall receives a value copy, never a live Persona reader.
type FrozenCore struct {
	SessionID  string           `json:"session_id"`
	CapturedAt time.Time        `json:"captured_at"`
	Sections   [6]FrozenSection `json:"sections"`
}

func (c FrozenCore) Content(section Section) string {
	if int(section) < 0 || int(section) >= len(c.Sections) {
		return ""
	}
	return c.Sections[section].Content
}

func (c FrozenCore) Render(maxChars int) string {
	if maxChars <= 0 {
		maxChars = 4000
	}
	var builder strings.Builder
	for _, section := range c.Sections {
		if section.Content == "" {
			continue
		}
		part := fmt.Sprintf("## Frozen Core — %s\n%s\n", section.Section.String(), section.Content)
		if runeLen(builder.String())+runeLen(part) > maxChars {
			remaining := maxChars - runeLen(builder.String())
			if remaining > 0 {
				builder.WriteString(truncateRunes(part, remaining))
			}
			break
		}
		builder.WriteString(part)
	}
	return builder.String()
}

type Reader interface {
	GetDocument(persona.Kind) (*persona.Document, error)
}

// Capture reads the six source documents once. USER contributes its
// Preferences subsection only, preserving the architecture's projection
// boundary.
func Capture(reader Reader, sessionID string, clock func() time.Time) (FrozenCore, error) {
	if reader == nil {
		return FrozenCore{}, errors.New("personactx: Persona reader is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return FrozenCore{}, errors.New("personactx: session_id is required")
	}
	if clock == nil {
		clock = time.Now
	}
	core := FrozenCore{SessionID: sessionID, CapturedAt: clock().UTC()}
	for index, section := range sections {
		document, err := reader.GetDocument(section.personaKind())
		if err != nil {
			return FrozenCore{}, err
		}
		content := document.Content
		if section == SectionUser {
			if preferences, ok := persona.ExtractMarkdownSection(document.Content, "Preferences"); ok {
				content = preferences
			}
		}
		core.Sections[index] = FrozenSection{
			Section:        section,
			Content:        truncateVisible(content, frozenLimit(section)),
			SourceRevision: document.Revision,
			SourceHash:     document.ContentHash,
		}
	}
	return core, nil
}

// Store persists the first capture for each session in Garden SQLite. The
// INSERT is guarded by a primary key, so a second process cannot replace a
// frozen session with a later Persona revision.
type Store struct {
	db *sql.DB
}

type FrozenProvider interface {
	Get(context.Context, string) (FrozenCore, error)
}

// SessionProvider wires persistent session capture to a Laputa reader. It is
// the only composition-root adapter needed by recall.
type SessionProvider struct {
	Store  *Store
	Reader Reader
}

func (p *SessionProvider) Get(ctx context.Context, sessionID string) (FrozenCore, error) {
	if p == nil || p.Store == nil {
		return FrozenCore{}, errors.New("personactx: session store is unavailable")
	}
	return p.Store.Capture(ctx, sessionID, p.Reader)
}

func OpenStore(path string) (*Store, error) {
	db, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS frozen_core_sessions(
 session_id TEXT PRIMARY KEY,
 captured_at TEXT NOT NULL,
 core_json TEXT NOT NULL
);`)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Capture(ctx context.Context, sessionID string, reader Reader) (FrozenCore, error) {
	if s == nil || s.db == nil {
		return FrozenCore{}, errors.New("personactx: store is unavailable")
	}
	if strings.TrimSpace(sessionID) == "" {
		return FrozenCore{}, errors.New("personactx: session_id is required")
	}
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT core_json FROM frozen_core_sessions WHERE session_id=?`, sessionID).Scan(&raw)
	if err == nil {
		var core FrozenCore
		if unmarshalErr := json.Unmarshal([]byte(raw), &core); unmarshalErr != nil {
			return FrozenCore{}, fmt.Errorf("personactx: corrupted frozen session: %w", unmarshalErr)
		}
		return core, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return FrozenCore{}, err
	}
	core, err := Capture(reader, sessionID, nil)
	if err != nil {
		return FrozenCore{}, err
	}
	rawBytes, err := json.Marshal(core)
	if err != nil {
		return FrozenCore{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO frozen_core_sessions(session_id,captured_at,core_json) VALUES(?,?,?)`, sessionID, core.CapturedAt.Format(time.RFC3339Nano), string(rawBytes))
	if err != nil {
		return FrozenCore{}, err
	}
	// If another process won the race, return its persisted value rather than
	// exposing a non-authoritative local capture.
	if persisted, loadErr := s.Get(ctx, sessionID); loadErr == nil {
		return persisted, nil
	}
	return core, nil
}

func (s *Store) Get(ctx context.Context, sessionID string) (FrozenCore, error) {
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT core_json FROM frozen_core_sessions WHERE session_id=?`, sessionID).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FrozenCore{}, ErrSessionNotFound
		}
		return FrozenCore{}, err
	}
	var core FrozenCore
	if err := json.Unmarshal([]byte(raw), &core); err != nil {
		return FrozenCore{}, fmt.Errorf("personactx: corrupted frozen session: %w", err)
	}
	return core, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func truncateVisible(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	value = persona.NormalizeMarkdown(value)
	graphemes := uniseg.NewGraphemes(value)
	visible := 0
	var builder strings.Builder
	for graphemes.Next() {
		cluster := graphemes.Str()
		if !allWhitespace(cluster) {
			if visible >= limit {
				break
			}
			visible++
		}
		builder.WriteString(cluster)
	}
	return strings.TrimSpace(builder.String())
}

func allWhitespace(value string) bool { return strings.TrimSpace(value) == "" }
func runeLen(value string) int        { return len([]rune(value)) }
func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-1]) + "…"
}
