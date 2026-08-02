// Package miner extracts memories from project files and conversations.
// It walks directories and stores content as drawers in the palace.
package miner

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/internal/search"
)

// SupportedFileExtensions is the set of file extensions the miner will process.
var SupportedFileExtensions = map[string]bool{
	// Programming languages
	".go": true, ".py": true, ".rs": true, ".js": true, ".ts": true,
	".jsx": true, ".tsx": true, ".java": true, ".rb": true, ".sh": true,
	".c": true, ".cpp": true, ".h": true, ".hpp": true,
	".swift": true, ".kt": true, ".scala": true, ".php": true,
	// Web
	".html": true, ".css": true, ".json": true,
	// Config
	".yaml": true, ".yml": true, ".toml": true, ".ini": true, ".env": true,
	// Data
	".csv": true, ".sql": true, ".xml": true,
	// Documentation
	".md": true, ".txt": true, ".rst": true, ".adoc": true,
}

// SkipDirs is the set of directory names to skip during mining.
var SkipDirs = map[string]bool{
	".git": true, ".svn": true, ".hg": true,
	"node_modules": true, "__pycache__": true, ".venv": true, "venv": true,
	".next": true, ".nuxt": true, "dist": true, "build": true, "out": true,
	"coverage": true, ".coverage": true, ".cache": true,
	".idea": true, ".vscode": true,
}

// RoomDetector determines which room a file should belong to.
type RoomDetector interface {
	DetectRoom(filePath string, content string, projectPath string) string
}

// Miner extracts project files into the memory palace.
type Miner struct {
	searcher     *search.Searcher
	ignoreFns    []func(path string) bool
	roomDetector RoomDetector
	projectDir   string
}

// NewMiner creates a new Miner.
func NewMiner(searcher *search.Searcher) *Miner {
	return &Miner{
		searcher:  searcher,
		ignoreFns: []func(path string) bool{},
	}
}

// AddIgnoreFn adds a custom ignore function.
func (m *Miner) AddIgnoreFn(fn func(path string) bool) {
	m.ignoreFns = append(m.ignoreFns, fn)
}

// SetRoomDetector sets a custom room detector.
func (m *Miner) SetRoomDetector(detector RoomDetector) {
	m.roomDetector = detector
}

// LoadGitignore loads .gitignore patterns from the given directory and adds an ignore function.
func (m *Miner) LoadGitignore(dir string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")
	patterns, err := loadGitignorePatterns(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No gitignore is fine
		}
		return err
	}
	if len(patterns) > 0 {
		m.AddIgnoreFn(func(path string) bool {
			return matchesGitignore(path, dir, patterns)
		})
	}
	return nil
}

// MineProject walks a directory and mines supported files into the palace.
func (m *Miner) MineProject(ctx context.Context, dir, wingOverride string) error {
	if wingOverride == "" {
		wingOverride = filepath.Base(dir)
	}
	m.projectDir = dir

	var mined, skipped, upToDate int

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue
		}

		// Skip directories
		if info.IsDir() {
			if SkipDirs[info.Name()] {
				return filepath.SkipDir
			}
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			skipped++
			return nil
		}

		// Skip files over 10MB
		if info.Size() > 10*1024*1024 {
			skipped++
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(path))
		if !SupportedFileExtensions[ext] {
			skipped++
			return nil
		}

		// Check ignore functions
		for _, fn := range m.ignoreFns {
			if fn(path) {
				skipped++
				return nil
			}
		}

		absPath, _ := filepath.Abs(path)

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Content-hash + mtime dedup
		contentHash := GenerateContentHash(string(content))
		if m.isUpToDate(absPath, info.ModTime()) && IsContentAlreadyMined(absPath, contentHash) {
			upToDate++
			return nil
		}

		// Determine room from path
		var room string
		if m.roomDetector != nil {
			room = m.roomDetector.DetectRoom(absPath, string(content), m.projectDir)
		} else {
			room = DetectRoomCombined(absPath, string(content))
		}

		// Chunk large files (800 chars, 100 overlap)
		chunks := chunkContent(string(content), 800, 100)
		fileMined := false
		for i, chunk := range chunks {
			drawer := palace.Drawer{
				ID:         fmt.Sprintf("d_%d_%d", time.Now().UnixNano(), i),
				Content:    chunk,
				Wing:       wingOverride,
				Room:       room,
				SourceFile: absPath,
				ChunkIndex: i,
				AddedBy:    "mempalace-go",
				FiledAt:    time.Now(),
				Metadata: map[string]string{
					"mtime": info.ModTime().Format(time.RFC3339),
				},
			}

			if err := m.searcher.Store(ctx, drawer); err != nil {
				fmt.Printf("Warning: failed to store %s chunk %d: %v\n", path, i, err)
			} else {
				mined++
				fileMined = true
			}
		}
		if fileMined {
			updateMtime(absPath, info.ModTime())
			RegisterContentHash(absPath, contentHash)
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Mined %d chunks, %d up-to-date, %d skipped into wing '%s'\n",
		mined, upToDate, skipped, wingOverride)
	return nil
}

