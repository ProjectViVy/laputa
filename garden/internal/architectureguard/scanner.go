package architectureguard

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Mode controls how a caller interprets scan results. Report mode is useful
// while the clean break is being staged; enforce mode is the blocking gate.
type Mode string

const (
	ModeReport  Mode = "report"
	ModeEnforce Mode = "enforce"
)

// Finding is one target-runtime architecture violation. Paths are relative to
// the scanned root so reports are stable across machines.
type Finding struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Rule  string `json:"rule"`
	Match string `json:"match"`
}

// Scan walks source/configuration files under root and returns deterministic
// clean-break findings. Historical documents, tests and build products are
// deliberately excluded: this guard protects the target runtime, while
// owning stories separately rewrite or retire obsolete tests.
func Scan(root string) ([]Finding, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDirectory(root, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isScannableFile(root, path) {
			return nil
		}
		fileFindings, scanErr := scanFile(root, path)
		if scanErr != nil {
			return scanErr
		}
		findings = append(findings, fileFindings...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		return findings[i].Match < findings[j].Match
	})
	return findings, nil
}

func shouldSkipDirectory(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return true
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return false
	}
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		switch strings.ToLower(part) {
		case ".git", ".omc", "node_modules", "dist", "vendor", "e2e-tmp":
			return true
		}
	}
	if strings.HasPrefix(rel, "docs/") || rel == "docs" {
		return true
	}
	// Support both repository-root scans and direct scans from the garden
	// module. The guard's own source contains the forbidden needles as data,
	// so it must never report itself.
	if rel == "internal/architectureguard" || rel == "cmd/architecture-guard" ||
		rel == "garden/internal/architectureguard" || rel == "garden/cmd/architecture-guard" {
		return true
	}
	return false
}

func isScannableFile(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	base := filepath.Base(path)
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") {
		return false
	}
	if strings.HasPrefix(base, "AGENTS.") || strings.HasPrefix(base, "README.") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

type forbiddenPattern struct {
	rule     string
	patterns []string
}

var runtimePatterns = []forbiddenPattern{
	{rule: "legacy-laputa-governance", patterns: []string{
		"github.com/ProjectViVy/laputa/laputa/governance",
		"governance.NewFileStore",
		"governance.NewEngine",
		"governance.NewGovernedService",
		"NewGovernedService(",
	}},
	{rule: "legacy-json-sections", patterns: []string{
		".laputa/sections",
		"01-identity",
		"02-relationship",
		"03-commitment",
		"04-preferences",
		"05-memory_md",
		"memory_md",
		"MEMORY.MD",
		"LONGMEM.MD",
		"history_md",
	}},
	{rule: "legacy-json-patch", patterns: []string{
		"evanphx/json-patch",
		"json-patch/jsonpatch",
		"JSON Patch",
		"json_patch",
	}},
	{rule: "legacy-governance-types", patterns: []string{
		"SectionMemoryMD",
		"GovernanceProjection",
		"WorldProjector",
		"WorldClaim",
		"WorldResponse",
		"WorldStore",
	}},
	{rule: "legacy-governance-routes", patterns: []string{
		"/v2/governance",
		"/v2/cognitive/world",
	}},
}

var mentleBoundaryPatterns = []string{
	"github.com/ProjectViVy/laputa/laputa",
	"laputa/persona",
	"PERSONA.MD",
	"WORLD.MD",
	"ACTMEM.MD",
}

func scanFile(root, path string) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	rel = filepath.ToSlash(rel)
	isMentle := strings.HasPrefix(rel, "mentle/")
	scanner := bufio.NewScanner(f)
	// Source/configuration lines are intentionally bounded; a very long line
	// is still scanned rather than silently skipped.
	scanner.Buffer(make([]byte, 4096), 2<<20)
	var findings []Finding
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		for _, pattern := range runtimePatterns {
			for _, needle := range pattern.patterns {
				if strings.Contains(line, needle) {
					findings = append(findings, Finding{Path: rel, Line: lineNumber, Rule: pattern.rule, Match: needle})
				}
			}
		}
		if isMentle {
			for _, needle := range mentleBoundaryPatterns {
				if strings.Contains(line, needle) {
					findings = append(findings, Finding{Path: rel, Line: lineNumber, Rule: "mentle-authority-boundary", Match: needle})
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", rel, err)
	}
	return findings, nil
}

// ExitCode returns the process code for a scan under mode.
func ExitCode(mode Mode, findingCount int) int {
	if mode == ModeEnforce && findingCount > 0 {
		return 1
	}
	return 0
}
