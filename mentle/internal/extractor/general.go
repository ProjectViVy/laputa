// Package extractor provides memory extraction from text using pattern matching.
// It identifies decisions, preferences, milestones, problems, and emotional content.
package extractor

import (
	"regexp"
	"strings"
)

type Memory struct {
	Content    string `json:"content"`
	MemoryType string `json:"memory_type"`
	ChunkIndex int    `json:"chunk_index"`
}

type Extractor struct {
	decisionPatterns   []*regexp.Regexp
	preferencePatterns []*regexp.Regexp
	milestonePatterns  []*regexp.Regexp
	problemPatterns    []*regexp.Regexp
	emotionalPatterns  []*regexp.Regexp
	positiveWords      map[string]bool
	negativeWords      map[string]bool
}

func NewExtractor() *Extractor {
	e := &Extractor{
		positiveWords: map[string]bool{
			"pride": true, "proud": true, "joy": true, "happy": true,
			"love": true, "loving": true, "beautiful": true, "amazing": true,
			"wonderful": true, "incredible": true, "fantastic": true,
			"brilliant": true, "perfect": true, "excited": true,
			"thrilled": true, "grateful": true, "warm": true,
			"breakthrough": true, "success": true, "works": true,
			"working": true, "solved": true, "fixed": true, "nailed": true,
			"heart": true, "hug": true, "precious": true, "adore": true,
		},
		negativeWords: map[string]bool{
			"bug": true, "error": true, "crash": true, "crashing": true,
			"crashed": true, "fail": true, "failed": true, "failing": true,
			"failure": true, "broken": true, "broke": true, "breaking": true,
			"breaks": true, "issue": true, "problem": true, "wrong": true,
			"stuck": true, "blocked": true, "unable": true, "impossible": true,
			"missing": true, "terrible": true, "horrible": true,
			"awful": true, "worse": true, "worst": true, "panic": true,
			"disaster": true, "mess": true,
		},
	}

	e.decisionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\blet'?s (use|go with|try|pick|choose|switch to)\b`),
		regexp.MustCompile(`(?i)\bwe (should|decided|chose|went with|picked|settled on)\b`),
		regexp.MustCompile(`(?i)\bi'?m going (to|with)\b`),
		regexp.MustCompile(`(?i)\bbetter (to|than|approach|option|choice)\b`),
		regexp.MustCompile(`(?i)\binstead of\b`),
		regexp.MustCompile(`(?i)\brather than\b`),
		regexp.MustCompile(`(?i)\bthe reason (is|was|being)\b`),
		regexp.MustCompile(`(?i)\bbecause\b`),
		regexp.MustCompile(`(?i)\btrade-?off\b`),
		regexp.MustCompile(`(?i)\bpros and cons\b`),
		regexp.MustCompile(`(?i)\bover\b.*\bbecause\b`),
		regexp.MustCompile(`(?i)\barchitecture\b`),
		regexp.MustCompile(`(?i)\bapproach\b`),
		regexp.MustCompile(`(?i)\bstrategy\b`),
		regexp.MustCompile(`(?i)\bpattern\b`),
		regexp.MustCompile(`(?i)\bstack\b`),
		regexp.MustCompile(`(?i)\bframework\b`),
		regexp.MustCompile(`(?i)\binfrastructure\b`),
		regexp.MustCompile(`(?i)\bset (it |this )?to\b`),
		regexp.MustCompile(`(?i)\bconfigure\b`),
		regexp.MustCompile(`(?i)\bdefault\b`),
	}

	e.preferencePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bi prefer\b`),
		regexp.MustCompile(`(?i)\balways use\b`),
		regexp.MustCompile(`(?i)\bnever use\b`),
		regexp.MustCompile(`(?i)\bdon'?t (ever |like to )?(use|do|mock|stub|import)\b`),
		regexp.MustCompile(`(?i)\bi like (to|when|how)\b`),
		regexp.MustCompile(`(?i)\bi hate (when|how|it when)\b`),
		regexp.MustCompile(`(?i)\bplease (always|never|don'?t)\b`),
		regexp.MustCompile(`(?i)\bmy (rule|preference|style|convention) is\b`),
		regexp.MustCompile(`(?i)\bwe (always|never)\b`),
		regexp.MustCompile(`(?i)\bsnake_?case\b`),
		regexp.MustCompile(`(?i)\bcamel_?case\b`),
		regexp.MustCompile(`(?i)\bfunctional\b.*\bstyle\b`),
		regexp.MustCompile(`(?i)\bimperative\b`),
		regexp.MustCompile(`(?i)\btabs\b.*\bspaces\b`),
		regexp.MustCompile(`(?i)\bspaces\b.*\btabs\b`),
		regexp.MustCompile(`(?i)\buse\b.*\binstead of\b`),
	}

	e.milestonePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bit works\b`),
		regexp.MustCompile(`(?i)\bit worked\b`),
		regexp.MustCompile(`(?i)\bgot it working\b`),
		regexp.MustCompile(`(?i)\bfixed\b`),
		regexp.MustCompile(`(?i)\bsolved\b`),
		regexp.MustCompile(`(?i)\bbreakthrough\b`),
		regexp.MustCompile(`(?i)\bfigured (it )?out\b`),
		regexp.MustCompile(`(?i)\bnailed it\b`),
		regexp.MustCompile(`(?i)\bcracked (it|the)\b`),
		regexp.MustCompile(`(?i)\bfinally\b`),
		regexp.MustCompile(`(?i)\bfirst time\b`),
		regexp.MustCompile(`(?i)\bfirst ever\b`),
		regexp.MustCompile(`(?i)\bnever (done|been|had) before\b`),
		regexp.MustCompile(`(?i)\bdiscovered\b`),
		regexp.MustCompile(`(?i)\brealized\b`),
		regexp.MustCompile(`(?i)\bfound (out|that)\b`),
		regexp.MustCompile(`(?i)\bturns out\b`),
		regexp.MustCompile(`(?i)\bthe key (is|was|insight)\b`),
		regexp.MustCompile(`(?i)\bthe trick (is|was)\b`),
		regexp.MustCompile(`(?i)\bnow i (understand|see|get it)\b`),
		regexp.MustCompile(`(?i)\bbuilt\b`),
		regexp.MustCompile(`(?i)\bcreated\b`),
		regexp.MustCompile(`(?i)\bimplemented\b`),
		regexp.MustCompile(`(?i)\bshipped\b`),
		regexp.MustCompile(`(?i)\blaunched\b`),
		regexp.MustCompile(`(?i)\bdeployed\b`),
		regexp.MustCompile(`(?i)\breleased\b`),
		regexp.MustCompile(`(?i)\bprototype\b`),
		regexp.MustCompile(`(?i)\bproof of concept\b`),
		regexp.MustCompile(`(?i)\bdemo\b`),
		regexp.MustCompile(`(?i)\bversion \d\b`),
		regexp.MustCompile(`(?i)\bv\d+\.\d+\b`),
		regexp.MustCompile(`(?i)\b\d+x (compression|faster|slower|better|improvement|reduction)\b`),
		regexp.MustCompile(`(?i)\b\d+% (reduction|improvement|faster|better|smaller)\b`),
	}

	e.problemPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bbroke\b`),
		regexp.MustCompile(`(?i)\bcrashed\b`),
		regexp.MustCompile(`(?i)\bfailed\b`),
		regexp.MustCompile(`(?i)\berror\b`),
		regexp.MustCompile(`(?i)\bexception\b`),
		regexp.MustCompile(`(?i)\bbug\b`),
		regexp.MustCompile(`(?i)\bissue\b`),
		regexp.MustCompile(`(?i)\bproblem\b`),
		regexp.MustCompile(`(?i)\bfix\b`),
		regexp.MustCompile(`(?i)\bworkaround\b`),
		regexp.MustCompile(`(?i)\broot cause\b`),
		regexp.MustCompile(`(?i)\bwhy\b`),
		regexp.MustCompile(`(?i)\bdoesn'?t work\b`),
		regexp.MustCompile(`(?i)\bnot working\b`),
		regexp.MustCompile(`(?i)\bwon'?t\b.*\bwork\b`),
		regexp.MustCompile(`(?i)\bkeeps? (failing|crashing|breaking|erroring)\b`),
		regexp.MustCompile(`(?i)\bthe (problem|issue|bug) (is|was)\b`),
		regexp.MustCompile(`(?i)\bthe fix (is|was)\b`),
		regexp.MustCompile(`(?i)\bthat'?s why\b`),
		regexp.MustCompile(`(?i)\bthe reason it\b`),
		regexp.MustCompile(`(?i)\bsolution (is|was)\b`),
		regexp.MustCompile(`(?i)\bthe answer (is|was)\b`),
		regexp.MustCompile(`(?i)\b(had|need) to\b.*\binstead\b`),
	}

	e.emotionalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bi feel\b`),
		regexp.MustCompile(`(?i)\bi was (sad|happy|angry|scared)\b`),
		regexp.MustCompile(`(?i)\bfeelings?\b`),
		regexp.MustCompile(`(?i)\bemotion\b`),
		regexp.MustCompile(`(?i)\bvulnerable\b`),
		regexp.MustCompile(`(?i)\bhonestly\b`),
		regexp.MustCompile(`(?i)\btruth is\b`),
		regexp.MustCompile(`(?i)\bthe thing is\b`),
		regexp.MustCompile(`(?i)\bi'?m (not |really )?sure\b`),
		regexp.MustCompile(`(?i)\bhope\b`),
		regexp.MustCompile(`(?i)\bfear\b`),
		regexp.MustCompile(`(?i)\bworry\b`),
		regexp.MustCompile(`(?i)\blove\b`),
		regexp.MustCompile(`(?i)\bproud\b`),
		regexp.MustCompile(`(?i)\bhurt\b`),
		regexp.MustCompile(`(?i)\bcry\b`),
		regexp.MustCompile(`(?i)\bcrying\b`),
		regexp.MustCompile(`(?i)\bmiss\b`),
		regexp.MustCompile(`(?i)\bsorry\b`),
		regexp.MustCompile(`(?i)\bgrateful\b`),
		regexp.MustCompile(`(?i)\bworried\b`),
		regexp.MustCompile(`(?i)\blonely\b`),
		regexp.MustCompile(`(?i)\bbeautiful\b`),
		regexp.MustCompile(`(?i)\bamazing\b`),
		regexp.MustCompile(`(?i)\bwonderful\b`),
		regexp.MustCompile(`(?i)\bi can'?t\b`),
		regexp.MustCompile(`(?i)\bi wish\b`),
		regexp.MustCompile(`(?i)\bi need\b`),
		regexp.MustCompile(`(?i)\bnever told anyone\b`),
		regexp.MustCompile(`(?i)\bnobody knows\b`),
	}

	return e
}

var codeLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s*[\$#]\s`),
	regexp.MustCompile(`^\s*(cd|source|echo|export|pip|npm|git|python|bash|curl|wget|mkdir|rm|cp|mv|ls|cat|grep|find|chmod|sudo|brew|docker)\s`),
	regexp.MustCompile("^\\s*```"),
	regexp.MustCompile(`^\s*(import|from|def|class|function|const|let|var|return)\s`),
	regexp.MustCompile(`^\s*[A-Z_]{2,}=`),
	regexp.MustCompile(`^\s*\|`),
	regexp.MustCompile(`^\s*[-]{2,}`),
	regexp.MustCompile(`^\s*[{}\[\]]\s*$`),
	regexp.MustCompile(`^\s*(if|for|while|try|except|elif|else:)\b`),
	regexp.MustCompile(`^\s*\w+\.\w+\(`),
	regexp.MustCompile(`^\s*\w+ = \w+\.\w+`),
}

var turnPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^>\s`),
	regexp.MustCompile(`(?i)^(Human|User|Q)\s*:`),
	regexp.MustCompile(`(?i)^(Assistant|AI|A|Claude|ChatGPT)\s*:`),
}

func (e *Extractor) ExtractMemories(text string, minConfidence float64) []Memory {
	var memories []Memory

	paragraphs := splitIntoSegments(text)

	for _, para := range paragraphs {
		if len(para) < 20 {
			continue
		}

		prose := extractProse(para)

		scores := make(map[string]float64)
		for memType, markers := range map[string][]*regexp.Regexp{
			"decision":   e.decisionPatterns,
			"preference": e.preferencePatterns,
			"milestone":  e.milestonePatterns,
			"problem":    e.problemPatterns,
			"emotional":  e.emotionalPatterns,
		} {
			score, _ := scoreMarkers(prose, markers)
			if score > 0 {
				scores[memType] = score
			}
		}

		if len(scores) == 0 {
			continue
		}

		lengthBonus := 0.0
		if len(para) > 500 {
			lengthBonus = 2
		} else if len(para) > 200 {
			lengthBonus = 1
		}

		maxType := ""
		maxScore := 0.0
		for t, s := range scores {
			if s > maxScore {
				maxScore = s
				maxType = t
			}
		}
		maxScore += lengthBonus

		maxType = disambiguate(maxType, prose, scores)

		confidence := maxScore / 5.0
		if confidence > 1.0 {
			confidence = 1.0
		}
		if confidence < minConfidence {
			continue
		}

		memories = append(memories, Memory{
			Content:    strings.TrimSpace(para),
			MemoryType: maxType,
			ChunkIndex: len(memories),
		})
	}

	return memories
}

func (e *Extractor) getSentiment(text string) string {
	words := make(map[string]bool)
	for _, w := range regexp.MustCompile(`\b\w+\b`).FindAllString(strings.ToLower(text), -1) {
		words[w] = true
	}

	pos := 0
	for w := range words {
		if e.positiveWords[w] {
			pos++
		}
	}

	neg := 0
	for w := range words {
		if e.negativeWords[w] {
			neg++
		}
	}

	if pos > neg {
		return "positive"
	}
	if neg > pos {
		return "negative"
	}
	return "neutral"
}

func hasResolution(text string) bool {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bfixed\b`),
		regexp.MustCompile(`(?i)\bsolved\b`),
		regexp.MustCompile(`(?i)\bresolved\b`),
		regexp.MustCompile(`(?i)\bpatched\b`),
		regexp.MustCompile(`(?i)\bgot it working\b`),
		regexp.MustCompile(`(?i)\bit works\b`),
		regexp.MustCompile(`(?i)\bnailed it\b`),
		regexp.MustCompile(`(?i)\bfigured (it )?out\b`),
		regexp.MustCompile(`(?i)\bthe (fix|answer|solution)\b`),
	}

	textLower := strings.ToLower(text)
	for _, p := range patterns {
		if p.MatchString(textLower) {
			return true
		}
	}
	return false
}

