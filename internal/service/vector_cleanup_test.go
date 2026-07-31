package service

import (
	"context"
	"errors"
	"testing"
)

type fakeDocumentVectorStore struct {
	closed             bool
	deleteCollectionID int64
	deleteDocumentID   int64
	deleteErr          error
}

func (s *fakeDocumentVectorStore) Close() error {
	s.closed = true
	return nil
}

func (s *fakeDocumentVectorStore) DeleteByDocument(_ context.Context, documentID int64) error {
	s.deleteDocumentID = documentID
	return s.deleteErr
}

func (s *fakeDocumentVectorStore) DeleteByCollection(_ context.Context, collectionID int64) error {
	s.deleteCollectionID = collectionID
	return s.deleteErr
}

func TestDeleteDocumentChunks_DeletesChunksAndClosesStore(t *testing.T) {
	store := &fakeDocumentVectorStore{}
	originalFactory := newDocumentVectorStore
	newDocumentVectorStore = func(context.Context) (documentVectorStore, error) {
		return store, nil
	}
	t.Cleanup(func() { newDocumentVectorStore = originalFactory })

	if err := deleteDocumentChunks(context.Background(), 42); err != nil {
		t.Fatalf("deleteDocumentChunks() error = %v", err)
	}
	if store.deleteDocumentID != 42 {
		t.Errorf("DeleteByDocument() documentID = %d, want 42", store.deleteDocumentID)
	}
	if !store.closed {
		t.Error("Close() was not called")
	}
}

func TestDeleteDocumentChunks_ReturnsDeleteError(t *testing.T) {
	deleteErr := errors.New("milvus unavailable")
	store := &fakeDocumentVectorStore{deleteErr: deleteErr}
	originalFactory := newDocumentVectorStore
	newDocumentVectorStore = func(context.Context) (documentVectorStore, error) {
		return store, nil
	}
	t.Cleanup(func() { newDocumentVectorStore = originalFactory })

	err := deleteDocumentChunks(context.Background(), 42)
	if !errors.Is(err, deleteErr) {
		t.Fatalf("deleteDocumentChunks() error = %v, want wrapped %v", err, deleteErr)
	}
	if !store.closed {
		t.Error("Close() was not called")
	}
}

func TestDeleteCollectionChunks_DeletesChunksAndClosesStore(t *testing.T) {
	store := &fakeDocumentVectorStore{}
	originalFactory := newDocumentVectorStore
	newDocumentVectorStore = func(context.Context) (documentVectorStore, error) {
		return store, nil
	}
	t.Cleanup(func() { newDocumentVectorStore = originalFactory })

	if err := deleteCollectionChunks(context.Background(), 7); err != nil {
		t.Fatalf("deleteCollectionChunks() error = %v", err)
	}
	if store.deleteCollectionID != 7 {
		t.Errorf("DeleteByCollection() collectionID = %d, want 7", store.deleteCollectionID)
	}
	if !store.closed {
		t.Error("Close() was not called")
	}
}
