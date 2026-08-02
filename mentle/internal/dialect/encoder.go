// Package dialect implements AAAK (Aphantix Abstraction Annotating Kit) compression.
// It compresses memory entries into compact, semantically-rich strings.
package dialect

import (
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"
)

var emotionCodes = map[string]string{
	"vulnerability": "vul", "vulnerable": "vul",
	"joy": "joy", "joyful": "joy",
	"fear": "fear", "mild_fear": "fear",
	"trust": "trust", "trust_building": "trust",
	"grief": "grief", "raw_grief": "grief",
	"wonder": "wonder", "philosophical_wonder": "wonder",
	"rage": "rage", "anger": "rage",
	"love": "love", "devotion": "love",
	"hope":    "hope",
	"despair": "despair", "hopelessness": "despair",
	"peace": "peace",
	"humor": "humor", "dark_humor": "humor",
	"tenderness":  "tender",
	"raw_honesty": "raw", "brutal_honesty": "raw",
	"self_doubt": "doubt",
	"anxiety":    "anx", "anxious": "anx",
	"exhaustion": "exhaust",
	"conviction": "convict", "quiet_passion": "passion",
	"warmth":        "warmth",
	"curiosity":     "curious",
	"gratitude":     "grat",
	"frustration":   "frust",
	"confusion":     "confuse",
	"satisfaction":  "satis",
	"excitement":    "excite",
	"determination": "determ",
	"surprise":      "surprise",
}

var emotionSignals = map[string]string{
	"decided": "determ", "prefer": "convict",
	"worried": "anx", "excited": "excite",
	"frustrated": "frust", "confused": "confuse",
	"love": "love", "hate": "rage",
	"hope": "hope", "fear": "fear",
	"trust": "trust", "happy": "joy",
	"sad": "grief", "surprised": "surprise",
	"grateful": "grat", "curious": "curious",
	"wonder": "wonder", "relieved": "relief",
	"satisf": "satis", "disappoint": "grief",
	"concern": "anx",
}

var flagSignals = map[string]string{
	"decided": "DECISION", "chose": "DECISION",
	"switched": "DECISION", "migrated": "DECISION",
	"replaced": "DECISION", "instead of": "DECISION",
	"because": "DECISION",
	"founded": "ORIGIN", "created": "ORIGIN",
	"started": "ORIGIN", "born": "ORIGIN",
	"launched": "ORIGIN", "first time": "ORIGIN",
	"core": "CORE", "fundamental": "CORE",
	"essential": "CORE", "principle": "CORE",
	"belief": "CORE", "always": "CORE",
	"never forget":  "CORE",
	"turning point": "PIVOT", "changed everything": "PIVOT",
	"realized": "PIVOT", "breakthrough": "PIVOT",
	"epiphany": "PIVOT",
	"api":      "TECHNICAL", "database": "TECHNICAL",
	"architecture": "TECHNICAL", "deploy": "TECHNICAL",
	"infrastructure": "TECHNICAL", "algorithm": "TECHNICAL",
	"framework": "TECHNICAL", "server": "TECHNICAL",
	"config": "TECHNICAL",
}

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true,
	"have": true, "has": true, "had": true, "do": true,
	"does": true, "did": true, "will": true, "would": true,
	"could": true, "should": true, "may": true, "might": true,
	"shall": true, "can": true, "to": true, "of": true,
	"in": true, "for": true, "on": true, "with": true,
	"at": true, "by": true, "from": true, "as": true,
	"into": true, "about": true, "between": true, "through": true,
	"during": true, "before": true, "after": true, "above": true,
	"below": true, "up": true, "down": true, "out": true,
	"off": true, "over": true, "under": true, "again": true,
	"further": true, "then": true, "once": true, "here": true,
	"there": true, "when": true, "where": true, "why": true,
	"how": true, "all": true, "each": true, "every": true,
	"both": true, "few": true, "more": true, "most": true,
	"other": true, "some": true, "such": true, "no": true,
	"nor": true, "not": true, "only": true, "own": true,
	"same": true, "so": true, "than": true, "too": true,
	"very": true, "just": true, "don": true, "now": true,
	"and": true, "but": true, "or": true, "if": true,
	"while": true, "that": true, "this": true, "these": true,
	"those": true, "it": true, "its": true, "i": true,
	"we": true, "you": true, "he": true, "she": true,
	"they": true, "me": true, "him": true, "her": true,
	"us": true, "them": true, "my": true, "your": true,
	"his": true, "our": true, "their": true, "what": true,
	"which": true, "who": true, "whom": true, "also": true,
	"much": true, "many": true, "like": true, "because": true,
	"since": true, "get": true, "got": true, "use": true,
	"used": true, "using": true, "make": true, "made": true,
	"thing": true, "things": true, "way": true, "well": true,
	"really": true, "want": true, "need": true,
}

