package governance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrUnauthorized = errors.New("governance: unauthorized mutation")

type ActorRole string

const (
	ActorUser         ActorRole = "user"
	ActorAgent        ActorRole = "agent"
	ActorReportSystem ActorRole = "report_system"
	ActorSystem       ActorRole = "system"
)

type MutationRequest struct {
	Section   SectionName    `json:"section"`
	Action    string         `json:"action"` // "write" | "patch" | "delete"
	Actor     ActorRole      `json:"actor"`
	Reason    string         `json:"reason"`
	RequestID string         `json:"request_id"`
	Data      map[string]any `json:"data,omitempty"`
	Path      string         `json:"path,omitempty"`
	Value     any            `json:"value,omitempty"`
}

type AuditEntry struct {
	Sequence    int64       `json:"sequence"`
	Section     SectionName `json:"section"`
	Action      string      `json:"action"`
	Actor       string      `json:"actor"`
	Reason      string      `json:"reason"`
	RequestID   string      `json:"request_id"`
	RollbackRef string      `json:"rollback_ref"`
	Timestamp   string      `json:"timestamp"`
}

type AuditLog interface {
	Append(ctx context.Context, entry AuditEntry) error
	Recent(ctx context.Context, limit int) ([]AuditEntry, error)
}

type GovernedService struct {
	engine *Engine
	audit  AuditLog
}

func NewGovernedService(engine *Engine, audit AuditLog) *GovernedService {
	return &GovernedService{engine: engine, audit: audit}
}

func (g *GovernedService) GetSection(ctx context.Context, section SectionName) (map[string]any, error) {
	return g.engine.GetSection(ctx, section)
}

func (g *GovernedService) ListSections(ctx context.Context) ([]SectionName, error) {
	return g.engine.ListSections(ctx)
}

func (g *GovernedService) Mutate(ctx context.Context, req MutationRequest) error {
	info, ok := SectionRegistry[req.Section]
	if !ok {
		return fmt.Errorf("governance: unknown section %s", req.Section)
	}
	if err := authorize(info.WriteAuth, req.Actor); err != nil {
		return err
	}

	var rollbackRef string
	if g.audit != nil && requiresAudit(req.Section, req.Actor) {
		prev, err := g.engine.GetSection(ctx, req.Section)
		if err == nil {
			rollbackRef = stateHash(prev)
		}
	}

	switch req.Action {
	case "write":
		if err := g.engine.SetSection(ctx, req.Section, req.Data); err != nil {
			return err
		}
	case "patch":
		if err := g.engine.UpdateSection(ctx, req.Section, req.Path, req.Value); err != nil {
			return err
		}
	case "delete":
		if err := g.engine.DeleteSectionPath(ctx, req.Section, req.Path); err != nil {
			return err
		}
	default:
		return fmt.Errorf("governance: unknown action %q", req.Action)
	}

	if g.audit != nil && requiresAudit(req.Section, req.Actor) {
		_ = g.audit.Append(ctx, AuditEntry{
			Section:     req.Section,
			Action:      req.Action,
			Actor:       string(req.Actor),
			Reason:      req.Reason,
			RequestID:   req.RequestID,
			RollbackRef: rollbackRef,
			Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
	return nil
}

func (g *GovernedService) RecentAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	if g.audit == nil {
		return nil, nil
	}
	return g.audit.Recent(ctx, limit)
}

func authorize(auth WriteAuthority, actor ActorRole) error {
	var allowed bool
	switch auth {
	case AuthorityUserOnly:
		allowed = actor == ActorUser
	case AuthorityAgentSelf:
		allowed = actor == ActorUser || actor == ActorAgent || actor == ActorSystem
	case AuthorityReport:
		allowed = actor == ActorUser || actor == ActorReportSystem
	case AuthorityTBD:
		allowed = actor == ActorUser
	default:
		allowed = false
	}
	if !allowed {
		return fmt.Errorf("%w: actor %q cannot mutate section with authority %q", ErrUnauthorized, actor, auth)
	}
	return nil
}

func requiresAudit(section SectionName, actor ActorRole) bool {
	if section == SectionMemoryMD && (actor == ActorSystem || actor == ActorAgent) {
		return false
	}
	switch section {
	case SectionIdentity, SectionRelationship, SectionCommitment, SectionPreferences:
		return true
	case SectionJournalReflective, SectionProposalInbox, SectionReportIndexes, SectionAAAKSummaries:
		return true
	}
	info := SectionRegistry[section]
	return info.WriteAuth == AuthorityUserOnly
}

func stateHash(data map[string]any) string {
	raw, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(raw)
	return fmt.Sprintf("%x", h[:8])
}
