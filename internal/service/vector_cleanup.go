package service

import (
	"context"
	"fmt"

	"rag/pkg/vectordb"
)

type documentVectorStore interface {
	Close() error
	DeleteByCollection(ctx context.Context, collectionID int64) error
	DeleteByDocument(ctx context.Context, documentID int64) error
}

var newDocumentVectorStore = func(ctx context.Context) (documentVectorStore, error) {
	return vectordb.New(ctx)
}

func deleteDocumentChunks(ctx context.Context, documentID uint) error {
	store, err := newDocumentVectorStore(ctx)
	if err != nil {
		return fmt.Errorf("初始化文档向量库失败: %w", err)
	}
	defer store.Close()

	if err := store.DeleteByDocument(ctx, int64(documentID)); err != nil {
		return fmt.Errorf("删除文档 chunk 失败: %w", err)
	}
	return nil
}

func deleteCollectionChunks(ctx context.Context, collectionID uint) error {
	store, err := newDocumentVectorStore(ctx)
	if err != nil {
		return fmt.Errorf("初始化文档向量库失败: %w", err)
	}
	defer store.Close()

	if err := store.DeleteByCollection(ctx, int64(collectionID)); err != nil {
		return fmt.Errorf("删除知识库 chunk 失败: %w", err)
	}
	return nil
}