var emotionalQuoteWords = map[string]bool{
	"love": true, "fear": true, "remember": true, "soul": true,
	"feel": true, "stupid": true, "scared": true, "beautiful": true,
	"destroy": true, "respect": true, "trust": true, "consciousness": true,
	"alive": true, "forget": true, "waiting": true, "peace": true,
	"matter": true, "real": true, "guilt": true, "escape": true,
	"rest": true, "hope": true, "dream": true, "lost": true,
	"found": true,
}

type Encoder struct {
	entityCodes map[string]string
	skipNames   []string
}

func NewEncoder() *Encoder {
	return &Encoder{
		entityCodes: make(map[string]string),
		skipNames:   []string{},
	}
}

func (e *Encoder) SetEntityCodes(codes map[string]string) {
	maps.Copy(e.entityCodes, codes)
}

func (e *Encoder) Compress(text string, metadata map[string]string) string {
	entities := e.detectEntities(text)
	entityStr := strings.Join(entities[:min(3, len(entities))], "+")
	if entityStr == "" {
		entityStr = "???"
	}

	topics := e.extractTopics(text, 3)
	topicStr := strings.Join(topics, "_")
	if topicStr == "" {
		topicStr = "misc"
	}

	quote := e.extractKeyQuote(text)
	if quote == "" {
		quote = e.extractKeySentence(text)
	}

	emotions := e.detectEmotions(text)
	emotionStr := strings.Join(emotions, "+")

	flags := e.detectFlags(text)
	flagStr := strings.Join(flags, "+")

	var parts []string
	parts = append(parts, "0:"+entityStr)
	parts = append(parts, topicStr)
	if quote != "" {
		parts = append(parts, "\""+quote+"\"")
	}
	if emotionStr != "" {
		parts = append(parts, emotionStr)
	}
	if flagStr != "" {
		parts = append(parts, flagStr)
	}

	return strings.Join(parts, "|")
}

func (e *Encoder) detectEmotions(text string) []string {
	textLower := strings.ToLower(text)
	var emotions []string
	seen := make(map[string]bool)

	for keyword, code := range emotionSignals {
		if strings.Contains(textLower, keyword) && !seen[code] {
			emotions = append(emotions, code)
			seen[code] = true
		}
	}

	return emotions[:min(3, len(emotions))]
}

func (e *Encoder) detectFlags(text string) []string {
	textLower := strings.ToLower(text)
	var flags []string
	seen := make(map[string]bool)

	for keyword, flag := range flagSignals {
		if strings.Contains(textLower, keyword) && !seen[flag] {
			flags = append(flags, flag)
			seen[flag] = true
		}
	}

	return flags[:min(3, len(flags))]
}

func (e *Encoder) extractTopics(text string, maxTopics int) []string {
	re := regexp.MustCompile(`[a-zA-Z][a-zA-Z_-]{2,}`)
	words := re.FindAllString(text, -1)

	freq := make(map[string]int)
	for _, w := range words {
		wLower := strings.ToLower(w)
		if stopWords[wLower] || len(wLower) < 3 {
			continue
		}
		freq[wLower]++

		if unicode.IsUpper(rune(w[0])) && wLower == strings.ToLower(w) {
			freq[wLower] += 2
		}

		if strings.Contains(w, "_") || strings.Contains(w, "-") {
			freq[wLower] += 2
		} else {
			hasUpper := false
			for _, c := range w[1:] {
				if unicode.IsUpper(c) {
					hasUpper = true
					break
				}
			}
			if hasUpper {
				freq[wLower] += 2
			}
		}
	}

	type pair struct {
		word string
		freq int
	}
	var ranked []pair
	for w, f := range freq {
		ranked = append(ranked, pair{w, f})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].freq > ranked[j].freq })

	var topics []string
	for i := 0; i < min(maxTopics, len(ranked)); i++ {
		topics = append(topics, ranked[i].word)
	}
	return topics
}

