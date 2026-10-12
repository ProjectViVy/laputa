package agentapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

type sharingFixture struct {
	i       *Installation
	a, b, c *BoundClient
	space   string
}

func sharingInstallation(t *testing.T) *sharingFixture {
	t.Helper()
	cfg := installationConfig(t)
	cfg.CatalogOnly = true
	return sharingInstallationConfig(t, cfg)
}
func sharingInstallationConfig(t *testing.T, cfg InstallationConfig) *sharingFixture {
	t.Helper()
	i, err := OpenInstallation(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Close() })
	for _, id := range []string{"A", "B", "C"} {
		installPersona(t, i, id)
	}
	space, err := i.CreateSharedSpace(context.Background(), "项目")
	if err != nil {
		t.Fatal(err)
	}
	if err = i.SetSharedMember(context.Background(), space.ID, "A", "read_write"); err != nil {
		t.Fatal(err)
	}
	if err = i.SetSharedMember(context.Background(), space.ID, "B", "read"); err != nil {
		t.Fatal(err)
	}
	a, _ := installedClient(t, i, "A").BindSession("chat")
	b, _ := installedClient(t, i, "B").BindSession("chat")
	c, _ := installedClient(t, i, "C").BindSession("chat")
	return &sharingFixture{i: i, a: a, b: b, c: c, space: space.ID}
}
func createBoundMemory(t *testing.T, b *BoundClient, space, id, body string) memory.MutationReceipt {
	t.Helper()
	receipt, err := b.MutateMemory(context.Background(), MemoryTarget{SharedSpaceID: space}, memory.AuthorizedMutation{OperationID: "create-" + id, Operation: evolution.MutationCreate, RecordID: id, Body: body, ExpectedAbsent: true, Inference: evolution.InferenceObserved})
	if err != nil || receipt.CanonicalStatus != memory.CanonicalCompleted {
		t.Fatal(receipt, err)
	}
	return receipt
}

func TestSharedMentleReadWrite(t *testing.T) {
	f := sharingInstallation(t)
	ctx := context.Background()
	privateA := createBoundMemory(t, f.a, "", "private-a", "needle A_PRIVATE")
	createBoundMemory(t, f.b, "", "private-b", "needle B_PRIVATE")
	r := createBoundMemory(t, f.a, f.space, "shared", "needle SHARED")
	page, err := f.b.SearchCards(ctx, CardSearch{Query: "needle", Limit: 20})
	if err != nil || len(page.Cards) != 2 {
		t.Fatal(page, err)
	}
	for _, card := range page.Cards {
		if card.ID == privateA.TargetRef {
			t.Fatal("foreign private card")
		}
	}
	if _, err = f.c.ReadEvidence(ctx, EvidenceRead{Items: []EvidenceRef{{CardID: r.TargetRef, ExpectedRevision: 1}}}); err == nil {
		t.Fatal("C expanded shared data")
	}
	_, err = f.b.MutateMemory(ctx, MemoryTarget{SharedSpaceID: f.space}, memory.AuthorizedMutation{OperationID: "readonly", Operation: evolution.MutationTombstone, RecordID: r.TargetRef, ExpectedRevision: 1, Inference: evolution.InferenceObserved})
	if err == nil {
		t.Fatal("read member deleted shared record")
	}
	got, err := f.a.MemoryReceipt(ctx, MemoryTarget{SharedSpaceID: f.space}, r.OperationID)
	if err != nil || got.Revision != r.Revision {
		t.Fatal("receipt", got, err)
	}
	createBoundMemory(t, f.a, "", "private-a", "needle A_PRIVATE") // identical operation replay, not another row.
	if _, err = f.a.MutateMemory(ctx, MemoryTarget{}, memory.AuthorizedMutation{OperationID: "create-private-a", Operation: evolution.MutationCreate, RecordID: "private-a", Body: "changed", ExpectedAbsent: true, Inference: evolution.InferenceObserved}); err == nil {
		t.Fatal("changed payload replay succeeded")
	}
}