func disambiguate(memoryType string, text string, scores map[string]float64) string {
	extractor := NewExtractor()
	sentiment := extractor.getSentiment(text)

	if memoryType == "problem" && hasResolution(text) {
		if scores["emotional"] > 0 && sentiment == "positive" {
			return "emotional"
		}
		return "milestone"
	}

	if memoryType == "problem" && sentiment == "positive" {
		if scores["milestone"] > 0 {
			return "milestone"
		}
		if scores["emotional"] > 0 {
			return "emotional"
		}
	}

	return memoryType
}

func isCodeLine(line string) bool {
	stripped := strings.TrimSpace(line)
	if stripped == "" {
		return false
	}

	for _, pattern := range codeLinePatterns {
		if pattern.MatchString(stripped) {
			return true
		}
	}

	alphaCount := 0
	for _, c := range stripped {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			alphaCount++
		}
	}
	alphaRatio := float64(alphaCount) / float64(len(stripped))
	if alphaRatio < 0.4 && len(stripped) > 10 {
		return true
	}

	return false
}

func extractProse(text string) string {
	lines := strings.Split(text, "\n")
	var prose []string
	inCode := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
			continue
		}

		if inCode {
			continue
		}

		if !isCodeLine(line) {
			prose = append(prose, line)
		}
	}

	result := strings.Join(prose, "\n")
	if result == "" {
		return text
	}
	return result
}

