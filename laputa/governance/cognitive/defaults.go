package cognitive

import (
	"os"
	"path/filepath"
)

const MemRulesFileName = "MEMRULES.MD"
const WorldFileName = "WORLD.MD"

// DefaultMemRulesText is the built-in fallback rulebook (ADR-0004 §2.2/§2.4).
const DefaultMemRulesText = `---
version: 1
updated: 2026-08-03T00:00:00Z
---

# Memory Rules

## R1 — Evidence primacy
Mentle raw material and evidence remain the primary source of truth.

## R2 — Claim distinction
Confirmed fact, observation, inference, and hypothesis are distinct categories.

## R3 — Contradiction handling
New contradictory evidence does not silently overwrite prior understanding.

## R4 — User authority
User-confirmed information outranks agent inference.

## R5 — Scope constraint
Scope, time, confidence, provenance, and visibility constrain use.

## R6 — WORLD entry gate
Entry into WORLD requires action relevance and a bounded, reviewable claim.

## R7 — No wholesale injection
WORLD is not copied wholesale into an Agent context.
`

// DefaultWorldText is the empty world file created on first boot.
const DefaultWorldText = "# WORLD\n"

// DefaultMemRules parses the built-in rulebook.
func DefaultMemRules() *MemRules {
	m := &MemRules{Raw: DefaultMemRulesText}
	m.parse()
	return m
}

// InitializeDir creates the cognitive directory and seeds MEMRULES.MD and
// WORLD.MD when absent. Existing files are never overwritten.
func InitializeDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for name, content := range map[string]string{
		MemRulesFileName: DefaultMemRulesText,
		WorldFileName:    DefaultWorldText,
	} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}
