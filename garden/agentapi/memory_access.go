package agentapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
)

type MemoryTarget struct {
	SharedSpaceID string `json:"shared_space_id,omitempty"`
}
type memoryAccess struct {
	s      *Service
	b      Binding
	p      Principal
	target MemoryTarget
}

func (a *memoryAccess) current(ctx context.Context, op Operation) (registry.ResolvedAccess, error) {
	if err := Authorize(a.s.runtime.ProfileID, a.b, a.p, op); err != nil {
		return registry.ResolvedAccess{}, err
	}
	return a.s.resolve(ctx, a.b, op, a.target.SharedSpaceID)
}
func (a *memoryAccess) backend(ctx context.Context, scope evolution.Scope) (memory.Backend, error) {
	return a.s.installation.owner.BackendFor(ctx, scope, BackendMentle)
}
func admittedScope(scopes []evolution.Scope, want evolution.Scope) bool {
	for _, scope := range scopes {
		if scope.Equal(want) {
			return true
		}
	}
	return false
}
func requestedScopes(want, allowed []evolution.Scope) ([]evolution.Scope, error) {
	if len(want) == 0 {
		return allowed, nil
	}
	for _, scope := range want {
		if !admittedScope(allowed, scope) {
			return nil, failure("principal_forbidden", "scope not admitted")
		}
	}
	return want, nil
}
func (a *memoryAccess) Capabilities() memory.Capabilities {
	// The selected provider reports its own model/index capability; controls
	// and membership remain live checks, never capability grants.
	b, err := a.backend(context.Background(), evolution.Scope{SubjectID: a.b.ProfileID, Kind: evolution.ScopePersonal})
	if err != nil {
		return memory.Capabilities{}
	}
	return b.Capabilities()
}
func (a *memoryAccess) Health(ctx context.Context) (memory.Health, error) {
	access, err := a.current(ctx, OpIndexHealth)
	if err != nil {
		return memory.Health{}, err
	}
	b, err := a.backend(ctx, access.WriteScope)
	if err != nil {
		return memory.Health{}, err
	}
	return b.Health(ctx)
}
func (a *memoryAccess) Close() error { return nil }

type scopePosition struct {
	Cursor string
	Offset int
}
type unionCursor struct {
	Batch     int
	Positions []scopePosition
}
type pageSlot[T any] struct {
	Value T
	Scope int
}

func (a *memoryAccess) openCursor(raw string, aad []byte) (unionCursor, error) {
	if raw == "" {
		return unionCursor{}, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return unionCursor{}, failure("invalid_request", "invalid continuation")
	}
	block, _ := aes.NewCipher(a.s.installation.cursorKey[:])
	gcm, _ := cipher.NewGCM(block)
	if len(data) < gcm.NonceSize() {
		return unionCursor{}, failure("invalid_request", "invalid continuation")
	}
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], aad)
	if err != nil {
		return unionCursor{}, failure("invalid_request", "continuation no longer admitted")
	}
	var cur unionCursor
	if err = json.Unmarshal(plain, &cur); err != nil {
		return cur, failure("invalid_request", "invalid continuation")
	}
	return cur, nil
}
func (a *memoryAccess) sealCursor(cur unionCursor, aad []byte) (string, error) {
	block, _ := aes.NewCipher(a.s.installation.cursorKey[:])
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	raw, err := json.Marshal(cur)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(gcm.Seal(nonce, nonce, raw, aad)), nil
}

