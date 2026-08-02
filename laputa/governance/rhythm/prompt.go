package rhythm

import (
	"encoding/json"
	"fmt"
)

// BuildPrompt constructs an LLM prompt from a Laputa snapshot.
func BuildPrompt(kind RhythmKind, snapshot map[string]any) string {
	sectionsJSON, _ := json.Marshal(snapshot)
	return fmt.Sprintf(`You are an autonomous rhythm reporter for an AI agent.

Cadence: %s

Laputa snapshot:
%s

Generate a structured report with: title, summary, highlights, open_questions.
Respond in JSON matching ReportResult.`, kind, string(sectionsJSON))
}
