package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"

	"github.com/ProjectViVy/laputa/laputa/evolution"
)

type AccessRequest struct{ BindingID, SessionID, Operation, SharedSpaceID string }
type ResolvedAccess struct {
	ProfileID, BindingID, SessionKey, MembershipKey string
	ReadScopes                                      []evolution.Scope
	WriteScope                                      evolution.Scope
	Worker                                          bool
	SelectedCards                                   []string
}

func (s *Store) Resolve(ctx context.Context, req AccessRequest) (ResolvedAccess, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return ResolvedAccess{}, errors.New("registry closed")
	}
	b, err := s.bindingLocked(ctx, req.BindingID)
	if err != nil {
		return ResolvedAccess{}, err
	}
	if b.Revoked {
		return ResolvedAccess{}, denied("binding_revoked")
	}
	if !slices.Contains(b.Operations, req.Operation) {
		return ResolvedAccess{}, denied("principal_forbidden")
	}
	if b.Worker && (req.Operation != "expand" || len(b.SelectedCards) == 0 || req.SharedSpaceID != "") {
		return ResolvedAccess{}, denied("principal_forbidden")
	}
	switch req.Operation {
	case "bootstrap", "search", "expand", "index_health", "actmem_read", "actmem_query":
		if !b.Controls.Recall {
			return ResolvedAccess{}, denied("principal_forbidden")
		}
	case "capture", "remember", "actmem_write", "actmem_maintain":
		if !b.Controls.Capture {
			return ResolvedAccess{}, denied("principal_forbidden")
		}
	case "persona_get", "persona_propose", "persona_p16", "persona_review", "persona_repair":
		if !b.Controls.PersonaSync {
			return ResolvedAccess{}, denied("principal_forbidden")
		}
	}
	key, err := SessionKey(b.ProfileID, b.BindingID, req.SessionID)
	if err != nil {
		return ResolvedAccess{}, denied("invalid_binding")
	}
	personal := evolution.Scope{SubjectID: b.ProfileID, Kind: evolution.ScopePersonal}
	a := ResolvedAccess{ProfileID: b.ProfileID, BindingID: b.BindingID, SessionKey: key, ReadScopes: []evolution.Scope{personal}, WriteScope: personal, Worker: b.Worker, SelectedCards: b.SelectedCards}
	if b.WorkspaceID != "" {
		a.WriteScope = evolution.Scope{SubjectID: b.ProfileID, Kind: evolution.ScopeWorkspace, WorkspaceID: b.WorkspaceID}
		a.ReadScopes = append(a.ReadScopes, a.WriteScope)
	}
	rights := [][2]string{}
	if !b.Worker {
		rows, err := s.db.QueryContext(ctx, `SELECT space_id,access FROM shared_members WHERE profile_id=? ORDER BY space_id`, b.ProfileID)
		if err != nil {
			return ResolvedAccess{}, err
		}
		for rows.Next() {
			var id, access string
			if err = rows.Scan(&id, &access); err != nil {
				rows.Close()
				return ResolvedAccess{}, err
			}
			rights = append(rights, [2]string{id, access})
			a.ReadScopes = append(a.ReadScopes, evolution.Scope{SubjectID: "shared:" + id, Kind: evolution.ScopePersonal})
		}
		err = errors.Join(rows.Err(), rows.Close())
		if err != nil {
			return ResolvedAccess{}, err
		}
	}
	if req.SharedSpaceID != "" {
		access := ""
		for _, r := range rights {
			if r[0] == req.SharedSpaceID {
				access = r[1]
			}
		}
		if access == "" || (req.Operation == "remember" && access != "read_write") {
			return ResolvedAccess{}, denied("principal_forbidden")
		}
		if req.Operation == "capture" {
			return ResolvedAccess{}, denied("principal_forbidden")
		}
		a.WriteScope = evolution.Scope{SubjectID: "shared:" + req.SharedSpaceID, Kind: evolution.ScopePersonal}
	}
	raw, _ := json.Marshal(struct {
		Binding HostBinding
		Rights  [][2]string
	}{b, rights})
	digest := sha256.Sum256(raw)
	a.MembershipKey = hex.EncodeToString(digest[:])
	return a, nil
}
