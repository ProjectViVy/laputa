// Package entity provides entity detection and classification for memory palace.
// It identifies people and projects from text based on linguistic patterns.
package entity

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
	"are": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true,
	"might": true, "must": true, "shall": true, "can": true, "this": true,
	"that": true, "these": true, "those": true, "it": true, "its": true,
	"they": true, "them": true, "their": true, "we": true, "our": true,
	"you": true, "your": true, "i": true, "my": true, "me": true,
	"he": true, "she": true, "his": true, "her": true, "who": true,
	"what": true, "when": true, "where": true, "why": true, "how": true,
	"which": true, "if": true, "then": true, "so": true, "not": true,
	"no": true, "yes": true, "ok": true, "okay": true, "just": true,
	"very": true, "really": true, "also": true, "already": true, "still": true,
	"even": true, "only": true, "here": true, "there": true, "now": true,
	"too": true, "up": true, "out": true, "about": true, "like": true,
	"use": true, "get": true, "got": true, "make": true, "made": true,
	"take": true, "put": true, "come": true, "go": true, "see": true,
	"know": true, "think": true, "true": true, "false": true, "none": true,
	"null": true, "new": true, "old": true, "all": true, "any": true, "some": true,
	"return": true, "print": true, "def": true, "class": true,
	"import": true, "step": true, "usage": true, "run": true,
	"check": true, "find": true, "add": true, "set": true, "list": true,
	"args": true, "dict": true, "str": true, "int": true, "bool": true,
	"path": true, "file": true, "type": true, "name": true, "note": true,
	"example": true, "option": true, "result": true, "error": true, "warning": true,
	"info": true, "every": true, "each": true, "more": true, "less": true,
	"next": true, "last": true, "first": true, "second": true, "stack": true,
	"layer": true, "mode": true, "test": true, "stop": true, "start": true,
	"copy": true, "move": true, "source": true, "target": true, "output": true,
	"input": true, "data": true, "item": true, "key": true, "value": true,
	"returns": true, "raises": true, "yields": true, "self": true, "cls": true,
	"kwargs": true, "world": true, "well": true, "want": true, "topic": true,
	"choose": true, "social": true, "cars": true, "phones": true, "healthcare": true,
	"ex": true, "machina": true, "deus": true, "human": true, "humans": true,
	"people": true, "things": true, "something": true, "nothing": true,
	"everything": true, "anything": true, "someone": true, "everyone": true,
	"anyone": true, "way": true, "time": true, "day": true, "life": true,
	"place": true, "thing": true, "part": true, "kind": true, "sort": true,
	"case": true, "point": true, "idea": true, "fact": true, "sense": true,
	"question": true, "answer": true, "reason": true, "number": true, "version": true,
	"system": true, "hey": true, "hi": true, "hello": true, "thanks": true,
	"thank": true, "right": true, "let": true, "click": true, "hit": true,
	"press": true, "tap": true, "drag": true, "drop": true, "open": true,
	"close": true, "save": true, "load": true, "launch": true, "install": true,
	"download": true, "upload": true, "scroll": true, "select": true, "enter": true,
	"submit": true, "cancel": true, "confirm": true, "delete": true,
	"paste": true, "write": true, "read": true, "search": true,
	"show": true, "hide": true, "desktop": true, "documents": true, "downloads": true,
	"users": true, "home": true, "library": true, "applications": true,
	"preferences": true, "settings": true, "terminal": true, "actor": true,
	"vector": true, "remote": true, "control": true, "duration": true, "fetch": true,
	"agents": true, "tools": true, "others": true, "guards": true, "ethics": true,
	"regulation": true, "learning": true, "thinking": true, "memory": true,
	"language": true, "intelligence": true, "technology": true, "society": true,
	"culture": true, "future": true, "history": true, "science": true,
	"model": true, "models": true, "network": true, "networks": true,
	"training": true, "inference": true,
}

