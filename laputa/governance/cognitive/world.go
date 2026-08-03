package cognitive

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type ClaimStatus string

const (
	ClaimConfirmed  ClaimStatus = "confirmed"
	ClaimObserved   ClaimStatus = "observed"
	ClaimInferred   ClaimStatus = "inferred"
	ClaimHypothesis ClaimStatus = "hypothesis"
	ClaimStale      ClaimStatus = "stale"
)

var ErrConfirmedProtected = errors.New("cannot overwrite user-confirmed claim")

type WorldClaim struct {
	Domain     string      `json:"domain"`
	Title      string      `json:"title"`
	Status     ClaimStatus `json:"status"`
	Confidence string      `json:"confidence"`
	Scopes     []string    `json:"scopes"`
	Source     string      `json:"source"`
	Updated    time.Time   `json:"updated"`
	Text       string      `json:"text"`
}

type WorldStore struct {
	mu     sync.RWMutex
	Path   string
	Claims []WorldClaim
}

func LoadWorld(path string) (*WorldStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	w := &WorldStore{Path: path}
	w.Claims = parseWorld(string(data))
	return w, nil
}

func (w *WorldStore) Total() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.Claims)
}

var confidenceRank = map[string]int{"high": 3, "medium": 2, "low": 1}

func (w *WorldStore) Project(scopes []string, budgetChars int) []WorldClaim {
	if budgetChars <= 0 {
		budgetChars = 4000
	}
	w.mu.RLock()
	snapshot := make([]WorldClaim, len(w.Claims))
	copy(snapshot, w.Claims)
	w.mu.RUnlock()

	if len(scopes) == 0 {
		sort.SliceStable(snapshot, func(i, j int) bool {
			return confidenceRank[snapshot[i].Confidence] > confidenceRank[snapshot[j].Confidence]
		})
	}

	result := []WorldClaim{}
	used := 0
	for _, claim := range snapshot {
		if len(scopes) > 0 && !claim.matchesScope(scopes) {
			continue
		}
		cost := len([]rune(claim.Text))
		if used+cost > budgetChars {
			break
		}
		result = append(result, claim)
		used += cost
	}
	return result
}

func (w *WorldStore) Save(actor string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if actor != "user" {
		for _, claim := range w.Claims {
			if claim.Status == ClaimConfirmed && claim.Source == "user" {
				return ErrConfirmedProtected
			}
		}
	}
	content := w.serialize()
	return os.WriteFile(w.Path, []byte(content), 0644)
}

func (claim *WorldClaim) matchesScope(scopes []string) bool {
	for _, s := range scopes {
		for _, cs := range claim.Scopes {
			if cs == s {
				return true
			}
		}
	}
	return false
}

func (w *WorldStore) serialize() string {
	var sb strings.Builder
	sb.WriteString("# WORLD\n\n")
	for _, claim := range w.Claims {
		sb.WriteString(fmt.Sprintf("## [%s] %s\n", claim.Domain, claim.Title))
		sb.WriteString(fmt.Sprintf("- status: %s\n", claim.Status))
		sb.WriteString(fmt.Sprintf("- confidence: %s\n", claim.Confidence))
		sb.WriteString(fmt.Sprintf("- scope: %s\n", strings.Join(claim.Scopes, ", ")))
		sb.WriteString(fmt.Sprintf("- source: %s\n", claim.Source))
		sb.WriteString(fmt.Sprintf("- updated: %s\n", claim.Updated.UTC().Format(time.RFC3339)))
		sb.WriteString("\n")
		sb.WriteString(claim.Text)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func parseWorld(raw string) []WorldClaim {
	var claims []WorldClaim
	scanner := bufio.NewScanner(strings.NewReader(raw))
	var current *WorldClaim

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "## [") {
			if current != nil {
				claims = append(claims, *current)
			}
			domain, title := parseClaimHeading(line)
			current = &WorldClaim{Domain: domain, Title: title}
			continue
		}

		if current == nil {
			continue
		}

		if strings.HasPrefix(line, "- ") {
			parseClaimMeta(current, line[2:])
			continue
		}

		if strings.TrimSpace(line) != "" {
			if current.Text != "" {
				current.Text += "\n"
			}
			current.Text += line
		}
	}
	if current != nil {
		claims = append(claims, *current)
	}
	return claims
}

func parseClaimHeading(line string) (domain, title string) {
	rest := strings.TrimPrefix(line, "## [")
	idx := strings.Index(rest, "]")
	if idx < 0 {
		return "", strings.TrimSpace(rest)
	}
	domain = rest[:idx]
	title = strings.TrimSpace(rest[idx+1:])
	return domain, title
}

func parseClaimMeta(claim *WorldClaim, field string) {
	key, value, found := strings.Cut(field, ":")
	if !found {
		return
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	switch key {
	case "status":
		claim.Status = ClaimStatus(value)
	case "confidence":
		claim.Confidence = value
	case "scope":
		parts := strings.Split(value, ",")
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				claim.Scopes = append(claim.Scopes, s)
			}
		}
	case "source":
		claim.Source = value
	case "updated":
		if t, err := time.Parse(time.RFC3339, value); err == nil {
			claim.Updated = t
		}
	}
}
