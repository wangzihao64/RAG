package service

import (
	"context"
	"errors"
	"testing"
)

type fakeDocumentVectorStore struct {
	deleteCollectionID int64
	deleteDocumentID   int64
	deleteErr          error
}

func (s *fakeDocumentVectorStore) DeleteByDocument(_ context.Context, documentID int64) error {
	s.deleteDocumentID = documentID
	return s.deleteErr
}

func (s *fakeDocumentVectorStore) DeleteByCollection(_ context.Context, collectionID int64) error {
	s.deleteCollectionID = collectionID
	return s.deleteErr
}

// stubCleanupStore 用 fake 替换共享连接，测试结束后还原。
func stubCleanupStore(t *testing.T, store documentVectorStore) {
	t.Helper()
	original := sharedCleanupStore
	sharedCleanupStore = func() documentVectorStore { return store }
	t.Cleanup(func() { sharedCleanupStore = original })
}

func TestDeleteDocumentChunks_DeletesChunksWithSharedStore(t *testing.T) {
	store := &fakeDocumentVectorStore{}
	stubCleanupStore(t, store)

	if err := deleteDocumentChunks(context.Background(), 42); err != nil {
		t.Fatalf("deleteDocumentChunks() error = %v", err)
	}
	if store.deleteDocumentID != 42 {
		t.Errorf("DeleteByDocument() documentID = %d, want 42", store.deleteDocumentID)
	}
}

func TestDeleteDocumentChunks_ReturnsDeleteError(t *testing.T) {
	deleteErr := errors.New("milvus unavailable")
	stubCleanupStore(t, &fakeDocumentVectorStore{deleteErr: deleteErr})

	err := deleteDocumentChunks(context.Background(), 42)
	if !errors.Is(err, deleteErr) {
		t.Fatalf("deleteDocumentChunks() error = %v, want wrapped %v", err, deleteErr)
	}
}

func TestDeleteDocumentChunks_StoreUnavailable(t *testing.T) {
	stubCleanupStore(t, nil)

	err := deleteDocumentChunks(context.Background(), 42)
	if !errors.Is(err, ErrQueryUnavailable) {
		t.Fatalf("deleteDocumentChunks() error = %v, want %v", err, ErrQueryUnavailable)
	}
}

func TestDeleteCollectionChunks_DeletesChunksWithSharedStore(t *testing.T) {
	store := &fakeDocumentVectorStore{}
	stubCleanupStore(t, store)

	if err := deleteCollectionChunks(context.Background(), 7); err != nil {
		t.Fatalf("deleteCollectionChunks() error = %v", err)
	}
	if store.deleteCollectionID != 7 {
		t.Errorf("DeleteByCollection() collectionID = %d, want 7", store.deleteCollectionID)
	}
}

func TestDeleteCollectionChunks_StoreUnavailable(t *testing.T) {
	stubCleanupStore(t, nil)

	err := deleteCollectionChunks(context.Background(), 7)
	if !errors.Is(err, ErrQueryUnavailable) {
		t.Fatalf("deleteCollectionChunks() error = %v, want %v", err, ErrQueryUnavailable)
	}
}
