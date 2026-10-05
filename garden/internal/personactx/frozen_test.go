package personactx

import (
	"context"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/laputa/persona"
)

type fakeReader struct {
	documents map[persona.Kind]*persona.Document
}

func (r fakeReader) GetDocument(kind persona.Kind) (*persona.Document, error) {
	if document, ok := r.documents[kind]; ok {
		return document, nil
	}
	return &persona.Document{}, nil
}

func TestCaptureHasExactlySevenBoundedSections(t *testing.T) {
	reader := fakeReader{documents: map[persona.Kind]*persona.Document{
		persona.KindIdentity:     {Content: "identity " + repeat("i", 300), Revision: 2, ContentHash: "id"},
		persona.KindRelationship: {Content: "relationship " + repeat("r", 200), Revision: 3, ContentHash: "rel"},
		persona.KindRedline:      {Content: "redline", Revision: 4, ContentHash: "red"},
		persona.KindUser:         {Content: "## Preferences\nuser preferences\n## Observations\nprivate observations", Revision: 5, ContentHash: "user"},
		persona.KindDream:        {Content: "dream " + repeat("d", 50), Revision: 6, ContentHash: "dream"},
		persona.KindDark:         {Content: "dark", Revision: 7, ContentHash: "dark"},
	}}
	core, err := Capture(reader, "session-1", func() time.Time { return time.Date(2026, 9, 3, 1, 2, 3, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	if len(core.Sections) != 7 {
		t.Fatalf("sections = %d", len(core.Sections))
	}
	if core.Sections[0].Kind != SectionMission || core.MissionStatus != evolution.MissionUnassigned {
		t.Fatalf("mission slot = %+v status=%q", core.Sections[0], core.MissionStatus)
	}
	if core.Content(SectionUser) != "user preferences" {
		t.Fatalf("user projection = %q", core.Content(SectionUser))
	}
	if core.Sections[1].SourceRevision != 2 {
		t.Fatalf("source revision = %#v", core.Sections[1])
	}
	if persona.VisibleLen(core.Content(SectionIdentity)) > 200 {
		t.Fatalf("identity not bounded")
	}
	if Render(core, 4000) == "" {
		t.Fatal("render is empty")
	}
}

func TestStoreDoesNotDriftAcrossRestart(t *testing.T) {
	temp := t.TempDir()
	path := temp + "\\garden.db"
	reader := fakeReader{documents: map[persona.Kind]*persona.Document{
		persona.KindIdentity:     {Content: "first", Revision: 1},
		persona.KindRelationship: &persona.Document{}, persona.KindRedline: &persona.Document{}, persona.KindUser: &persona.Document{}, persona.KindDream: &persona.Document{}, persona.KindDark: &persona.Document{},
	}}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Capture(context.Background(), "session-1", reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reader.documents[persona.KindIdentity] = &persona.Document{Content: "second", Revision: 2}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	second, err := store.Capture(context.Background(), "session-1", reader)
	if err != nil {
		t.Fatal(err)
	}
	if second.Content(SectionIdentity) != first.Content(SectionIdentity) || second.Sections[1].SourceRevision != 1 {
		t.Fatalf("frozen session drifted: first=%#v second=%#v", first, second)
	}
}

func repeat(value string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += value
	}
	return result
}