// Each request fetches at most sixteen scope views and one bounded provider
// page per view. Continuation stores only offsets/provider cursors, no content.
func unionPage[T any](ctx context.Context, a *memoryAccess, access registry.ResolvedAccess, scopes []evolution.Scope, family, query, collection, cursor string, limit int, fetch func(evolution.Scope, string) ([]T, string, error), less func(T, T) bool) ([]T, string, bool, error) {
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return nil, "", false, failure("invalid_request", "limit must be 1..100")
	}
	aad, _ := json.Marshal(struct {
		Binding, Membership, Family, Query, Collection string
		Scopes                                         []evolution.Scope
		Limit                                          int
	}{a.s.bindingID, access.MembershipKey, family, query, collection, scopes, limit})
	cur, err := a.openCursor(cursor, aad)
	if err != nil {
		return nil, "", false, err
	}
	batches := (len(scopes) + 15) / 16
	if cur.Batch < 0 || cur.Batch >= batches {
		return nil, "", false, failure("invalid_request", "invalid continuation batch")
	}
	start := cur.Batch * 16
	end := min(start+16, len(scopes))
	if len(cur.Positions) == 0 {
		cur.Positions = make([]scopePosition, end-start)
	}
	if len(cur.Positions) != end-start {
		return nil, "", false, failure("invalid_request", "invalid continuation scopes")
	}
	slots := []pageSlot[T]{}
	pages := make([][]T, end-start)
	nexts := make([]string, end-start)
	for j, scope := range scopes[start:end] {
		values, next, err := fetch(scope, cur.Positions[j].Cursor)
		if err != nil {
			return nil, "", false, err
		}
		sort.SliceStable(values, func(x, y int) bool { return less(values[x], values[y]) })
		pages[j] = values
		nexts[j] = next
		offset := cur.Positions[j].Offset
		if offset < 0 || offset > len(values) {
			return nil, "", false, failure("invalid_request", "continuation position changed")
		}
		for _, v := range values[offset:] {
			slots = append(slots, pageSlot[T]{v, j})
		}
	}
	sort.SliceStable(slots, func(i, j int) bool {
		if less(slots[i].Value, slots[j].Value) {
			return true
		}
		if less(slots[j].Value, slots[i].Value) {
			return false
		}
		return slots[i].Scope < slots[j].Scope
	})
	out := []T{}
	for _, slot := range slots[:min(limit, len(slots))] {
		out = append(out, slot.Value)
		cur.Positions[slot.Scope].Offset++
	}
	remaining := false
	for j, values := range pages {
		if cur.Positions[j].Offset < len(values) {
			remaining = true
		} else if nexts[j] != "" {
			cur.Positions[j] = scopePosition{Cursor: nexts[j]}
			remaining = true
		}
	}
	if !remaining && cur.Batch+1 < batches {
		cur.Batch++
		cur.Positions = nil
		remaining = true
	}
	if remaining {
		next, err := a.sealCursor(cur, aad)
		return out, next, true, err
	}
	return out, "", false, nil
}

func (a *memoryAccess) Search(ctx context.Context, q memory.AuthorizedSearch) (memory.CardPage, error) {
	access, err := a.current(ctx, OpSearch)
	if err != nil {
		return memory.CardPage{}, err
	}
	scopes, err := requestedScopes(q.Scopes, access.ReadScopes)
	if err != nil {
		return memory.CardPage{}, err
	}
	items, next, truncated, err := unionPage(ctx, a, access, scopes, "search", q.Query, q.Collection, q.Cursor, q.Limit,
		func(scope evolution.Scope, cursor string) ([]memory.MemoryCard, string, error) {
			b, err := a.backend(ctx, scope)
			if err != nil {
				return nil, "", err
			}
			p, err := b.Search(ctx, memory.AuthorizedSearch{Scopes: []evolution.Scope{scope}, Query: q.Query, Collection: q.Collection, Cursor: cursor, Limit: 100, BudgetChars: q.BudgetChars})
			return p.Items, p.NextCursor, err
		},
		func(x, y memory.MemoryCard) bool {
			if x.CandidateScore != y.CandidateScore {
				return x.CandidateScore > y.CandidateScore
			}
			if x.Scope != y.Scope {
				return x.Scope < y.Scope
			}
			return x.ID < y.ID
		})
	if err != nil {
		return memory.CardPage{}, err
	}
	for j := range items {
		record, err := a.rawRecord(ctx, access, items[j].ID)
		if err != nil {
			return memory.CardPage{}, err
		}
		if len(a.visibleSources(ctx, access, record.Sources)) != len(record.Sources) {
			items[j].SourceRef = "unavailable"
		}
	}
	return memory.CardPage{Items: items, NextCursor: next, Truncated: truncated}, nil
}

