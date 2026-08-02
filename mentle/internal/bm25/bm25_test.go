package bm25

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "hello world", []string{"hello", "world"}},
		{"uppercase", "Hello World", []string{"hello", "world"}},
		{"punctuation", "hello, world!", []string{"hello", "world"}},
		{"numbers", "test 123 abc", []string{"test", "123", "abc"}},
		{"hyphens", "state-of-the-art", []string{"state", "of", "the", "art"}},
		{"empty", "", nil},
		{"underscores", "foo_bar", []string{"foo_bar"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("tokenize(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIndex_AddAndSearch(t *testing.T) {
	idx := New(DefaultK1, DefaultB)

	idx.Add("doc1", "The quick brown fox jumps over the lazy dog")
	idx.Add("doc2", "The fox is quick and brown")
	idx.Add("doc3", "The lazy dog sleeps all day")

	// Search for "quick fox" should rank doc2 and doc1 highest.
	results := idx.Search("quick fox", 10)
	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	// doc2 has both "quick" and "fox" with shorter length -> higher score.
	if results[0].ID != "doc2" && results[0].ID != "doc1" {
		t.Errorf("expected doc1 or doc2 first, got %s", results[0].ID)
	}
}

func TestIndex_SearchLimit(t *testing.T) {
	idx := New(DefaultK1, DefaultB)

	for range 10 {
		idx.Add("doc", "repeated content for testing")
	}

	results := idx.Search("testing", 3)
	if len(results) > 3 {
		t.Errorf("expected at most 3 results, got %d", len(results))
	}
}

func TestIndex_Remove(t *testing.T) {
	idx := New(DefaultK1, DefaultB)

	idx.Add("doc1", "the quick brown fox")
	idx.Add("doc2", "the lazy dog")

	idx.Remove("doc1")

	results := idx.Search("quick", 10)
	if len(results) != 0 {
		t.Errorf("expected no results after removal, got %d", len(results))
	}

	// doc2 should still be searchable.
	results = idx.Search("lazy", 10)
	if len(results) != 1 || results[0].ID != "doc2" {
		t.Errorf("expected doc2, got %v", results)
	}
}

func TestIndex_Upsert(t *testing.T) {
	idx := New(DefaultK1, DefaultB)

	idx.Add("doc1", "original content about dogs")
	idx.Add("doc1", "updated content about cats") // should replace

	results := idx.Search("cats", 10)
	if len(results) != 1 || results[0].ID != "doc1" {
		t.Errorf("expected doc1 with 'cats', got %v", results)
	}

	results = idx.Search("dogs", 10)
	if len(results) != 0 {
		t.Errorf("expected no 'dogs' results after upsert, got %v", results)
	}
}

func TestIndex_EmptySearch(t *testing.T) {
	idx := New(DefaultK1, DefaultB)
	idx.Add("doc1", "some content")

	results := idx.Search("", 10)
	if len(results) != 0 {
		t.Errorf("expected no results for empty query, got %d", len(results))
	}
}

func TestIDF(t *testing.T) {
	idx := New(DefaultK1, DefaultB)

	// Common term appears in all docs.
	idx.Add("doc1", "common term unique1")
	idx.Add("doc2", "common term unique2")
	idx.Add("doc3", "common term unique3")

	// "unique1" has higher IDF (appears in 1 doc) than "common" (appears in 3).
	results := idx.Search("unique1 common", 10)
	if len(results) == 0 {
		t.Fatal("expected results")
	}

	// doc1 should score highest because "unique1" is rare.
	if results[0].ID != "doc1" {
		t.Errorf("expected doc1 first (rare term), got %s", results[0].ID)
	}
}