var personVerbPatterns = []string{
	`\b{name}\s+said\b`, `\b{name}\s+asked\b`, `\b{name}\s+told\b`,
	`\b{name}\s+replied\b`, `\b{name}\s+laughed\b`, `\b{name}\s+smiled\b`,
	`\b{name}\s+cried\b`, `\b{name}\s+felt\b`, `\b{name}\s+thinks?\b`,
	`\b{name}\s+wants?\b`, `\b{name}\s+loves?\b`, `\b{name}\s+hates?\b`,
	`\b{name}\s+knows?\b`, `\b{name}\s+decided\b`, `\b{name}\s+pushed\b`,
	`\b{name}\s+wrote\b`, `\bhey\s+{name}\b`, `\bthanks?\s+{name}\b`,
	`\bhi\s+{name}\b`, `\bdear\s+{name}\b`,
}

var pronounPatterns = []string{
	`\bshe\b`, `\bher\b`, `\bhers\b`, `\bhe\b`, `\bhim\b`, `\bhis\b`,
	`\bthey\b`, `\bthem\b`, `\btheir\b`,
}

var dialoguePatterns = []string{
	`^>\s*{name}[:\s]`,
	`^{name}:\s`,
	`^\[{name}\]`,
	`"{name}\s+said`,
}

var projectVerbPatterns = []string{
	`\bbuilding\s+{name}\b`, `\bbuilt\s+{name}\b`, `\bship(?:ping|ped)?\s+{name}\b`,
	`\blaunch(?:ing|ed)?\s+{name}\b`, `\bdeploy(?:ing|ed)?\s+{name}\b`,
	`\binstall(?:ing|ed)?\s+{name}\b`, `\bthe\s+{name}\s+architecture\b`,
	`\bthe\s+{name}\s+pipeline\b`, `\bthe\s+{name}\s+system\b`, `\bthe\s+{name}\s+repo\b`,
	`\b{name}\s+v\d+\b`, `\b{name}\.py\b`, `\b{name}-core\b`, `\b{name}-local\b`,
	`\bimport\s+{name}\b`, `\bpip\s+install\s+{name}\b`,
}

type Entity struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Confidence float64  `json:"confidence"`
	Frequency  int      `json:"frequency"`
	Signals    []string `json:"signals"`
}

type Detector struct{}

func NewDetector() *Detector {
	return &Detector{}
}

func (d *Detector) ExtractCandidates(text string) map[string]int {
	candidates := make(map[string]int)

	capitalized := regexp.MustCompile(`\b([A-Z][a-z]{1,19})\b`)
	for _, word := range capitalized.FindAllString(text, -1) {
		lower := strings.ToLower(word)
		if !stopWords[lower] && len(word) > 1 {
			candidates[word]++
		}
	}

	multiWord := regexp.MustCompile(`\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)+)\b`)
	for _, phrase := range multiWord.FindAllString(text, -1) {
		words := strings.Split(phrase, " ")
		if !anyStopWord(words) {
			candidates[phrase]++
		}
	}

	result := make(map[string]int)
	for name, count := range candidates {
		if count >= 3 {
			result[name] = count
		}
	}
	return result
}

func anyStopWord(words []string) bool {
	for _, w := range words {
		if stopWords[strings.ToLower(w)] {
			return true
		}
	}
	return false
}

