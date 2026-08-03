package cognitive

import (
	"crypto/sha256"
	"errors"
	"os"
	"sync"

	lpcog "github.com/dashimaki/laputa/governance/cognitive"
)

// RulesProvider exposes the loaded MEMRULES rulebook. The interface lives in
// Garden; the schema and authority stay in Laputa (ADR-0004 §2.4).
type RulesProvider interface {
	Rules() []lpcog.Rule
	Version() string
	Origin() string // "file" or "builtin"
}

var defaultHash = sha256.Sum256([]byte(lpcog.DefaultMemRulesText))

// Service loads MEMRULES.MD once at boot. A missing file falls back to the
// built-in default rulebook without error.
type Service struct {
	mu      sync.RWMutex
	path    string
	current *lpcog.MemRules
	origin  string
}

func Load(path string) (*Service, error) {
	s := &Service{path: path}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Reload() error {
	m, err := lpcog.LoadMemRules(s.path)
	origin := "file"
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		m = lpcog.DefaultMemRules()
		m.Path = s.path
		origin = "builtin"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = m
	s.origin = origin
	return nil
}

func (s *Service) Rules() []lpcog.Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.Rules
}

func (s *Service) Version() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.Version
}

func (s *Service) Origin() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.origin
}

// MatchesDefault reports whether the loaded rulebook is byte-identical to the
// built-in default. A false result means a human edited MEMRULES.MD, which
// ADR-0004 §5.3 requires to be audited at reload.
func (s *Service) MatchesDefault() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.origin == "builtin" {
		return true
	}
	return sha256.Sum256([]byte(s.current.Raw)) == defaultHash
}

// HashPrefix returns a short content hash for audit rollback references.
func (s *Service) HashPrefix() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := sha256.Sum256([]byte(s.current.Raw))
	const hex = "0123456789abcdef"
	out := make([]byte, 16)
	for i := 0; i < 8; i++ {
		out[i*2] = hex[h[i]>>4]
		out[i*2+1] = hex[h[i]&0x0f]
	}
	return string(out)
}
