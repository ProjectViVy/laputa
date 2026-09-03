package report

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dashimaki/mentle/facade"
	_ "github.com/mattn/go-sqlite3"
)

var ErrNotFound = errors.New("report not found")

const (
	GeneratorDeterministic = "deterministic"
	GeneratorLLM           = "llm"
)

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
	Scope         string    `json:"scope"`
	Goals         []string  `json:"goals"`
	Completed     []string  `json:"completed"`
	Decisions     []string  `json:"decisions"`
	OpenLoops     []string  `json:"open_loops"`
	SourceRefs    []string  `json:"source_refs"`
	Revision      int       `json:"revision"`
	Generator     string    `json:"generator"`
	Modules       []string  `json:"modules"`
}

// artifact carries the ADR-0005 artifact fields persisted in one column.
type artifact struct {
	Scope      string   `json:"scope"`
	Goals      []string `json:"goals"`
	Completed  []string `json:"completed"`
	Decisions  []string `json:"decisions"`
	OpenLoops  []string `json:"open_loops"`
	SourceRefs []string `json:"source_refs"`
	Revision   int      `json:"revision"`
	Generator  string   `json:"generator"`
	Modules    []string `json:"modules"`
}

func (a artifact) apply(r *Report) {
	r.Scope = a.Scope
	r.Goals = a.Goals
	r.Completed = a.Completed
	r.Decisions = a.Decisions
	r.OpenLoops = a.OpenLoops
	r.SourceRefs = a.SourceRefs
	r.Revision = a.Revision
	r.Generator = a.Generator
	r.Modules = a.Modules
	if r.Generator == "" {
		r.Generator = GeneratorDeterministic
	}
	if r.Goals == nil {
		r.Goals = []string{}
	}
	if r.Completed == nil {
		r.Completed = []string{}
	}
	if r.Decisions == nil {
		r.Decisions = []string{}
	}
	if r.OpenLoops == nil {
		r.OpenLoops = []string{}
	}
	if r.SourceRefs == nil {
		r.SourceRefs = []string{}
	}
	if r.Modules == nil {
		r.Modules = []string{}
	}
}