func (d *Detector) ScoreEntity(name, text string) (personScore, projectScore int) {
	lines := strings.Split(text, "\n")
	nameEscaped := regexp.QuoteMeta(name)

	for _, pattern := range dialoguePatterns {
		compiled := regexp.MustCompile(strings.Replace(pattern, "{name}", nameEscaped, 1))
		matches := len(compiled.FindAllString(text, -1))
		personScore += matches * 3
	}

	for _, pattern := range personVerbPatterns {
		compiled := regexp.MustCompile(strings.Replace(pattern, "{name}", nameEscaped, 1))
		matches := len(compiled.FindAllString(text, -1))
		personScore += matches * 2
	}

	nameLower := strings.ToLower(name)
	var nameLineIndices []int
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), nameLower) {
			nameLineIndices = append(nameLineIndices, i)
		}
	}

	pronounHits := 0
	for _, idx := range nameLineIndices {
		start := max(idx-2, 0)
		end := min(idx+3, len(lines))
		window := strings.ToLower(strings.Join(lines[start:end], " "))
		for _, pronPat := range pronounPatterns {
			if regexp.MustCompile(pronPat).MatchString(window) {
				pronounHits++
				break
			}
		}
	}
	personScore += pronounHits * 2

	direct := regexp.MustCompile(fmt.Sprintf(`\bhey\s+%s\b|\bthanks?\s+%s\b|\bhi\s+%s\b`, nameEscaped, nameEscaped, nameEscaped))
	personScore += len(direct.FindAllString(text, -1)) * 4

	for _, pattern := range projectVerbPatterns {
		compiled := regexp.MustCompile(strings.Replace(pattern, "{name}", nameEscaped, 1))
		matches := len(compiled.FindAllString(text, -1))
		projectScore += matches * 2
	}

	versioned := regexp.MustCompile(fmt.Sprintf(`\b%v[-v]\w+`, nameEscaped))
	projectScore += len(versioned.FindAllString(text, -1)) * 3

	codeRef := regexp.MustCompile(fmt.Sprintf(`\b%v\.(py|js|ts|yaml|yml|json|sh)\b`, nameEscaped))
	projectScore += len(codeRef.FindAllString(text, -1)) * 3

	return personScore, projectScore
}

func (d *Detector) ClassifyEntity(name string, frequency int, personScore, projectScore int, personSignals, projectSignals []string) Entity {
	total := personScore + projectScore

	if total == 0 {
		confidence := 0.4
		if frequency > 10 {
			confidence = 0.5
		}
		return Entity{
			Name:       name,
			Type:       "uncertain",
			Confidence: confidence,
			Frequency:  frequency,
			Signals:    []string{fmt.Sprintf("appears %dx, no strong type signals", frequency)},
		}
	}

	personRatio := float64(personScore) / float64(total)

	signalCategories := make(map[string]bool)
	for _, s := range personSignals {
		if strings.Contains(s, "dialogue") {
			signalCategories["dialogue"] = true
		}
		if strings.Contains(s, "action") {
			signalCategories["action"] = true
		}
		if strings.Contains(s, "pronoun") {
			signalCategories["pronoun"] = true
		}
		if strings.Contains(s, "addressed") {
			signalCategories["addressed"] = true
		}
	}

	hasTwoSignalTypes := len(signalCategories) >= 2

	if personRatio >= 0.7 && hasTwoSignalTypes && personScore >= 5 {
		confidence := 0.5 + personRatio*0.5
		if confidence > 0.99 {
			confidence = 0.99
		}
		signals := personSignals
		if len(signals) == 0 {
			signals = []string{fmt.Sprintf("appears %dx", frequency)}
		}
		return Entity{
			Name:       name,
			Type:       "person",
			Confidence: confidence,
			Frequency:  frequency,
			Signals:    signals,
		}
	}

	if personRatio >= 0.7 && (!hasTwoSignalTypes || personScore < 5) {
		return Entity{
			Name:       name,
			Type:       "uncertain",
			Confidence: 0.4,
			Frequency:  frequency,
			Signals:    append(personSignals, fmt.Sprintf("appears %dx — pronoun-only match", frequency)),
		}
	}

	if personRatio <= 0.3 {
		confidence := 0.5 + (1-personRatio)*0.5
		if confidence > 0.99 {
			confidence = 0.99
		}
		signals := projectSignals
		if len(signals) == 0 {
			signals = []string{fmt.Sprintf("appears %dx", frequency)}
		}
		return Entity{
			Name:       name,
			Type:       "project",
			Confidence: confidence,
			Frequency:  frequency,
			Signals:    signals,
		}
	}

	signals := append(personSignals, projectSignals...)
	if len(signals) > 3 {
		signals = signals[:3]
	}
	signals = append(signals, "mixed signals — needs review")

	return Entity{
		Name:       name,
		Type:       "uncertain",
		Confidence: 0.5,
		Frequency:  frequency,
		Signals:    signals,
	}
}

