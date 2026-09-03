package embedder

import "time"

// Identity represents the exact characteristics of the embedding model used,
// to ensure vectors can be matched correctly in the database.
type Identity struct {
	Model      string    `json:"model"`
	Dimension  int       `json:"dimension"`
	Metric     string    `json:"metric"`
	Normalize  bool      `json:"normalize"`
	Provider   string    `json:"provider"`
	Version    string    `json:"version"`
	CapturedAt time.Time `json:"captured_at"`
}
