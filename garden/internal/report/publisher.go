package report

import "context"

// Publisher is retained as a narrow extension seam for callers that need an
// explicit notification after the Garden SQLite commit. Report authority is
// the Service database; implementations must not write Laputa authority.
type Publisher interface {
	Publish(context.Context, Report) error
}