type Service struct {
	db        *sql.DB
	memory    MemoryLister
	Publisher Publisher
	Enricher  Enricher
	clock     func() time.Time
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

type MemoryLister interface {
	ListMemories(context.Context, facade.ListMemoryOptions) (facade.MemoryPage, error)
}

func Open(path string, memory MemoryLister, publisher Publisher, enricher Enricher) (*Service, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS reports(cadence TEXT NOT NULL,window_start TEXT NOT NULL,window_end TEXT NOT NULL,source_ids TEXT NOT NULL,source_hash TEXT NOT NULL,title TEXT NOT NULL,summary TEXT NOT NULL,highlights TEXT NOT NULL,open_questions TEXT NOT NULL,generated_at TEXT NOT NULL,PRIMARY KEY(cadence,window_start,source_hash));CREATE INDEX IF NOT EXISTS reports_latest ON reports(cadence,generated_at DESC);CREATE TABLE IF NOT EXISTS human_modules(id TEXT PRIMARY KEY,kind TEXT NOT NULL CHECK(kind IN ('ambition','suggestion')),content TEXT NOT NULL,status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','dismissed')),created_at TEXT NOT NULL,updated_at TEXT NOT NULL);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err := migrateArtifactColumn(db); err != nil {
		db.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{db: db, memory: memory, Publisher: publisher, Enricher: enricher, cancel: cancel}
	s.wg.Add(1)
	go s.loop(ctx)
	return s, nil
}

// SetClock installs a deterministic clock for callers that need to exercise
// time-window behavior. Production callers leave the clock unset.
func (s *Service) SetClock(clock func() time.Time) {
	s.clock = clock
}

func (s *Service) now() time.Time {
	if s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
}

func migrateArtifactColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(reports)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "artifact" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(`ALTER TABLE reports ADD COLUMN artifact TEXT NOT NULL DEFAULT '{}'`)
	return err
}

func (s *Service) Close() error { s.cancel(); s.wg.Wait(); return s.db.Close() }
func (s *Service) loop(ctx context.Context) {
	defer s.wg.Done()
	s.GenerateAll(ctx, s.now())
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
	decisions := []string{}
	var summary strings.Builder
	for i, m := range memories {
		ids[i] = m.ID
		if len(highlights) < 10 {
			highlights = append(highlights, truncate(m.Content, 240))
		}
		if m.Kind == "decision" && len(decisions) < 10 {
			decisions = append(decisions, truncate(m.Content, 240))
		}
		if summary.Len() < 4000 {
			summary.WriteString("- " + truncate(m.Content, 500) + "\n")
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	r := Report{Cadence: cadence, WindowStart: start, WindowEnd: end, SourceIDs: ids, SourceHash: "sha256:" + hex.EncodeToString(sum[:]), Title: strings.Title(cadence) + " Garden Memory Report", Summary: summary.String(), Highlights: highlights, OpenQuestions: []string{}, GeneratedAt: s.now(), Scope: "mentle_active", Completed: highlights, Decisions: decisions, Goals: []string{}, OpenLoops: []string{}, SourceRefs: ids, Generator: GeneratorDeterministic, Modules: []string{}}
	if s.Enricher != nil {
		if enriched, eerr := s.Enricher.Enrich(ctx, r); eerr != nil {
			log.Printf("report enrich %s: %v", cadence, eerr)
		} else {
			r = enriched
		}
	}
	if cadence == "monthly" {
		names, merr := s.moduleNamesInWindow(ctx, start, end)
		if merr != nil {
			return Report{}, merr
		}
		r.Modules = names
	}
	r.Revision, err = s.nextRevision(ctx, cadence, start)
	if err != nil {
		return Report{}, err
	}
	res, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO reports(cadence,window_start,window_end,source_ids,source_hash,title,summary,highlights,open_questions,generated_at,artifact) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, r.Cadence, r.WindowStart.Format(time.RFC3339Nano), r.WindowEnd.Format(time.RFC3339Nano), encode(r.SourceIDs), r.SourceHash, r.Title, r.Summary, encode(r.Highlights), encode(r.OpenQuestions), r.GeneratedAt.Format(time.RFC3339Nano), encode(artifact{Scope: r.Scope, Goals: r.Goals, Completed: r.Completed, Decisions: r.Decisions, OpenLoops: r.OpenLoops, SourceRefs: r.SourceRefs, Revision: r.Revision, Generator: r.Generator, Modules: r.Modules}))
	if err != nil {
		return Report{}, err
	}
	if affected, aerr := res.RowsAffected(); aerr == nil && affected == 0 {
		return s.saved(ctx, cadence, r.SourceHash)
	}
	return r, nil
}

func (s *Service) saved(ctx context.Context, cadence, sourceHash string) (Report, error) {
	var r Report
	var start, end, ids, highlights, questions, generated, art string
	err := s.db.QueryRowContext(ctx, `SELECT cadence,window_start,window_end,source_ids,source_hash,title,summary,highlights,open_questions,generated_at,artifact FROM reports WHERE cadence=? AND source_hash=? LIMIT 1`, cadence, sourceHash).Scan(&r.Cadence, &start, &end, &ids, &r.SourceHash, &r.Title, &r.Summary, &highlights, &questions, &generated, &art)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	return scanReport(&r, start, end, ids, highlights, questions, generated, art), nil
}

func scanReport(r *Report, start, end, ids, highlights, questions, generated, art string) Report {
	r.WindowStart, _ = time.Parse(time.RFC3339Nano, start)
	r.WindowEnd, _ = time.Parse(time.RFC3339Nano, end)
	r.GeneratedAt, _ = time.Parse(time.RFC3339Nano, generated)
	_ = json.Unmarshal([]byte(ids), &r.SourceIDs)
	_ = json.Unmarshal([]byte(highlights), &r.Highlights)
	_ = json.Unmarshal([]byte(questions), &r.OpenQuestions)
	var a artifact
	_ = json.Unmarshal([]byte(art), &a)
	a.apply(r)
	return *r
}

func (s *Service) nextRevision(ctx context.Context, cadence string, start time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reports WHERE cadence=? AND window_start=?`, cadence, start.Format(time.RFC3339Nano)).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count + 1, nil
}

func (s *Service) Latest(ctx context.Context, cadence string) (Report, error) {
	if cadence != "daily" && cadence != "weekly" && cadence != "monthly" {
		return Report{}, errors.New("cadence must be daily, weekly, or monthly")
	}
	var r Report
	var start, end, ids, highlights, questions, generated, art string
	err := s.db.QueryRowContext(ctx, `SELECT cadence,window_start,window_end,source_ids,source_hash,title,summary,highlights,open_questions,generated_at,artifact FROM reports WHERE cadence=? ORDER BY window_start DESC, generated_at DESC, rowid DESC LIMIT 1`, cadence).Scan(&r.Cadence, &start, &end, &ids, &r.SourceHash, &r.Title, &r.Summary, &highlights, &questions, &generated, &art)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	return scanReport(&r, start, end, ids, highlights, questions, generated, art), nil
}

func (s *Service) List(ctx context.Context, cadence string, limit int) ([]Report, error) {
	if cadence != "daily" && cadence != "weekly" && cadence != "monthly" {
		return nil, errors.New("invalid cadence: must be daily, weekly, or monthly")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT cadence,window_start,window_end,source_ids,source_hash,title,summary,highlights,open_questions,generated_at,artifact FROM reports WHERE cadence=? ORDER BY window_start DESC, generated_at DESC, rowid DESC LIMIT ?`, cadence, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Report{}
	for rows.Next() {
		var r Report
		var start, end, ids, highlights, questions, generated, art string
		if err := rows.Scan(&r.Cadence, &start, &end, &ids, &r.SourceHash, &r.Title, &r.Summary, &highlights, &questions, &generated, &art); err != nil {
			return nil, err
		}
		out = append(out, scanReport(&r, start, end, ids, highlights, questions, generated, art))
	}
	return out, rows.Err()
}

// OrientationNote is the contractual disclaimer carried by every orientation
// response (ADR-0005 §7): orientation never replaces transient spool recovery.
const OrientationNote = "orientation only; does not replace transient spool recovery"

type OrientationView struct {
	Cadence     string   `json:"cadence"`
	Orientation string   `json:"orientation"`
	BudgetChars int      `json:"budget_chars"`
	Note        string   `json:"note"`
	Warnings    []string `json:"warnings"`
}

func (s *Service) Orientation(ctx context.Context, budgetChars int) (OrientationView, error) {
	if budgetChars <= 0 {
		budgetChars = 2000
	}
	if budgetChars > 8000 {
		budgetChars = 8000
	}
	if budgetChars < 100 {
		budgetChars = 100
	}
	view := OrientationView{Cadence: "daily", BudgetChars: budgetChars, Note: OrientationNote, Warnings: []string{}}
	r, err := s.Latest(ctx, "daily")
	if errors.Is(err, ErrNotFound) {
		view.Warnings = append(view.Warnings, "no daily report available")
		return view, nil
	}
	if err != nil {
		return view, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s to %s), revision %d, generator %s\n", r.Title, r.WindowStart.Format("2006-01-02"), r.WindowEnd.Format("2006-01-02"), r.Revision, r.Generator)
	b.WriteString(r.Summary)
	appendLabeledLines(&b, "completed", r.Completed)
	appendLabeledLines(&b, "decisions", r.Decisions)
	appendLabeledLines(&b, "goals", r.Goals)
	appendLabeledLines(&b, "open loops", r.OpenLoops)
	text := b.String()
	if runes := []rune(text); len(runes) > budgetChars {
		text = string(runes[:budgetChars-1]) + "…"
	}
	view.Orientation = text
	return view, nil
}

func appendLabeledLines(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString(label + ":\n")
	for _, item := range items {
		b.WriteString("- " + item + "\n")
	}
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
