package memory

import (
	"context"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"time"
)

type Record struct {
	ID         string                `json:"id"`
	Scope      evolution.Scope       `json:"scope"`
	Revision   uint64                `json:"revision"`
	Body       string                `json:"body"`
	Status     string                `json:"status"`
	Kind       string                `json:"kind"`
	Tags       []string              `json:"tags"`
	Sources    []evolution.SourceRef `json:"sources"`
	OccurredAt time.Time             `json:"occurred_at"`
	CapturedAt time.Time             `json:"captured_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}
type ListRequest struct {
	Scopes     []evolution.Scope `json:"scopes,omitempty"`
	Collection string            `json:"collection,omitempty"`
	Cursor     string            `json:"cursor,omitempty"`
	Limit      int               `json:"limit,omitempty"`
}
type RecordPage struct {
	Items      []Record `json:"items"`
	NextCursor string   `json:"next_cursor"`
	Truncated  bool     `json:"truncated,omitempty"`
}
type Collection struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}
type ManagementBackend interface {
	List(context.Context, ListRequest) (RecordPage, error)
	Get(context.Context, string) (Record, error)
	Collections(context.Context, []evolution.Scope) ([]Collection, error)
}
