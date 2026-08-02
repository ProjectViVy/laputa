// Package layers implements the L0-L3 memory stack model.
// L0 is identity, L1 is essential story, L2 is on-demand recall.
package layers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/search"
)

type Layer0 struct {
	path string
}

func NewLayer0(identityPath string) *Layer0 {
	return &Layer0{path: identityPath}
}

func (l *Layer0) Render() (string, error) {
	data, err := os.ReadFile(l.path)
	if os.IsNotExist(err) {
		return "## L0 — IDENTITY\nNo identity configured.", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

type Layer1 struct {
	searcher *search.Searcher
	wing     string
}

func NewLayer1(searcher *search.Searcher) *Layer1 {
	return &Layer1{searcher: searcher}
}

func (l *Layer1) Generate(ctx context.Context) (string, error) {
	results, err := l.searcher.Search(ctx, "", l.wing, "", 15)
	if err != nil || len(results) == 0 {
		return "## L1 — No memories yet.", nil
	}
	var lines []string
	lines = append(lines, "## L1 — ESSENTIAL STORY")
	for _, r := range results {
		lines = append(lines, "- ["+r.Wing+"/"+r.Room+"] "+r.Content)
	}
	return strings.Join(lines, "\n"), nil
}

type MemoryStack struct {
	cfg      *config.Config
	l0       *Layer0
	l1       *Layer1
	searcher *search.Searcher
}

func NewMemoryStack(cfg *config.Config, searcher *search.Searcher) *MemoryStack {
	identityPath, _ := cfg.GetIdentityPath()
	return &MemoryStack{
		cfg:      cfg,
		l0:       NewLayer0(identityPath),
		l1:       NewLayer1(searcher),
		searcher: searcher,
	}
}

func (s *MemoryStack) WakeUp(ctx context.Context, wing string) (string, error) {
	l0Text, err := s.l0.Render()
	if err != nil {
		return "", err
	}
	s.l1.wing = wing
	l1Text, err := s.l1.Generate(ctx)
	if err != nil {
		return "", err
	}
	return l0Text + "\n\n" + l1Text, nil
}

func (s *MemoryStack) Search(ctx context.Context, query string, wing, room string, nResults int) ([]search.Drawer, error) {
	return s.searcher.Search(ctx, query, wing, room, nResults)
}

func (s *MemoryStack) Recall(ctx context.Context, wing, room string, nResults int) (string, error) {
	results, err := s.searcher.Search(ctx, "", wing, room, nResults)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No drawers found.", nil
	}
	var lines []string
	lines = append(lines, "## L2 — ON-DEMAND")
	for _, r := range results {
		lines = append(lines, "- ["+r.Wing+"/"+r.Room+"] "+r.Content)
	}
	return strings.Join(lines, "\n"), nil
}

func (s *MemoryStack) DeepSearch(ctx context.Context, query string, wing, room string, nResults int) ([]search.Drawer, error) {
	return s.searcher.Search(ctx, query, wing, room, nResults)
}

func (s *MemoryStack) DeepSearchFormatted(ctx context.Context, query string, wing, room string, nResults int) (string, error) {
	results, err := s.searcher.Search(ctx, query, wing, room, nResults)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No drawers found.", nil
	}
	var lines []string
	lines = append(lines, "## L3 — DEEP SEARCH")
	lines = append(lines, fmt.Sprintf("Query: %s", query))
	lines = append(lines, "")
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("### [%s/%s] %s", r.Wing, r.Room, r.ID))
		lines = append(lines, r.Content)
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n"), nil
}