// MtimeIndex tracks the last modification time of mined files.
var mtimeIndex = make(map[string]string)

// LoadMtimeIndex loads the mtime index from the palace path.
func LoadMtimeIndex(palacePath string) error {
	path := filepath.Join(palacePath, "mtime_index.txt")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 2)
		if len(parts) == 2 {
			mtimeIndex[parts[0]] = parts[1]
		}
	}
	return scanner.Err()
}

// SaveMtimeIndex saves the mtime index to the palace path.
func SaveMtimeIndex(palacePath string) error {
	path := filepath.Join(palacePath, "mtime_index.txt")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for file, mtime := range mtimeIndex {
		fmt.Fprintf(f, "%s\t%s\n", file, mtime)
	}
	return nil
}

func LoadContentHashIndex(palacePath string) error {
	path := filepath.Join(palacePath, "content_hash_index.txt")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 2)
		if len(parts) == 2 {
			contentHashIndex[parts[0]] = parts[1]
		}
	}
	return scanner.Err()
}

func SaveContentHashIndex(palacePath string) error {
	path := filepath.Join(palacePath, "content_hash_index.txt")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for file, hash := range contentHashIndex {
		fmt.Fprintf(f, "%s\t%s\n", file, hash)
	}
	return nil
}

// isUpToDate checks if a file has been modified since last mining.
func (m *Miner) isUpToDate(absPath string, modTime time.Time) bool {
	stored, ok := mtimeIndex[absPath]
	if !ok {
		return false
	}
	storedTime, err := time.Parse(time.RFC3339, stored)
	if err != nil {
		return false
	}
	return !modTime.After(storedTime)
}

// updateMtime updates the mtime index for a file.
func updateMtime(absPath string, modTime time.Time) {
	mtimeIndex[absPath] = modTime.Format(time.RFC3339)
}

var contentHashIndex = make(map[string]string)

func ResetContentHashIndex() {
	contentHashIndex = make(map[string]string)
}

func RegisterContentHash(absPath string, hash string) {
	contentHashIndex[absPath] = hash
}

func IsContentAlreadyMined(absPath string, currentHash string) bool {
	stored, ok := contentHashIndex[absPath]
	if !ok {
		return false
	}
	return stored == currentHash
}

// chunkContent splits content into overlapping chunks.
// It tries to split on paragraph/line boundaries for more semantically meaningful chunks.
func chunkContent(content string, chunkSize, overlap int) []string {
	// Clean up
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return []string{}
	}

	if len(content) <= chunkSize {
		return []string{content}
	}

	var chunks []string
	start := 0

	for start < len(content) {
		end := min(start+chunkSize, len(content))

		// Try to break at paragraph boundary (double newline)
		if end < len(content) {
			newlinePos := strings.LastIndex(string(content[start:end]), "\n\n")
			if newlinePos != -1 && newlinePos > chunkSize/2 {
				// Found a good paragraph break
				end = start + newlinePos
			} else {
				// Try to break at line boundary (single newline)
				newlinePos = strings.LastIndex(string(content[start:end]), "\n")
				if newlinePos != -1 && newlinePos > chunkSize/2 {
					end = start + newlinePos
				}
			}
		}

		chunk := strings.TrimSpace(content[start:end])
		if len(chunk) >= 50 { // MIN_CHUNK_SIZE equivalent
			chunks = append(chunks, chunk)
		}

		if end == len(content) {
			break
		}

		// Move start position, accounting for overlap
		start = max(end-overlap, 0)
	}

	return chunks
}

