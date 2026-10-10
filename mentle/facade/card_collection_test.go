package facade

import (
	"context"
	"testing"
)

func TestCardCollectionMatchesCanonicalDefault(t *testing.T) {
	ctx := context.Background()
	svc, err := OpenCatalogService(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "canonicalcollectionprobe ordinary note", Kind: "note"}, "collection-create", "collection-digest")
	if err != nil {
		t.Fatal(err)
	}
	record, err := svc.GetMemory(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	page, err := svc.SearchCards(ctx, CardQuery{Text: "canonicalcollectionprobe", Limit: 4})
	if err != nil || len(page.Cards) != 1 {
		t.Fatalf("cards=%+v err=%v", page, err)
	}
	if page.Cards[0].Collection != record.Collection {
		t.Fatalf("card collection %q differs from canonical %q", page.Cards[0].Collection, record.Collection)
	}
	filtered, err := svc.SearchCards(ctx, CardQuery{Text: "canonicalcollectionprobe", Collection: record.Collection, Limit: 4})
	if err != nil || len(filtered.Cards) != 1 || filtered.Cards[0].ID != record.ID {
		t.Fatalf("canonical collection filter lost record: %+v %v", filtered, err)
	}
}
