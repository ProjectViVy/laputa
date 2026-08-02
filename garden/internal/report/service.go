package report

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dashimaki/mentle/facade"
	_ "github.com/mattn/go-sqlite3"
)

var ErrNotFound = errors.New("report not found")

type Report struct {
	Cadence       string    `json:"cadence"`
	WindowStart   time.Time `json:"window_start"`
	WindowEnd     time.Time `json:"window_end"`
	SourceIDs     []string  `json:"source_ids"`
	SourceHash    string    `json:"source_hash"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	Highlights    []string  `json:"highlights"`
	OpenQuestions []string  `json:"open_questions"`
	GeneratedAt   time.Time `json:"generated_at"`
}

type Service struct {
	db     *sql.DB
	memory MemoryLister
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type MemoryLister interface {
	ListMemories(context.Context, facade.ListMemoryOptions) (facade.MemoryPage, error)
}

func Open(path string, memory MemoryLister) (*Service, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS reports(cadence TEXT NOT NULL,window_start TEXT NOT NULL,window_end TEXT NOT NULL,source_ids TEXT NOT NULL,source_hash TEXT NOT NULL,title TEXT NOT NULL,summary TEXT NOT NULL,highlights TEXT NOT NULL,open_questions TEXT NOT NULL,generated_at TEXT NOT NULL,PRIMARY KEY(cadence,window_start,source_hash));CREATE INDEX IF NOT EXISTS reports_latest ON reports(cadence,generated_at DESC);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{db: db, memory: memory, cancel: cancel}
	s.wg.Add(1)
	go s.loop(ctx)
	return s, nil
}
func (s *Service) Close() error { s.cancel(); s.wg.Wait(); return s.db.Close() }
func (s *Service) loop(ctx context.Context) {
	defer s.wg.Done()
	s.GenerateAll(ctx, time.Now().UTC())
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.GenerateAll(ctx, now.UTC())
		}
	}
}
func (s *Service) GenerateAll(ctx context.Context, now time.Time) {
	for _, c := range []string{"daily", "weekly", "monthly"} {
		_, _ = s.Generate(ctx, c, now)
	}
}
func (s *Service) Generate(ctx context.Context, cadence string, now time.Time) (Report, error) {
	if s.memory == nil {
		return Report{}, facade.ErrUnavailable
	}
	start, end, err := window(cadence, now)
	if err != nil {
		return Report{}, err
	}
	memories := []facade.Memory{}
	cursor := ""
	for {
		page, e := s.memory.ListMemories(ctx, facade.ListMemoryOptions{Limit: 200, Cursor: cursor, Status: "active"})
		if e != nil {
			return Report{}, e
		}
		for _, m := range page.Items {
			if !m.UpdatedAt.Before(start) && m.UpdatedAt.Before(end) {
				memories = append(memories, m)
			}
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if len(memories) == 0 {
		return Report{}, ErrNotFound
	}
	sort.Slice(memories, func(i, j int) bool { return memories[i].ID < memories[j].ID })
	ids := make([]string, len(memories))
	highlights := []string{}
	var summary strings.Builder
	for i, m := range memories {
		ids[i] = m.ID
		if len(highlights) < 10 {
			highlights = append(highlights, truncate(m.Content, 240))
		}
		if summary.Len() < 4000 {
			summary.WriteString("- " + truncate(m.Content, 500) + "\n")
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	r := Report{Cadence: cadence, WindowStart: start, WindowEnd: end, SourceIDs: ids, SourceHash: "sha256:" + hex.EncodeToString(sum[:]), Title: strings.Title(cadence) + " Garden Memory Report", Summary: summary.String(), Highlights: highlights, OpenQuestions: []string{}, GeneratedAt: time.Now().UTC()}
	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO reports(cadence,window_start,window_end,source_ids,source_hash,title,summary,highlights,open_questions,generated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, r.Cadence, r.WindowStart.Format(time.RFC3339Nano), r.WindowEnd.Format(time.RFC3339Nano), encode(r.SourceIDs), r.SourceHash, r.Title, r.Summary, encode(r.Highlights), encode(r.OpenQuestions), r.GeneratedAt.Format(time.RFC3339Nano))
	return r, err
}
func (s *Service) Latest(ctx context.Context, cadence string) (Report, error) {
	if cadence != "daily" && cadence != "weekly" && cadence != "monthly" {
		return Report{}, errors.New("cadence must be daily, weekly, or monthly")
	}
	var r Report
	var start, end, ids, highlights, questions, generated string
	err := s.db.QueryRowContext(ctx, `SELECT cadence,window_start,window_end,source_ids,source_hash,title,summary,highlights,open_questions,generated_at FROM reports WHERE cadence=? ORDER BY generated_at DESC LIMIT 1`, cadence).Scan(&r.Cadence, &start, &end, &ids, &r.SourceHash, &r.Title, &r.Summary, &highlights, &questions, &generated)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.WindowStart, _ = time.Parse(time.RFC3339Nano, start)
	r.WindowEnd, _ = time.Parse(time.RFC3339Nano, end)
	r.GeneratedAt, _ = time.Parse(time.RFC3339Nano, generated)
	_ = json.Unmarshal([]byte(ids), &r.SourceIDs)
	_ = json.Unmarshal([]byte(highlights), &r.Highlights)
	_ = json.Unmarshal([]byte(questions), &r.OpenQuestions)
	return r, nil
}
func window(c string, now time.Time) (time.Time, time.Time, error) {
	now = now.UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	switch c {
	case "daily":
		return day, day.AddDate(0, 0, 1), nil
	case "weekly":
		start := day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
		return start, start.AddDate(0, 0, 7), nil
	case "monthly":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0), nil
	default:
		return time.Time{}, time.Time{}, errors.New("invalid cadence")
	}
}
func encode(v any) string { b, _ := json.Marshal(v); return string(b) }
func truncate(v string, n int) string {
	r := []rune(v)
	if len(r) <= n {
		return v
	}
	return string(r[:n]) + "…"
}
