// Package bm25 provides an inverted index with BM25 scoring for lexical search.
// It complements vector similarity search by capturing exact keyword matches.
package bm25

import (
	"math"
	"strings"
	"sync"
)

// Default BM25 parameters.
// k1 controls term frequency saturation (1.2-2.0 is typical).
// b controls document length normalization (0.75 is typical).
const (
	DefaultK1 = 1.5
	DefaultB  = 0.75
)

// Index is an in-memory inverted index for BM25 scoring.
type Index struct {
	mu sync.RWMutex

	// inverted maps term -> docID -> term frequency.
	inverted map[string]map[string]int

	// docLengths stores the token count per document.
	docLengths map[string]int

	// docPayloads stores optional payload per document (e.g. wing, room).
	docPayloads map[string]map[string]any

	// docContents stores the original content for tokenization on delete.
	docContents map[string]string

	// N is the total number of documents.
	N int

	// avgDocLength is the average document length in tokens.
	avgDocLength float64

	// totalTokens is the sum of all document lengths.
	totalTokens int

	k1 float64
	b  float64
}

// New creates a new BM25 index.
func New(k1, b float64) *Index {
	if k1 <= 0 {
		k1 = DefaultK1
	}
	if b < 0 || b > 1 {
		b = DefaultB
	}
	return &Index{
		inverted:    make(map[string]map[string]int),
		docLengths:  make(map[string]int),
		docPayloads: make(map[string]map[string]any),
		docContents: make(map[string]string),
		k1:          k1,
		b:           b,
	}
}

// Add indexes a single document with the given ID and content.
func (idx *Index) Add(docID, content string) {
	idx.AddWithPayload(docID, content, nil)
}

// AddWithPayload indexes a single document with the given ID, content, and optional payload.
func (idx *Index) AddWithPayload(docID, content string, payload map[string]any) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Remove old entry if it exists (upsert semantics).
	idx.deleteLocked(docID)

	tokens := tokenize(content)
	docLen := len(tokens)

	idx.docLengths[docID] = docLen
	idx.docContents[docID] = content
	if payload != nil {
		idx.docPayloads[docID] = payload
	}
	idx.N++
	idx.totalTokens += docLen
	idx.avgDocLength = float64(idx.totalTokens) / float64(idx.N)

	// Count term frequencies.
	tf := make(map[string]int)
	for _, tok := range tokens {
		tf[tok]++
	}

	// Add to inverted index.
	for term, count := range tf {
		if idx.inverted[term] == nil {
			idx.inverted[term] = make(map[string]int)
		}
		idx.inverted[term][docID] = count
	}
}

// Remove removes a document from the index.
func (idx *Index) Remove(docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.deleteLocked(docID)
}

func (idx *Index) deleteLocked(docID string) {
	if docLen, exists := idx.docLengths[docID]; exists {
		idx.totalTokens -= docLen
		delete(idx.docLengths, docID)
		delete(idx.docPayloads, docID)
		delete(idx.docContents, docID)
		idx.N--
		if idx.N > 0 {
			idx.avgDocLength = float64(idx.totalTokens) / float64(idx.N)
		} else {
			idx.avgDocLength = 0
		}
	}

	// Remove from inverted index.
	for term, docMap := range idx.inverted {
		delete(docMap, docID)
		if len(docMap) == 0 {
			delete(idx.inverted, term)
		}
	}
}

// Score returns the BM25 score for a query against all documents.
// Returns a map of docID -> score, sorted by score descending.
type ScoredDoc struct {
	ID      string
	Score   float64
	Payload map[string]any
}

// Search returns documents scored by BM25, sorted by score descending.
func (idx *Index) Search(query string, limit int) []ScoredDoc {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.N == 0 {
		return nil
	}

	tokens := tokenize(query)
	if len(tokens) == 0 {
		return nil
	}

	// Aggregate scores across query terms.
	scores := make(map[string]float64)

	for _, term := range tokens {
		df := len(idx.inverted[term]) // number of docs containing term
		if df == 0 {
			continue
		}

		// IDF: log((N - df + 0.5) / (df + 0.5) + 1)
		idf := math.Log(float64(idx.N-df)/float64(df) + 1.0)

		for docID, tf := range idx.inverted[term] {
			docLen := float64(idx.docLengths[docID])
			// TF component: (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * docLen/avgDL))
			tfNorm := float64(tf) * (idx.k1 + 1.0)
			tfDenom := float64(tf) + idx.k1*(1.0-idx.b+idx.b*docLen/idx.avgDocLength)
			tfComponent := tfNorm / tfDenom

			scores[docID] += idf * tfComponent
		}
	}

	// Convert to sorted slice.
	results := make([]ScoredDoc, 0, len(scores))
	for docID, score := range scores {
		results = append(results, ScoredDoc{
			ID:      docID,
			Score:   score,
			Payload: idx.docPayloads[docID],
		})
	}

	// Sort descending.
	sortScoredDocs(results)

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// tokenize splits text into lowercase tokens, stripping punctuation.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if isTokenChar(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func isTokenChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}

// sortScoredDocs sorts results by score descending (simple insertion sort).
// For large result sets, replace with sort.Slice.
func sortScoredDocs(docs []ScoredDoc) {
	for i := 1; i < len(docs); i++ {
		key := docs[i]
		j := i - 1
		for j >= 0 && docs[j].Score < key.Score {
			docs[j+1] = docs[j]
			j--
		}
		docs[j+1] = key
	}
}
