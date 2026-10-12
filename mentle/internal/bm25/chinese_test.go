package bm25

import "testing"

func TestChineseQueriesMatchSharedDecision(t *testing.T) {
	idx := New(DefaultK1, DefaultB)
	idx.Add("decision", "项目上线日期选择周五，部署方案已经确认")
	for _, query := range []string{"上线日期", "周五上线", "项目上线日期选择周五"} {
		got := idx.Search(query, 10)
		if len(got) != 1 || got[0].ID != "decision" {
			t.Fatalf("Chinese query %q: %+v", query, got)
		}
	}
	idx.Remove("decision")
	if got := idx.Search("上线日期", 10); len(got) != 0 {
		t.Fatal("removed Chinese record persisted", got)
	}
}