// detectRoomFromPath determines the room from the file path.
func detectRoomFromPath(path string) string {
	dir := filepath.Dir(path)
	parts := strings.SplitSeq(dir, string(filepath.Separator))

	for part := range parts {
		lower := strings.ToLower(part)
		switch {
		case containsAny(lower, "frontend", "ui", "components", "views", "widgets"):
			return "frontend"
		case containsAny(lower, "backend", "api", "server", "handlers", "controllers"):
			return "backend"
		case containsAny(lower, "docs", "documentation", "readme", "wiki"):
			return "documentation"
		case containsAny(lower, "test", "tests", "spec", "specs", "__tests__"):
			return "testing"
		case containsAny(lower, "config", "cfg", "settings", "conf"):
			return "configuration"
		case containsAny(lower, "script", "scripts", "bin", "tools"):
			return "scripts"
		case containsAny(lower, "design", "assets", "styles", "css"):
			return "design"
		case containsAny(lower, "db", "database", "migrations", "schema"):
			return "database"
		case containsAny(lower, "infra", "infrastructure", "deploy", "k8s", "docker"):
			return "infrastructure"
		}
	}

	// Fallback: check filename itself
	base := strings.ToLower(filepath.Base(path))
	switch {
	case containsAny(base, "frontend", "ui", "component", "view", "widget"):
		return "frontend"
	case containsAny(base, "backend", "api", "server", "handler", "controller"):
		return "backend"
	case containsAny(base, "doc", "readme", "wiki"):
		return "documentation"
	case containsAny(base, "test", "spec"):
		return "testing"
	case containsAny(base, "config", "cfg", "settings"):
		return "configuration"
	case containsAny(base, "script", "tool"):
		return "scripts"
	case containsAny(base, "style", "css", "design", "asset"):
		return "design"
	case containsAny(base, "db", "migration", "schema"):
		return "database"
	case containsAny(base, "deploy", "docker", "k8s", "infra"):
		return "infrastructure"
	}

	return "general"
}

func containsAny(s string, items ...string) bool {
	for _, item := range items {
		if strings.Contains(s, item) {
			return true
		}
	}
	return false
}

// --- .gitignore support ---

type gitignorePattern struct {
	pattern  string
	negated  bool
	dirOnly  bool
	anchored bool
	prefix   string
	suffix   string
}

func loadGitignorePatterns(path string) ([]gitignorePattern, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var patterns []gitignorePattern
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		pattern := parseGitignoreLine(line)
		if pattern != nil {
			patterns = append(patterns, *pattern)
		}
	}
	return patterns, scanner.Err()
}

func parseGitignoreLine(line string) *gitignorePattern {
	negated := false
	if strings.HasPrefix(line, "!") {
		negated = true
		line = line[1:]
	}

	dirOnly := strings.HasSuffix(line, "/")
	if dirOnly {
		line = line[:len(line)-1]
	}

	anchored := strings.Contains(line, "/") && !strings.HasSuffix(line, "/")
	if anchored && !strings.HasPrefix(line, "/") {
		line = "/" + line
	}

	if strings.HasPrefix(line, "/") {
		anchored = true
		line = line[1:]
	}

	return &gitignorePattern{
		pattern:  line,
		negated:  negated,
		dirOnly:  dirOnly,
		anchored: anchored,
		prefix:   getPrefix(line),
		suffix:   getSuffix(line),
	}
}

func getPrefix(pattern string) string {
	if idx := strings.IndexAny(pattern, "*?["); idx >= 0 {
		return pattern[:idx]
	}
	return pattern
}