func (a *memoryAccess) rawRecord(ctx context.Context, access registry.ResolvedAccess, id string) (memory.Record, error) {
	for _, scope := range access.ReadScopes {
		b, err := a.backend(ctx, scope)
		if err != nil {
			return memory.Record{}, err
		}
		m, ok := b.(memory.ManagementBackend)
		if !ok {
			return memory.Record{}, failure("unavailable", "management reads unavailable")
		}
		record, err := m.Get(ctx, id)
		if err != nil {
			if evolution.CodeOf(err) == evolution.ErrAuthorityDenied || evolution.CodeOf(err) == evolution.ErrEffectNotFound {
				continue
			}
			return memory.Record{}, materialError(err)
		}
		if !admittedScope(access.ReadScopes, record.Scope) {
			continue
		}
		return record, nil
	}
	return memory.Record{}, failure("not_found", "memory not found")
}
func (a *memoryAccess) visibleSources(ctx context.Context, access registry.ResolvedAccess, sources []evolution.SourceRef) []evolution.SourceRef {
	out := []evolution.SourceRef{}
	for _, source := range sources {
		if !admittedScope(access.ReadScopes, source.Scope) {
			continue
		}
		if source.SourceID == BackendMentle {
			r, err := a.rawRecord(ctx, access, source.RecordID)
			if err != nil || !r.Scope.Equal(source.Scope) || r.Revision != source.Revision || r.Status != "active" {
				continue
			}
		}
		out = append(out, source)
	}
	return out
}
func (a *memoryAccess) Expand(ctx context.Context, q memory.AuthorizedExpansion) (memory.EvidencePage, error) {
	access, err := a.current(ctx, OpExpand)
	if err != nil {
		return memory.EvidencePage{}, err
	}
	if access.Worker {
		allowed := false
		for _, id := range access.SelectedCards {
			allowed = allowed || id == q.CardID
		}
		if !allowed {
			return memory.EvidencePage{}, failure("principal_forbidden", "worker source not selected")
		}
	}
	if _, err = requestedScopes(q.Scopes, access.ReadScopes); err != nil {
		return memory.EvidencePage{}, err
	}
	record, err := a.rawRecord(ctx, access, q.CardID)
	if err != nil {
		return memory.EvidencePage{}, err
	}
	b, err := a.backend(ctx, record.Scope)
	if err != nil {
		return memory.EvidencePage{}, err
	}
	q.Scopes = []evolution.Scope{record.Scope}
	page, err := b.Expand(ctx, q)
	if err != nil {
		return page, materialError(err)
	}
	hidden := len(a.visibleSources(ctx, access, record.Sources)) != len(record.Sources)
	for j := range page.Items {
		f := &page.Items[j]
		if uri, err := url.Parse(f.SourceURI); f.SourceURI != "" {
			encoded := ""
			if err == nil {
				encoded = uri.Query().Get("scope")
				if id := uri.Query().Get("record_id"); id != "" {
					target, e := a.rawRecord(ctx, access, id)
					if e != nil || target.Status != "active" || memory.EncodeScope(target.Scope) != encoded {
						hidden = true
					}
				}
			}
			if encoded != "" && !memory.ScopeVisible(encoded, access.ReadScopes) {
				hidden = true
			}
			if strings.HasPrefix(record.Scope.SubjectID, "shared:") && encoded == "" {
				hidden = true
			}
		}
		refs := []string{}
		for _, id := range f.EvidenceRefs {
			r, e := a.rawRecord(ctx, access, id)
			if e == nil && r.Status == "active" {
				refs = append(refs, id)
			} else {
				hidden = true
			}
		}
		f.EvidenceRefs = refs
		if hidden {
			f.SourceURI = ""
			f.SourceRev = ""
			f.Validity = "source_unavailable"
		}
	}
	return page, nil
}
func (a *memoryAccess) Mutate(ctx context.Context, m memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	access, err := a.current(ctx, OpRemember)
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	if m.Scope.SubjectID != "" && !m.Scope.Equal(access.WriteScope) {
		return memory.MutationReceipt{}, failure("invalid_binding", "write scope does not match selected target")
	}
	m.Scope = access.WriteScope
	m.DestinationID = BackendMentle
	for _, source := range m.Sources {
		if !admittedScope(access.ReadScopes, source.Scope) {
			return memory.MutationReceipt{}, failure("principal_forbidden", "source scope not admitted")
		}
		if source.SourceID == BackendMentle {
			record, err := a.rawRecord(ctx, access, source.RecordID)
			if err != nil || !record.Scope.Equal(source.Scope) || record.Revision != source.Revision {
				return memory.MutationReceipt{}, failure("conflict", "source changed or unavailable")
			}
		}
	}
	b, err := a.backend(ctx, access.WriteScope)
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	return b.Mutate(ctx, m)
}
func (a *memoryAccess) MutationStatus(ctx context.Context, id string) (memory.MutationReceipt, error) {
	access, err := a.current(ctx, OpRemember)
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	b, err := a.backend(ctx, access.WriteScope)
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	return b.MutationStatus(ctx, id)
}

