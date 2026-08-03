package report

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ModuleKindAmbition   = "ambition"
	ModuleKindSuggestion = "suggestion"

	ModuleStatusActive    = "active"
	ModuleStatusDismissed = "dismissed"

	MaxModuleContentRunes = 2000
)

var (
	ErrInvalidModuleKind    = errors.New("module kind must be ambition or suggestion")
	ErrInvalidModuleStatus  = errors.New("module status must be active or dismissed")
	ErrEmptyModuleContent   = errors.New("module content must not be empty")
	ErrModuleContentTooLong = errors.New("module content exceeds 2000 runes")
	ErrNoModuleChange       = errors.New("no module change provided")
)

type Module struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func validModuleKind(kind string) bool {
	return kind == ModuleKindAmbition || kind == ModuleKindSuggestion
}

func validModuleStatus(status string) bool {
	return status == ModuleStatusActive || status == ModuleStatusDismissed
}

func validateModuleContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return ErrEmptyModuleContent
	}
	if len([]rune(content)) > MaxModuleContentRunes {
		return ErrModuleContentTooLong
	}
	return nil
}

func (s *Service) CreateModule(ctx context.Context, kind, content string) (Module, error) {
	if !validModuleKind(kind) {
		return Module{}, ErrInvalidModuleKind
	}
	if err := validateModuleContent(content); err != nil {
		return Module{}, err
	}
	now := time.Now().UTC()
	m := Module{ID: "mod_" + uuid.NewString()[:8], Kind: kind, Content: content, Status: ModuleStatusActive, CreatedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `INSERT INTO human_modules(id,kind,content,status,created_at,updated_at) VALUES(?,?,?,?,?,?)`, m.ID, m.Kind, m.Content, m.Status, m.CreatedAt.Format(time.RFC3339Nano), m.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Module{}, err
	}
	return m, nil
}

func (s *Service) ListModules(ctx context.Context, kind, status string) ([]Module, error) {
	if !validModuleKind(kind) {
		return nil, ErrInvalidModuleKind
	}
	query := `SELECT id,kind,content,status,created_at,updated_at FROM human_modules WHERE kind=?`
	args := []any{kind}
	if status != "" && status != "all" {
		if !validModuleStatus(status) {
			return nil, ErrInvalidModuleStatus
		}
		query += ` AND status=?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Module{}
	for rows.Next() {
		var m Module
		var created, updated string
		if err := rows.Scan(&m.ID, &m.Kind, &m.Content, &m.Status, &created, &updated); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		m.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) UpdateModule(ctx context.Context, id, content, status string) (Module, error) {
	var m Module
	var created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id,kind,content,status,created_at,updated_at FROM human_modules WHERE id=?`, id).Scan(&m.ID, &m.Kind, &m.Content, &m.Status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Module{}, ErrNotFound
	}
	if err != nil {
		return Module{}, err
	}
	if content == "" && status == "" {
		return Module{}, ErrNoModuleChange
	}
	if content != "" {
		if err := validateModuleContent(content); err != nil {
			return Module{}, err
		}
		m.Content = content
	}
	if status != "" {
		if !validModuleStatus(status) {
			return Module{}, ErrInvalidModuleStatus
		}
		m.Status = status
	}
	m.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `UPDATE human_modules SET content=?,status=?,updated_at=? WHERE id=?`, m.Content, m.Status, m.UpdatedAt.Format(time.RFC3339Nano), m.ID)
	if err != nil {
		return Module{}, err
	}
	return m, nil
}

// moduleNamesInWindow returns the display names of modules with active
// entries created inside [start, end), for the monthly report attachment.
func (s *Service) moduleNamesInWindow(ctx context.Context, start, end time.Time) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT kind FROM human_modules WHERE status=? AND created_at>=? AND created_at<? ORDER BY kind`, ModuleStatusActive, start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); err != nil {
			return nil, err
		}
		switch kind {
		case ModuleKindAmbition:
			names = append(names, "AMBITION")
		case ModuleKindSuggestion:
			names = append(names, "USER SUGGESTIONS")
		}
	}
	return names, rows.Err()
}
