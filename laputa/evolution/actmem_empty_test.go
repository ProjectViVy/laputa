package evolution

import "testing"

func TestEmptyActmemDocumentRoundTrips(t *testing.T) {
	document := ActmemDocument{Schema: ActmemSchemaV2, Revision: 2, Updated: "2026-10-10T00:00:00Z", Sections: map[EntrySection][]ActmemEntry{SectionPulse: {}, SectionRecap: {}, SectionWork: {}}}
	parsed, err := ParseActmemDocument(document.Render())
	if err != nil || parsed.Revision != 2 {
		t.Fatalf("empty native head cannot reopen: %+v %v", parsed, err)
	}
}
