package evolution

import "time"

// FrozenCoreV2SchemaVersion is the v2 wire marker. v1 six-slot data must
// never carry it.
const FrozenCoreV2SchemaVersion = "laputa.frozen-core/v2"

// FrozenCoreKind is the closed named-kind roster for v2 sections. These are
// wire names, independent of the persona storage intenum.
type FrozenCoreKind string

const (
	FrozenKindMission      FrozenCoreKind = "mission"
	FrozenKindIdentity     FrozenCoreKind = "identity"
	FrozenKindRelationship FrozenCoreKind = "relationship"
	FrozenKindRedline      FrozenCoreKind = "redline"
	FrozenKindUser         FrozenCoreKind = "user"
	FrozenKindDream        FrozenCoreKind = "dream"
	FrozenKindDark         FrozenCoreKind = "dark"
)

// FrozenCoreV2Kinds is the exact ordered slot roster. WORLD and ACTMEM are
// never frozen slots.
var FrozenCoreV2Kinds = []FrozenCoreKind{
	FrozenKindMission,
	FrozenKindIdentity,
	FrozenKindRelationship,
	FrozenKindRedline,
	FrozenKindUser,
	FrozenKindDream,
	FrozenKindDark,
}

// MissionStatus is the closed mission envelope vocabulary.
type MissionStatus string

const (
	MissionUnassigned MissionStatus = "unassigned"
	MissionAssigned   MissionStatus = "assigned"
)

// FrozenSectionV2 is one named, bounded authority projection.
type FrozenSectionV2 struct {
	Kind           FrozenCoreKind `json:"kind"`
	Content        string         `json:"content"`
	SourceRevision uint64         `json:"source_revision"`
	SourceHash     string         `json:"source_hash"`
}

// FrozenCoreV2 is the session-frozen projection envelope. The mission slot
// is present in every capture; when mission_status is unassigned its content
// is empty and source_revision is zero.
type FrozenCoreV2 struct {
	SchemaVersion string            `json:"schema_version"`
	SessionID     string            `json:"session_id"`
	CapturedAt    time.Time         `json:"captured_at"`
	MissionStatus MissionStatus     `json:"mission_status"`
	Sections      []FrozenSectionV2 `json:"sections"`
}

// Section returns the named slot, or false when the kind is not part of the
// v2 roster.
func (c FrozenCoreV2) Section(kind FrozenCoreKind) (FrozenSectionV2, bool) {
	for _, section := range c.Sections {
		if section.Kind == kind {
			return section, true
		}
	}
	return FrozenSectionV2{}, false
}

// Content is the named slot's projected content, or "" when absent.
func (c FrozenCoreV2) Content(kind FrozenCoreKind) string {
	section, ok := c.Section(kind)
	if !ok {
		return ""
	}
	return section.Content
}

// MissionRevision is the pinned Mission source revision (0 when unassigned
// or the envelope carries no mission slot).
func (c FrozenCoreV2) MissionRevision() uint64 {
	if mission, ok := c.Section(FrozenKindMission); ok {
		return mission.SourceRevision
	}
	return 0
}

// DecodeFrozenCoreV2 strict-decodes and validates the v2 envelope: exact
// schema marker, exactly seven slots in the declared order, and a mission
// section consistent with mission_status.
func DecodeFrozenCoreV2(data []byte) (FrozenCoreV2, error) {
	var core FrozenCoreV2
	if err := DecodeStrictJSON(data, &core); err != nil {
		return FrozenCoreV2{}, err
	}
	if err := core.Validate(); err != nil {
		return FrozenCoreV2{}, err
	}
	return core, nil
}

// Validate enforces the v2 slot roster and mission envelope consistency.
func (c FrozenCoreV2) Validate() error {
	if c.SchemaVersion != FrozenCoreV2SchemaVersion {
		return invalidSchema("schema_version must be %q", FrozenCoreV2SchemaVersion)
	}
	switch c.MissionStatus {
	case MissionAssigned, MissionUnassigned:
	default:
		return invalidSchema("unknown mission_status %q", c.MissionStatus)
	}
	if len(c.Sections) != len(FrozenCoreV2Kinds) {
		return invalidSchema("v2 frozen core requires exactly %d named sections", len(FrozenCoreV2Kinds))
	}
	for i, want := range FrozenCoreV2Kinds {
		if c.Sections[i].Kind != want {
			return invalidSchema("v2 section %d must be %q, got %q", i, want, c.Sections[i].Kind)
		}
	}
	mission := c.Sections[0]
	if c.MissionStatus == MissionUnassigned && (mission.Content != "" || mission.SourceRevision != 0) {
		return invalidSchema("unassigned mission slot must be empty with source_revision 0")
	}
	if c.MissionStatus == MissionAssigned && mission.SourceRevision == 0 {
		return invalidSchema("assigned mission slot requires positive source_revision")
	}
	return nil
}