func getSuffix(pattern string) string {
	if idx := strings.LastIndexAny(pattern, "*?["); idx >= 0 {
		return pattern[idx+1:]
	}
	return pattern
}

func matchesGitignore(path, baseDir string, patterns []gitignorePattern) bool {
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return false
	}
	relPath = filepath.ToSlash(relPath)

	matched := false
	for _, p := range patterns {
		if p.negated {
			if matchPattern(relPath, p) {
				matched = false
			}
		} else {
			if matchPattern(relPath, p) {
				matched = true
			}
		}
	}
	return matched
}

func matchPattern(path string, p gitignorePattern) bool {
	if p.anchored {
		return matchGlob(path, p.pattern)
	}
	// Try matching against any path component
	parts := strings.Split(path, "/")
	for i := range parts {
		subPath := strings.Join(parts[i:], "/")
		if matchGlob(subPath, p.pattern) {
			return true
		}
	}
	return false
}

func matchGlob(path, pattern string) bool {
	pattern = strings.ReplaceAll(pattern, "**/", "**/")
	pattern = strings.ReplaceAll(pattern, "**", "**")

	var patternRegex strings.Builder
	patternRegex.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				patternRegex.WriteString(".*")
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
				}
			} else {
				patternRegex.WriteString("[^/]*")
			}
		case '?':
			patternRegex.WriteString("[^/]")
		case '.':
			patternRegex.WriteString("\\.")
		case '[':
			patternRegex.WriteString("[")
		case ']':
			patternRegex.WriteString("]")
		case '\\':
			if i+1 < len(pattern) {
				i++
				patternRegex.WriteString("\\" + string(pattern[i]))
			}
		default:
			patternRegex.WriteString(string(c))
		}
	}
	patternRegex.WriteString("$")

	matched, _ := filepath.Match(strings.ReplaceAll(patternRegex.String(), ".*", "*"), path)
	// Simple fallback: use basic string matching
	if !matched {
		matched = strings.Contains(path, strings.ReplaceAll(pattern, "**", ""))
	}
	return matched
}

// GenerateContentHash generates a SHA256 hash of file content for dedup.
func GenerateContentHash(content string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}

func generateID() string {
	return fmt.Sprintf("d_%d", time.Now().UnixNano())
}

var roomKeywords = map[string][]string{
	"frontend":       {"component", "react", "vue", "angular", "jsx", "tsx", "css", "html", "widget", "ui", "button", "render", "styled"},
	"backend":        {"api", "server", "handler", "controller", "endpoint", "route", "middleware", "grpc", "rest", "http"},
	"documentation":  {"readme", "docs", "wiki", "guide", "tutorial", "documentation", "changelog", "md"},
	"testing":        {"test", "spec", "assert", "mock", "stub", "suite", "benchmark", "coverage"},
	"configuration":  {"config", "setting", "env", "yaml", "toml", "json", "ini"},
	"scripts":        {"script", "bin", "tool", "cli", "makefile", "pipeline", "workflow"},
	"design":         {"design", "asset", "style", "theme", "layout", "mockup", "figma"},
	"database":       {"database", "schema", "migration", "sql", "query", "table", "index", "db"},
	"infrastructure": {"docker", "k8s", "deploy", "infra", "terraform", "ansible", "ci/cd", "container"},
}

func DetectRoomFromContent(content string) string {
	contentLen := min(len(content), 2000)
	contentLower := strings.ToLower(content[:contentLen])

	scores := make(map[string]int)
	for room, keywords := range roomKeywords {
		score := 0
		for _, keyword := range keywords {
			score += strings.Count(contentLower, keyword)
		}
		if score > 0 {
			scores[room] = score
		}
	}

	if len(scores) == 0 {
		return "general"
	}

	bestRoom := "general"
	bestScore := 0
	for room, score := range scores {
		if score > bestScore {
			bestScore = score
			bestRoom = room
		}
	}
	return bestRoom
}

func DetectRoomCombined(path string, content string) string {
	room := detectRoomFromPath(path)
	if room != "general" {
		return room
	}
	return DetectRoomFromContent(content)
}