func (a *memoryAccess) List(ctx context.Context, q memory.ListRequest) (memory.RecordPage, error) {
	access, err := a.current(ctx, OpSearch)
	if err != nil {
		return memory.RecordPage{}, err
	}
	scopes, err := requestedScopes(q.Scopes, access.ReadScopes)
	if err != nil {
		return memory.RecordPage{}, err
	}
	items, next, truncated, err := unionPage(ctx, a, access, scopes, "list", "", q.Collection, q.Cursor, q.Limit,
		func(scope evolution.Scope, cursor string) ([]memory.Record, string, error) {
			b, err := a.backend(ctx, scope)
			if err != nil {
				return nil, "", err
			}
			m, ok := b.(memory.ManagementBackend)
			if !ok {
				return nil, "", failure("unavailable", "management reads unavailable")
			}
			p, err := m.List(ctx, memory.ListRequest{Scopes: []evolution.Scope{scope}, Collection: q.Collection, Cursor: cursor, Limit: 100})
			return p.Items, p.NextCursor, err
		},
		func(x, y memory.Record) bool {
			if !x.UpdatedAt.Equal(y.UpdatedAt) {
				return x.UpdatedAt.After(y.UpdatedAt)
			}
			sx, sy := memory.EncodeScope(x.Scope), memory.EncodeScope(y.Scope)
			if sx != sy {
				return sx < sy
			}
			return x.ID < y.ID
		})
	if err != nil {
		return memory.RecordPage{}, err
	}
	for j := range items {
		items[j].Sources = a.visibleSources(ctx, access, items[j].Sources)
		if a.p != PrincipalUser {
			items[j].Body = ""
		}
	}
	return memory.RecordPage{Items: items, NextCursor: next, Truncated: truncated}, nil
}
func (a *memoryAccess) Get(ctx context.Context, id string) (memory.Record, error) {
	access, err := a.current(ctx, OpSearch)
	if err != nil {
		return memory.Record{}, err
	}
	r, err := a.rawRecord(ctx, access, id)
	if err != nil {
		return r, err
	}
	r.Sources = a.visibleSources(ctx, access, r.Sources)
	if a.p != PrincipalUser {
		r.Body = ""
	}
	return r, nil
}
func (a *memoryAccess) Collections(ctx context.Context, scopes []evolution.Scope) ([]memory.Collection, error) {
	access, err := a.current(ctx, OpSearch)
	if err != nil {
		return nil, err
	}
	scopes, err = requestedScopes(scopes, access.ReadScopes)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, scope := range scopes {
		b, err := a.backend(ctx, scope)
		if err != nil {
			return nil, err
		}
		m, ok := b.(memory.ManagementBackend)
		if !ok {
			return nil, failure("unavailable", "management reads unavailable")
		}
		cs, err := m.Collections(ctx, []evolution.Scope{scope})
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			counts[c.ID] += c.Count
		}
	}
	out := []memory.Collection{}
	for id, count := range counts {
		out = append(out, memory.Collection{ID: id, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Service) accessBackend(binding Binding, p Principal, target MemoryTarget) *memoryAccess {
	return &memoryAccess{s: s, b: binding, p: p, target: target}
}
func (s *Service) mutateMemory(ctx context.Context, p Principal, binding Binding, target MemoryTarget, m memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	if err := s.check(binding, p, OpRemember); err != nil {
		return memory.MutationReceipt{}, err
	}
	if len(m.Body) > maxInputBytes || len(m.OperationID) > 256 {
		return memory.MutationReceipt{}, failure("invalid_request", "mutation bounds exceeded")
	}
	prefix := s.bindingID + ":"
	if s.bindingID == "" {
		prefix = bindingKey(binding) + ":"
	}
	if m.OperationID == "" {
		m.OperationID = randomHex16()
	}
	if !strings.HasPrefix(m.OperationID, prefix) {
		m.OperationID = prefix + m.OperationID
	}
	if m.RecordID == "" && m.Operation == evolution.MutationCreate {
		digest := sha256.Sum256([]byte(m.OperationID))
		m.RecordID = fmt.Sprintf("mem_%x", digest[:16])
	}
	m.DestinationID = BackendMentle
	if s.installation == nil {
		return memory.MutationReceipt{}, failure("unavailable", "installation mutation boundary required")
	}
	access, err := s.resolve(ctx, binding, OpRemember, target.SharedSpaceID)
	if err != nil {
		return memory.MutationReceipt{}, err
	}
	if m.Scope.SubjectID != "" && !m.Scope.Equal(access.WriteScope) {
		return memory.MutationReceipt{}, failure("invalid_binding", "write scope does not match selected target")
	}
	m.Scope = access.WriteScope
	if m.Operation == evolution.MutationUpdate && m.Tags != nil && m.Body == "" {
		r, err := s.accessBackend(binding, p, target).rawRecord(ctx, access, m.RecordID)
		if err != nil {
			return memory.MutationReceipt{}, err
		}
		m.Body = r.Body
	}
	m.Origin = memory.MutationOrigin{ProfileID: binding.ProfileID, AgentID: binding.AgentID, Platform: binding.Platform, SessionID: binding.SessionID}
	m.PayloadDigest = mutationDigest(m)
	if err = m.Validate(); err != nil {
		return memory.MutationReceipt{}, failure("invalid_request", err.Error())
	}
	receipt, err := s.accessBackend(binding, p, target).Mutate(ctx, m)
	return receipt, materialErrorOrNil(err)
}
func materialErrorOrNil(err error) error {
	if err == nil {
		return nil
	}
	return materialError(err)
}

func (b *BoundClient) MutateMemory(ctx context.Context, target MemoryTarget, m memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	s, u, e := b.service()
	if e != nil {
		return memory.MutationReceipt{}, e
	}
	defer u()
	return s.mutateMemory(ctx, b.caller(), b.binding, target, m)
}
func (b *BoundClient) MemoryReceipt(ctx context.Context, target MemoryTarget, id string) (memory.MutationReceipt, error) {
	s, u, e := b.service()
	if e != nil {
		return memory.MutationReceipt{}, e
	}
	defer u()
	if e = s.check(b.binding, b.caller(), OpRemember); e != nil {
		return memory.MutationReceipt{}, e
	}
	r, e := s.accessBackend(b.binding, b.caller(), target).MutationStatus(ctx, id)
	return r, materialErrorOrNil(e)
}
func (b *BoundClient) ListMemory(ctx context.Context, q memory.ListRequest) (memory.RecordPage, error) {
	s, u, e := b.service()
	if e != nil {
		return memory.RecordPage{}, e
	}
	defer u()
	return s.accessBackend(b.binding, b.caller(), MemoryTarget{}).List(ctx, q)
}
func (b *BoundClient) GetMemory(ctx context.Context, id string) (memory.Record, error) {
	s, u, e := b.service()
	if e != nil {
		return memory.Record{}, e
	}
	defer u()
	return s.accessBackend(b.binding, b.caller(), MemoryTarget{}).Get(ctx, id)
}
func (b *BoundClient) MemoryCollections(ctx context.Context) ([]memory.Collection, error) {
	s, u, e := b.service()
	if e != nil {
		return nil, e
	}
	defer u()
	return s.accessBackend(b.binding, b.caller(), MemoryTarget{}).Collections(ctx, nil)
}