func (e *Encoder) extractKeySentence(text string) string {
	sentences := regexp.MustCompile(`[^.!?\n]+`).FindAllString(text, -1)

	type scored struct {
		text  string
		score int
	}
	var scoredSentences []scored

	decisionWords := map[string]bool{
		"decided": true, "because": true, "instead": true,
		"prefer": true, "switched": true, "chose": true,
		"realized": true, "important": true, "key": true,
		"critical": true, "discovered": true, "learned": true,
		"conclusion": true, "solution": true, "reason": true,
		"why": true, "breakthrough": true, "insight": true,
	}

	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) < 10 {
			continue
		}
		score := 0
		sLower := strings.ToLower(s)
		for w := range decisionWords {
			if strings.Contains(sLower, w) {
				score += 2
			}
		}
		if len(s) < 80 {
			score++
		}
		if len(s) > 150 {
			score -= 2
		}
		scoredSentences = append(scoredSentences, scored{s, score})
	}

	if len(scoredSentences) == 0 {
		return ""
	}

	sort.Slice(scoredSentences, func(i, j int) bool {
		return scoredSentences[i].score > scoredSentences[j].score
	})

	best := scoredSentences[0].text
	if len(best) > 55 {
		best = best[:52] + "..."
	}
	return best
}

func (e *Encoder) extractKeyQuote(text string) string {
	var quotes []string

	doubleQuotes := regexp.MustCompile(`"([^"]{8,55})"`)
	for _, m := range doubleQuotes.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			quotes = append(quotes, m[1])
		}
	}

	singleQuotes := regexp.MustCompile(`(?:^|[\s(])'([^']{8,55})'(?:[\s.,;:!?)]|$)`)
	for _, m := range singleQuotes.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			quotes = append(quotes, m[1])
		}
	}

	saidQuotes := regexp.MustCompile(`(?:says?|said|articulates?|reveals?|admits?|confesses?|asks?):\s*["\']?([^.!?]{10,55})[.!?]`)
	for _, m := range saidQuotes.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			quotes = append(quotes, m[1])
		}
	}

	if len(quotes) == 0 {
		return ""
	}

	type scoredQuote struct {
		text  string
		score int
	}
	var scoredQuotes []scoredQuote

	for _, q := range quotes {
		q = strings.TrimSpace(q)
		if len(q) < 8 {
			continue
		}

		score := 0
		if len(q) > 0 && (unicode.IsUpper(rune(q[0])) || strings.HasPrefix(q, "I ")) {
			score += 2
		}

		qLower := strings.ToLower(q)
		for w := range emotionalQuoteWords {
			if strings.Contains(qLower, w) {
				score += 2
			}
		}

		if len(q) > 20 {
			score += 1
		}

		if strings.HasPrefix(q, "The ") || strings.HasPrefix(q, "This ") || strings.HasPrefix(q, "She ") {
			score -= 2
		}

		scoredQuotes = append(scoredQuotes, scoredQuote{q, score})
	}

	if len(scoredQuotes) == 0 {
		return ""
	}

	sort.Slice(scoredQuotes, func(i, j int) bool {
		return scoredQuotes[i].score > scoredQuotes[j].score
	})

	return scoredQuotes[0].text
}

func (e *Encoder) detectEntities(text string) []string {
	var entities []string

	for name, code := range e.entityCodes {
		lowerName := strings.ToLower(name)
		if strings.Contains(strings.ToLower(text), lowerName) && code != "" {
			entities = append(entities, code)
		}
	}

	if len(entities) > 0 {
		sort.Strings(entities)
		return entities[:min(3, len(entities))]
	}

	re := regexp.MustCompile(`\b([A-Z][a-z]{2,})\b`)
	matches := re.FindAllStringSubmatch(text, -1)
	var seen []string
	for _, m := range matches {
		if len(seen) >= 3 {
			break
		}
		code := strings.ToUpper(m[1][:min(3, len(m[1]))])
		if !contains(seen, code) {
			seen = append(seen, code)
		}
	}
	return seen
}

func (e *Encoder) CountTokens(text string) int {
	words := strings.Fields(text)
	return max(1, int(float64(len(words))*1.3))
}

type CompressionStats struct {
	OriginalTokensEst int     `json:"original_tokens_est"`
	SummaryTokensEst  int     `json:"summary_tokens_est"`
	SizeRatio         float64 `json:"size_ratio"`
	OriginalChars     int     `json:"original_chars"`
	SummaryChars      int     `json:"summary_chars"`
	Note              string  `json:"note"`
}

func (e *Encoder) CompressionStats(originalText, compressed string) CompressionStats {
	origTokens := e.CountTokens(originalText)
	compTokens := e.CountTokens(compressed)
	ratio := float64(origTokens) / float64(max(compTokens, 1))

	return CompressionStats{
		OriginalTokensEst: origTokens,
		SummaryTokensEst:  compTokens,
		SizeRatio:         ratio,
		OriginalChars:     len(originalText),
		SummaryChars:      len(compressed),
		Note:              "Estimates only. AAAK is lossy summarization.",
	}
}

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