func TestRemovedMemberOldEvidenceCursorContext(t *testing.T) {
	f := sharingInstallation(t)
	ctx := context.Background()
	createBoundMemory(t, f.b, "", "private-b", "needle B_PRIVATE")
	shared := createBoundMemory(t, f.a, f.space, "shared", "needle SHARED")
	page, err := f.b.SearchCards(ctx, CardSearch{Query: "needle", Limit: 1})
	if err != nil || page.NextCursor == nil {
		t.Fatal("no continuation", page, err)
	}
	before, err := f.b.FastRecall(ctx, FastRecallRequest{Query: "needle", BudgetChars: 6000})
	if err != nil || !strings.Contains(before.Context, "SHARED") {
		t.Fatal("shared not useful in real context", before, err)
	}
	if err = f.i.RemoveSharedMember(ctx, f.space, "B"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.b.ReadEvidence(ctx, EvidenceRead{Items: []EvidenceRef{{CardID: shared.TargetRef, ExpectedRevision: 1}}}); err == nil {
		t.Fatal("old card granted removed member")
	}
	if _, err = f.b.SearchCards(ctx, CardSearch{Query: "needle", Limit: 1, Cursor: *page.NextCursor}); err == nil {
		t.Fatal("stale cursor reused")
	}
	after, err := f.b.FastRecall(ctx, FastRecallRequest{Query: "needle", BudgetChars: 6000})
	if err != nil || strings.Contains(after.Context, "SHARED") {
		t.Fatal("stale shared context reinjected", after, err)
	}
}

func TestSharedLineageHidesPrivateSource(t *testing.T) {
	f := sharingInstallation(t)
	ctx := context.Background()
	root, err := f.i.owner.Subject(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	private := evolution.Scope{SubjectID: "A", Kind: evolution.ScopePersonal}
	privateReceipt := createBoundMemory(t, f.a, "", "lineage-private", "PRIVATE_SECRET")
	shared := evolution.Scope{SubjectID: "shared:" + f.space, Kind: evolution.ScopePersonal}
	source := "garden-source://evidence?record_id=" + privateReceipt.TargetRef + "&scope=" + url.QueryEscape(memory.EncodeScope(private))
	record, err := root.Mentle.CreateMemory(ctx, facade.CreateMemoryRequest{Content: "shared fact", Kind: "note", Scope: memory.EncodeScope(shared), Source: facade.MemorySource{Type: "agent", URI: source, Revision: "1"}, Metadata: map[string]any{"evolution_sources": []evolution.SourceRef{{SourceID: "mentle", RecordID: privateReceipt.TargetRef, Scope: shared, Revision: 1}}}}, "lineage", "lineage-digest")
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := f.b.ReadEvidence(ctx, EvidenceRead{Items: []EvidenceRef{{CardID: record.ID, ExpectedRevision: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(fragments)
	if strings.Contains(string(raw), privateReceipt.TargetRef) || strings.Contains(string(raw), "garden-source:") {
		t.Fatal("private source metadata disclosed", string(raw))
	}
	if len(fragments) == 0 || fragments[0].Excerpt != "shared fact" {
		t.Fatal("permitted fact disappeared", fragments)
	}
}

func TestChineseSharedRecall(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprintf("native-vector-%t", native), func(t *testing.T) {
			cfg := installationConfig(t)
			cfg.CatalogOnly = !native
			if native {
				cfg.ModelsDir, _ = filepath.Abs("../../mentle/models")
				cfg.RequireLocalModel = true
			}
			f := sharingInstallationConfig(t, cfg)
			ctx := context.Background()
			shared := createBoundMemory(t, f.a, f.space, "shared", "项目上线日期选择周五，部署方案已经确认")
			for i := 0; i < 110; i++ {
				createBoundMemory(t, f.c, "", fmt.Sprintf("c-%d", i), "上线日期 上线日期 周五上线")
			}
			for _, query := range []string{"上线日期", "周五上线", "项目上线日期选择周五"} {
				page, err := f.b.SearchCards(ctx, CardSearch{Query: query, Limit: 20})
				if err != nil || len(page.Cards) != 1 || page.Cards[0].ID != shared.TargetRef {
					t.Fatalf("%s: %+v %v", query, page, err)
				}
			}
			if native {
				b, err := f.i.owner.BackendFor(ctx, evolution.Scope{SubjectID: "B", Kind: evolution.ScopePersonal}, BackendMentle)
				if err != nil || !b.Capabilities().Vector {
					t.Fatal("native vector unavailable", err)
				}
				root, _ := f.i.owner.Subject(ctx, "B")
				vector, err := root.Mentle.Searcher.SearchWithFilter(ctx, "项目上线日期选择周五", map[string]any{"scope": memory.EncodeScope(evolution.Scope{SubjectID: "shared:" + f.space, Kind: evolution.ScopePersonal})}, 20)
				if err != nil || len(vector) == 0 {
					t.Fatal("scoped vector returned no shared candidate", vector, err)
				}
			}
		})
	}
}

func TestScopedManagementMetadata(t *testing.T) {
	f := sharingInstallation(t)
	ctx := context.Background()
	createBoundMemory(t, f.a, "", "a", "own")
	createBoundMemory(t, f.a, f.space, "s", "shared")
	root, _ := f.i.owner.Subject(ctx, "B")
	_, err := root.Mentle.CreateMemory(ctx, facade.CreateMemoryRequest{Content: "hidden", Kind: "note", Scope: memory.EncodeScope(evolution.Scope{SubjectID: "B", Kind: evolution.ScopePersonal}), Metadata: map[string]any{"collection": "B_SECRET_COLLECTION"}}, "hidden", "hidden-digest")
	if err != nil {
		t.Fatal(err)
	}
	records, err := f.a.ListMemory(ctx, memory.ListRequest{Limit: 20})
	if err != nil || len(records.Items) != 2 {
		t.Fatal(records, err)
	}
	if _, err = f.a.GetMemory(ctx, "nonexistent"); err == nil {
		t.Fatal("missing record returned")
	}
	collections, err := f.a.MemoryCollections(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(collections)
	if strings.Contains(string(raw), "B_SECRET_COLLECTION") {
		t.Fatal("foreign collection disclosed")
	}
	for _, item := range records.Items {
		if item.Scope.SubjectID == "B" {
			t.Fatal("foreign body in list")
		}
	}
}
