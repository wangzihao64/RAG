package service

import (
	"context"
	"errors"
	"testing"

	"rag/pkg/reranker"
	"rag/pkg/vectordb"
)

type testReranker struct {
	results []reranker.Result
	err     error
}

func (r testReranker) Rerank(context.Context, string, []string, int) ([]reranker.Result, error) {
	return r.results, r.err
}

func TestRerankHits_ReordersAndUpdatesScore(t *testing.T) {
	original := []vectordb.SearchHit{
		{Content: "first", Score: 0.9},
		{Content: "second", Score: 0.8},
		{Content: "third", Score: 0.7},
	}
	queryReranker = testReranker{results: []reranker.Result{
		{Index: 2, Score: 0.99},
		{Index: 0, Score: 0.88},
	}}
	t.Cleanup(func() { queryReranker = nil })

	got := rerankHits(context.Background(), "query", original, 2)
	if len(got) != 2 || got[0].Content != "third" || got[1].Content != "first" {
		t.Fatalf("unexpected reranked hits: %#v", got)
	}
	if got[0].Score != 0.99 || got[1].Score != 0.88 {
		t.Fatalf("unexpected rerank scores: %#v", got)
	}
}

func TestRerankHits_FallsBackOnError(t *testing.T) {
	original := []vectordb.SearchHit{
		{Content: "first"},
		{Content: "second"},
		{Content: "third"},
	}
	queryReranker = testReranker{err: errors.New("unavailable")}
	t.Cleanup(func() { queryReranker = nil })

	got := rerankHits(context.Background(), "query", original, 2)
	if len(got) != 2 || got[0].Content != "first" || got[1].Content != "second" {
		t.Fatalf("unexpected fallback hits: %#v", got)
	}
}