func scoreMarkers(text string, markers []*regexp.Regexp) (float64, []string) {
	textLower := strings.ToLower(text)
	score := 0.0
	var keywords []string

	for _, marker := range markers {
		matches := marker.FindAllString(textLower, -1)
		if len(matches) > 0 {
			score += float64(len(matches))
			keywords = append(keywords, matches...)
		}
	}

	seen := make(map[string]bool)
	unique := []string{}
	for _, k := range keywords {
		if !seen[k] {
			seen[k] = true
			unique = append(unique, k)
		}
	}

	return score, unique
}

func splitIntoSegments(text string) []string {
	lines := strings.Split(text, "\n")

	turnCount := 0
	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		for _, pat := range turnPatterns {
			if pat.MatchString(stripped) {
				turnCount++
				break
			}
		}
	}

	if turnCount >= 3 {
		return splitByTurns(lines)
	}

	paragraphs := []string{}
	for p := range strings.SplitSeq(text, "\n\n") {
		if strings.TrimSpace(p) != "" {
			paragraphs = append(paragraphs, strings.TrimSpace(p))
		}
	}

	if len(paragraphs) <= 1 && len(lines) > 20 {
		var segments []string
		for i := 0; i < len(lines); i += 25 {
			end := min(i+25, len(lines))
			group := strings.Join(lines[i:end], "\n")
			if strings.TrimSpace(group) != "" {
				segments = append(segments, strings.TrimSpace(group))
			}
		}
		return segments
	}

	return paragraphs
}

func splitByTurns(lines []string) []string {
	var segments []string
	var current []string

	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		isTurn := false
		for _, pat := range turnPatterns {
			if pat.MatchString(stripped) {
				isTurn = true
				break
			}
		}

		if isTurn && len(current) > 0 {
			segments = append(segments, strings.Join(current, "\n"))
			current = []string{line}
		} else {
			current = append(current, line)
		}
	}

	if len(current) > 0 {
		segments = append(segments, strings.Join(current, "\n"))
	}

	return segments
}

func splitSentences(text string) []string {
	re := regexp.MustCompile(`[^.!?\n]+`)
	return re.FindAllString(text, -1)
}
