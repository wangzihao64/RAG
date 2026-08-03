package service

import (
	"context"
	"fmt"
)

// documentVectorStore 抽象出删除路径用到的向量库能力，便于在测试中替换。
// 注意接口刻意不含 Close：共享连接的生命周期由 InitQuery/CloseQuery 统一
// 管理，单次删除操作不能关闭共享连接。
type documentVectorStore interface {
	DeleteByCollection(ctx context.Context, collectionID int64) error
	DeleteByDocument(ctx context.Context, documentID int64) error
}

// sharedCleanupStore 返回删除路径共享的 Milvus 长连接，直接复用 InitQuery
// 初始化的 queryStore，避免每次删除都新建连接（连接握手 + HasCollection +
// LoadCollection 的开销远大于删除本身）。声明为变量是为了让测试可以替换。
// queryStore 为 nil（启动时 Milvus 不可用）时返回 nil，调用方据此返回 503。
var sharedCleanupStore = func() documentVectorStore {
	if queryStore == nil {
		// 不能把类型化 nil 直接包进接口：nil *Store 转成的接口值非 nil，
		// 判空会失效，调用方法时 panic
		return nil
	}
	return queryStore
}

func deleteDocumentChunks(ctx context.Context, documentID uint) error {
	store := sharedCleanupStore()
	if store == nil {
		return ErrQueryUnavailable
	}
	if err := store.DeleteByDocument(ctx, int64(documentID)); err != nil {
		return fmt.Errorf("删除文档 chunk 失败: %w", err)
	}
	return nil
}

func deleteCollectionChunks(ctx context.Context, collectionID uint) error {
	store := sharedCleanupStore()
	if store == nil {
		return ErrQueryUnavailable
	}
	if err := store.DeleteByCollection(ctx, int64(collectionID)); err != nil {
		return fmt.Errorf("删除知识库 chunk 失败: %w", err)
	}
	return nil
}