func (d *Detector) getSignals(name, text string, lines []string, personScore, projectScore int) ([]string, []string) {
	var personSignals, projectSignals []string

	nameEscaped := regexp.QuoteMeta(name)

	for _, pattern := range dialoguePatterns {
		compiled := regexp.MustCompile(strings.Replace(pattern, "{name}", nameEscaped, 1))
		matches := len(compiled.FindAllString(text, -1))
		if matches > 0 {
			personSignals = append(personSignals, fmt.Sprintf("dialogue marker (%dx)", matches))
		}
	}

	for _, pattern := range personVerbPatterns {
		compiled := regexp.MustCompile(strings.Replace(pattern, "{name}", nameEscaped, 1))
		matches := len(compiled.FindAllString(text, -1))
		if matches > 0 {
			personSignals = append(personSignals, fmt.Sprintf("'%s ...' action (%dx)", name, matches))
		}
	}

	nameLower := strings.ToLower(name)
	var nameLineIndices []int
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), nameLower) {
			nameLineIndices = append(nameLineIndices, i)
		}
	}

	pronounHits := 0
	for _, idx := range nameLineIndices {
		start := max(idx-2, 0)
		end := min(idx+3, len(lines))
		window := strings.ToLower(strings.Join(lines[start:end], " "))
		for _, pronPat := range pronounPatterns {
			if regexp.MustCompile(pronPat).MatchString(window) {
				pronounHits++
				break
			}
		}
	}
	if pronounHits > 0 {
		personSignals = append(personSignals, fmt.Sprintf("pronoun nearby (%dx)", pronounHits))
	}

	direct := regexp.MustCompile(fmt.Sprintf(`\bhey\s+%s\b|\bthanks?\s+%s\b|\bhi\s+%s\b`, nameEscaped, nameEscaped, nameEscaped))
	directMatches := len(direct.FindAllString(text, -1))
	if directMatches > 0 {
		personSignals = append(personSignals, fmt.Sprintf("addressed directly (%dx)", directMatches))
	}

	for _, pattern := range projectVerbPatterns {
		compiled := regexp.MustCompile(strings.Replace(pattern, "{name}", nameEscaped, 1))
		matches := len(compiled.FindAllString(text, -1))
		if matches > 0 {
			projectSignals = append(projectSignals, fmt.Sprintf("project verb (%dx)", matches))
		}
	}

	versioned := regexp.MustCompile(fmt.Sprintf(`\b%v[-v]\w+`, nameEscaped))
	versionedMatches := len(versioned.FindAllString(text, -1))
	if versionedMatches > 0 {
		projectSignals = append(projectSignals, fmt.Sprintf("versioned/hyphenated (%dx)", versionedMatches))
	}

	codeRef := regexp.MustCompile(fmt.Sprintf(`\b%v\.(py|js|ts|yaml|yml|json|sh)\b`, nameEscaped))
	codeRefMatches := len(codeRef.FindAllString(text, -1))
	if codeRefMatches > 0 {
		projectSignals = append(projectSignals, fmt.Sprintf("code file reference (%dx)", codeRefMatches))
	}

	return personSignals, projectSignals
}

func (d *Detector) Detect(text string) []Entity {
	lines := strings.Split(text, "\n")
	candidates := d.ExtractCandidates(text)

	var entities []Entity
	for name, freq := range candidates {
		ps, prs := d.ScoreEntity(name, text)
		pSignals, projSignals := d.getSignals(name, text, lines, ps, prs)
		entity := d.ClassifyEntity(name, freq, ps, prs, pSignals, projSignals)
		entities = append(entities, entity)
	}

	return entities
}

func isCapitalized(s string) bool {
	for _, r := range s {
		return unicode.IsUpper(r)
	}
	return false
}
