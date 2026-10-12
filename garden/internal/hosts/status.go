package hosts

import (
	"context"
	"encoding/json"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"os"
	"reflect"
	"sort"
)

type HostStatus struct {
	BindingID        string            `json:"binding_id"`
	SavedRevisions   map[string]uint64 `json:"saved_revisions"`
	WrittenRevisions map[string]uint64 `json:"written_revisions,omitempty"`
	LoadedRevisions  map[string]uint64 `json:"loaded_revisions,omitempty"`
	WrittenHash      string            `json:"written_hash,omitempty"`
	LoadedHash       string            `json:"loaded_hash,omitempty"`
	SessionID        string            `json:"session_id,omitempty"`
	AdapterVersion   string            `json:"adapter_version,omitempty"`
	State            string            `json:"state"`
}
type projectionState struct {
	HostStatus
	Path             string            `json:"path,omitempty"`
	BackupPath       string            `json:"backup_path,omitempty"`
	LastBackupPath   string            `json:"last_backup_path,omitempty"`
	OriginalExists   bool              `json:"original_exists"`
	PlannedHash      string            `json:"planned_hash,omitempty"`
	PlannedRevisions map[string]uint64 `json:"planned_revisions,omitempty"`
}

func (s *Service) state(ctx context.Context, id string) (projectionState, error) {
	b, err := s.registry.Binding(ctx, id)
	if err != nil {
		return projectionState{}, err
	}
	v := projectionState{HostStatus: HostStatus{BindingID: id, State: "inactive"}}
	if len(b.Projection) > 0 {
		err = json.Unmarshal(b.Projection, &v)
	}
	return v, err
}
func (s *Service) save(ctx context.Context, v projectionState) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.registry.SetProjection(ctx, v.BindingID, raw)
}
func selectedKinds(revisions map[string]uint64) ([]persona.Kind, error) {
	names := []string{}
	for name := range revisions {
		names = append(names, name)
	}
	sort.Strings(names)
	kinds := []persona.Kind{}
	for _, name := range names {
		kind, err := persona.ParseKind(name)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}
func (s *Service) recoverFile(v *projectionState, file string) error {
	a, z, err := managedRegion(file, v.BindingID)
	if err != nil {
		return err
	}
	if a >= 0 && v.PlannedHash != "" && hash(file[a:z]) == v.PlannedHash {
		v.WrittenHash = v.PlannedHash
		v.WrittenRevisions = v.PlannedRevisions
		v.PlannedHash = ""
		v.PlannedRevisions = nil
		v.State = "pending_reload"
	}
	if v.WrittenHash != "" && (a < 0 || hash(file[a:z]) != v.WrittenHash) {
		return failure("conflict", "managed block changed; manual recovery required")
	}
	return nil
}
func (s *Service) Apply(ctx context.Context, p FilePlan) (HostStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validateTarget(ctx, p.Path); err != nil {
		return HostStatus{}, err
	}
	if _, err := s.binding(ctx, p.BindingID, true); err != nil {
		return HostStatus{}, err
	}
	v, err := s.state(ctx, p.BindingID)
	if err != nil {
		return v.HostStatus, err
	}
	file, exists, err := readFile(p.Path)
	if err != nil {
		return v.HostStatus, err
	}
	if v.Path != "" && v.Path != p.Path {
		return v.HostStatus, failure("conflict", "binding already manages another path")
	}
	if err = s.recoverFile(&v, file); err != nil {
		return v.HostStatus, err
	}
	if hash(file) != p.OriginalHash {
		return v.HostStatus, failure("conflict", "host file changed after preview")
	}
	kinds, err := selectedKinds(p.SourceRevisions)
	if err != nil || len(kinds) == 0 {
		return v.HostStatus, failure("invalid_request", "projection kinds missing")
	}
	body, revisions, err := s.projection(ctx, p.BindingID, kinds)
	if err != nil {
		return v.HostStatus, err
	}
	block := managedBlock(p.BindingID, body)
	a, z, err := managedRegion(file, p.BindingID)
	if err != nil {
		return v.HostStatus, err
	}
	proposed := file + block
	if a >= 0 {
		if v.WrittenHash == "" {
			return v.HostStatus, failure("conflict", "existing block not owned")
		}
		proposed = file[:a] + block + file[z:]
	}
	if p.AdapterVersion != "garden-projection/v1" || !reflect.DeepEqual(revisions, p.SourceRevisions) || hash(block) != p.ManagedHash || proposed != p.ProposedContent {
		return v.HostStatus, failure("conflict", "projection or saved source changed after preview")
	}
	if err = ctx.Err(); err != nil {
		return v.HostStatus, err
	}
	backupPath, err := backup(s.backupDir, file)
	if err != nil {
		return v.HostStatus, err
	}
	if v.BackupPath == "" {
		v.BackupPath = backupPath
		v.OriginalExists = exists
	}
	v.LastBackupPath = backupPath
	v.Path = p.Path
	v.SavedRevisions = revisions
	v.PlannedHash = p.ManagedHash
	v.PlannedRevisions = revisions
	v.AdapterVersion = p.AdapterVersion
	v.State = "unknown"
	if err = s.save(ctx, v); err != nil {
		return v.HostStatus, err
	}
	// Queued writes recheck control and the physical file immediately before
	// replacement, after backup/intent persistence. No snapshot is authority.
	if _, err = s.binding(ctx, p.BindingID, true); err != nil {
		return v.HostStatus, err
	}
	current, currentExists, err := readFile(p.Path)
	if err != nil {
		return v.HostStatus, err
	}
	if currentExists != exists || hash(current) != p.OriginalHash {
		return v.HostStatus, failure("conflict", "host file changed before write")
	}
	if err = replace(p.Path, p.ProposedContent); err != nil {
		return v.HostStatus, err
	}
	v.WrittenHash = p.ManagedHash
	v.WrittenRevisions = revisions
	v.PlannedHash = ""
	v.PlannedRevisions = nil
	v.LoadedHash = ""
	v.LoadedRevisions = nil
	v.SessionID = ""
	v.State = "pending_reload"
	if err = s.save(ctx, v); err != nil {
		v.State = "unknown"
		return v.HostStatus, err
	}
	return v.HostStatus, nil
}
func (s *Service) Status(ctx context.Context, id string) (HostStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return HostStatus{}, failure("unavailable", "host service closed")
	}
	b, err := s.registry.Binding(ctx, id)
	if err != nil {
		return HostStatus{}, err
	}
	v, err := s.state(ctx, id)
	if err != nil {
		return v.HostStatus, err
	}
	if b.Revoked {
		if v.State != "conflict" {
			v.State = "revoked"
		}
		return v.HostStatus, nil
	}
	kinds, err := selectedKinds(v.SavedRevisions)
	if err != nil {
		return v.HostStatus, err
	}
	_, saved, err := s.projection(ctx, id, kinds)
	if err != nil {
		return v.HostStatus, err
	}
	v.SavedRevisions = saved
	if v.Path != "" {
		file, _, err := readFile(v.Path)
		if err != nil {
			return v.HostStatus, err
		}
		if err = s.recoverFile(&v, file); err != nil {
			v.State = "conflict"
			v.LoadedHash = ""
			v.LoadedRevisions = nil
			_ = s.save(ctx, v)
			return v.HostStatus, err
		}
	}
	if !b.Controls.PersonaSync {
		v.State = "inactive"
	} else if v.WrittenHash != "" && !reflect.DeepEqual(v.SavedRevisions, v.WrittenRevisions) {
		v.State = "pending_reload"
	}
	if err = s.save(ctx, v); err != nil {
		return v.HostStatus, err
	}
	return v.HostStatus, nil
}
func (s *Service) AcknowledgeLoaded(ctx context.Context, id, sessionID, loadedHash string) (HostStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.binding(ctx, id, true)
	if err != nil {
		return HostStatus{}, err
	}
	v, err := s.state(ctx, id)
	if err != nil {
		return v.HostStatus, err
	}
	file, _, err := readFile(v.Path)
	if err != nil {
		return v.HostStatus, err
	}
	if err = s.recoverFile(&v, file); err != nil {
		return v.HostStatus, err
	}
	if sessionID == "" || loadedHash == "" || loadedHash != v.WrittenHash {
		return v.HostStatus, failure("conflict", "loaded session/hash does not match written projection")
	}
	if s.LoadedEvidence == nil {
		v.State = "unknown"
		_ = s.save(ctx, v)
		return v.HostStatus, failure("unavailable", "host has no qualified loaded evidence")
	}
	if err = s.LoadedEvidence(ctx, b, sessionID, loadedHash); err != nil {
		return v.HostStatus, err
	}
	v.LoadedHash = loadedHash
	v.LoadedRevisions = v.WrittenRevisions
	v.SessionID = sessionID
	v.State = "loaded"
	if err = s.save(ctx, v); err != nil {
		return v.HostStatus, err
	}
	return v.HostStatus, nil
}
func (s *Service) Revoke(ctx context.Context, id string) (HostStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return HostStatus{}, failure("unavailable", "host service closed")
	}
	b, err := s.registry.Binding(ctx, id)
	if err != nil {
		return HostStatus{}, err
	}
	if !b.Revoked {
		if err = s.registry.RevokeHostBinding(ctx, id); err != nil {
			return HostStatus{}, err
		}
	}
	v, err := s.state(ctx, id)
	if err != nil {
		return v.HostStatus, err
	}
	v.State = "revoked"
	v.LoadedHash = ""
	v.LoadedRevisions = nil
	v.SessionID = ""
	if v.Path == "" {
		return v.HostStatus, s.save(ctx, v)
	}
	file, exists, err := readFile(v.Path)
	if err != nil {
		return v.HostStatus, err
	}
	if !exists {
		return v.HostStatus, s.save(ctx, v)
	}
	a, z, err := managedRegion(file, id)
	if err != nil {
		v.State = "conflict"
		_ = s.save(ctx, v)
		return v.HostStatus, err
	}
	if a < 0 || (hash(file[a:z]) != v.WrittenHash && hash(file[a:z]) != v.PlannedHash) {
		v.State = "conflict"
		_ = s.save(ctx, v)
		return v.HostStatus, failure("conflict", "managed region edited; credential revoked, manual cleanup required")
	}
	outside := file[:a] + file[z:]
	if !v.OriginalExists && outside == "" {
		err = os.Remove(v.Path)
	} else {
		err = replace(v.Path, outside)
	}
	if err != nil {
		v.State = "conflict"
		_ = s.save(ctx, v)
		return v.HostStatus, err
	}
	v.WrittenHash = ""
	v.WrittenRevisions = nil
	v.PlannedHash = ""
	v.PlannedRevisions = nil
	v.Path = ""
	return v.HostStatus, s.save(ctx, v)
}
