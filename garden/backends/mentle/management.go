package mentle

import (
	"context"
	"encoding/json"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

func (a *Adapter) managementScopes(scopes []evolution.Scope) ([]string, error) {
	if len(scopes) == 0 {
		return nil, &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "exact scopes required"}
	}
	encoded := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		if !a.admit(scope) {
			return nil, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "scope not admitted"}
		}
		encoded = append(encoded, memory.EncodeScope(scope))
	}
	return encoded, nil
}
func (a *Adapter) List(ctx context.Context, req memory.ListRequest) (memory.RecordPage, error) {
	scopes, err := a.managementScopes(req.Scopes)
	if err != nil {
		return memory.RecordPage{}, err
	}
	if req.Limit < 0 || req.Limit > 100 {
		return memory.RecordPage{}, &evolution.ContractError{Code: evolution.ErrInvalidSchema, Message: "invalid list limit"}
	}
	if req.Limit == 0 {
		req.Limit = 20
	}
	page, err := a.svc.ListMemories(ctx, facade.ListMemoryOptions{Scopes: scopes, Collection: req.Collection, Limit: req.Limit, Cursor: req.Cursor})
	if err != nil {
		return memory.RecordPage{}, mapFacadeError(err)
	}
	out := memory.RecordPage{Items: []memory.Record{}}
	for _, record := range page.Items {
		mapped, err := managementRecord(record)
		if err != nil {
			return memory.RecordPage{}, err
		}
		out.Items = append(out.Items, mapped)
	}
	if page.NextCursor != nil {
		out.NextCursor = *page.NextCursor
	}
	return out, nil
}
func (a *Adapter) Get(ctx context.Context, id string) (memory.Record, error) {
	record, err := a.svc.GetMemory(ctx, id)
	if err != nil {
		return memory.Record{}, mapFacadeError(err)
	}
	if !memory.ScopeVisible(record.Scope, a.admitted) {
		return memory.Record{}, &evolution.ContractError{Code: evolution.ErrAuthorityDenied, Message: "record not admitted"}
	}
	return managementRecord(record)
}
func (a *Adapter) Collections(ctx context.Context, scopes []evolution.Scope) ([]memory.Collection, error) {
	encoded, err := a.managementScopes(scopes)
	if err != nil {
		return nil, err
	}
	rows, err := a.svc.ScopedCollections(ctx, encoded)
	if err != nil {
		return nil, mapFacadeError(err)
	}
	out := []memory.Collection{}
	for _, r := range rows {
		out = append(out, memory.Collection{ID: r.Name, Count: r.Count})
	}
	return out, nil
}
func managementRecord(r facade.Memory) (memory.Record, error) {
	scope, err := memory.DecodeScope(r.Scope)
	if err != nil {
		return memory.Record{}, err
	}
	sources := []evolution.SourceRef{}
	if raw := r.Metadata["evolution_sources"]; raw != nil {
		encoded, _ := json.Marshal(raw)
		if err = json.Unmarshal(encoded, &sources); err != nil {
			return memory.Record{}, &evolution.ContractError{Code: evolution.ErrRecoveryRequired, Message: "invalid stored lineage"}
		}
	}
	return memory.Record{ID: r.ID, Scope: scope, Revision: uint64(r.Version), Body: r.Content, Status: r.Status, Kind: r.Kind, Tags: r.Tags, Sources: sources, OccurredAt: r.ValidFrom, CapturedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}
