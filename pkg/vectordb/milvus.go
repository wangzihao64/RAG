// Package vectordb 封装 Milvus 向量库操作：建表、写入 chunk 向量、按文档删除、相似检索。
package vectordb

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"

	"rag/config"
)

// 字段名常量，写入与检索共用，避免拼写漂移。
const (
	fieldID           = "id"
	fieldDocumentID   = "document_id"
	fieldCollectionID = "collection_id"
	fieldChunkIndex   = "chunk_index"
	fieldContent      = "content"
	fieldVector       = "vector"

	maxContentLen = 8192 // content varchar 上限（字符）
)

// Store 是对 Milvus 的高层封装。
type Store struct {
	cli        client.Client
	collection string
	dim        int
}

// Chunk 是待写入的一条文本块及其向量。
type Chunk struct {
	DocumentID   int64
	CollectionID int64
	ChunkIndex   int64
	Content      string
	Vector       []float32
}

// SearchHit 是一次相似检索的命中结果。
type SearchHit struct {
	DocumentID   int64
	CollectionID int64
	ChunkIndex   int64
	Content      string
	Score        float32
}

// New 连接 Milvus 并确保目标 collection 存在（不存在则建表+建索引+加载）。
func New(ctx context.Context) (*Store, error) {
	cli, err := client.NewClient(ctx, client.Config{Address: config.MilvusAddress})
	if err != nil {
		return nil, fmt.Errorf("连接 Milvus 失败: %w", err)
	}
	dim := config.EmbeddingDim
	if dim <= 0 {
		dim = 1024
	}
	s := &Store{cli: cli, collection: config.MilvusCollection, dim: dim}
	if err := s.ensureCollection(ctx); err != nil {
		cli.Close()
		return nil, err
	}
	return s, nil
}

// Close 释放底层连接。
func (s *Store) Close() error { return s.cli.Close() }

// ensureCollection 保证 collection、索引存在并已加载。
func (s *Store) ensureCollection(ctx context.Context) error {
	has, err := s.cli.HasCollection(ctx, s.collection)
	if err != nil {
		return fmt.Errorf("检查 collection 失败: %w", err)
	}
	if !has {
		if err := s.createCollection(ctx); err != nil {
			return err
		}
		if err := s.createIndex(ctx); err != nil {
			return err
		}
	}
	// 检索前必须 load 到内存
	if err := s.cli.LoadCollection(ctx, s.collection, false); err != nil {
		return fmt.Errorf("加载 collection 失败: %w", err)
	}
	return nil
}

func (s *Store) createCollection(ctx context.Context) error {
	schema := entity.NewSchema().
		WithName(s.collection).
		WithDescription("RAG 文本块向量").
		WithField(entity.NewField().WithName(fieldID).WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName(fieldDocumentID).WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName(fieldCollectionID).WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName(fieldChunkIndex).WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName(fieldContent).WithDataType(entity.FieldTypeVarChar).WithMaxLength(maxContentLen)).
		WithField(entity.NewField().WithName(fieldVector).WithDataType(entity.FieldTypeFloatVector).WithDim(int64(s.dim)))

	if err := s.cli.CreateCollection(ctx, schema, 1); err != nil {
		return fmt.Errorf("创建 collection 失败: %w", err)
	}
	return nil
}

func (s *Store) createIndex(ctx context.Context) error {
	// 用 cosine 距离 + HNSW，适合语义相似检索
	idx, err := entity.NewIndexHNSW(entity.COSINE, 8, 64)
	if err != nil {
		return fmt.Errorf("构建索引参数失败: %w", err)
	}
	if err := s.cli.CreateIndex(ctx, s.collection, fieldVector, idx, false); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}
	return nil
}

// Insert 批量写入一个文档的若干 chunk。
func (s *Store) Insert(ctx context.Context, chunks []Chunk) error {
	if len(chunks) == 0 {
		return nil
	}
	n := len(chunks)
	docIDs := make([]int64, n)
	collIDs := make([]int64, n)
	idxs := make([]int64, n)
	contents := make([]string, n)
	vectors := make([][]float32, n)
	for i, c := range chunks {
		docIDs[i] = c.DocumentID
		collIDs[i] = c.CollectionID
		idxs[i] = c.ChunkIndex
		contents[i] = truncate(c.Content, maxContentLen)
		vectors[i] = c.Vector
	}

	cols := []entity.Column{
		entity.NewColumnInt64(fieldDocumentID, docIDs),
		entity.NewColumnInt64(fieldCollectionID, collIDs),
		entity.NewColumnInt64(fieldChunkIndex, idxs),
		entity.NewColumnVarChar(fieldContent, contents),
		entity.NewColumnFloatVector(fieldVector, s.dim, vectors),
	}
	if _, err := s.cli.Insert(ctx, s.collection, "", cols...); err != nil {
		return fmt.Errorf("写入向量失败: %w", err)
	}
	// 立即 flush，保证可检索
	if err := s.cli.Flush(ctx, s.collection, false); err != nil {
		return fmt.Errorf("flush 失败: %w", err)
	}
	return nil
}

// DeleteByDocument 删除某文档的所有 chunk（用于删除文档或重新处理前清理）。
func (s *Store) DeleteByDocument(ctx context.Context, documentID int64) error {
	expr := fmt.Sprintf("%s == %d", fieldDocumentID, documentID)
	if err := s.cli.Delete(ctx, s.collection, "", expr); err != nil {
		return fmt.Errorf("删除文档向量失败: %w", err)
	}
	return nil
}

// Search 在指定 collection 范围内做相似检索，返回 topK 命中。
func (s *Store) Search(ctx context.Context, collectionID int64, vector []float32, topK int) ([]SearchHit, error) {
	if topK <= 0 {
		topK = 5
	}
	sp, err := entity.NewIndexHNSWSearchParam(64)
	if err != nil {
		return nil, err
	}
	expr := fmt.Sprintf("%s == %d", fieldCollectionID, collectionID)
	outputs := []string{fieldDocumentID, fieldCollectionID, fieldChunkIndex, fieldContent}

	results, err := s.cli.Search(
		ctx, s.collection, nil, expr, outputs,
		[]entity.Vector{entity.FloatVector(vector)},
		fieldVector, entity.COSINE, topK, sp,
	)
	if err != nil {
		return nil, fmt.Errorf("检索失败: %w", err)
	}

	var hits []SearchHit
	for _, r := range results {
		docCol := columnInt64(r.Fields, fieldDocumentID)
		collCol := columnInt64(r.Fields, fieldCollectionID)
		idxCol := columnInt64(r.Fields, fieldChunkIndex)
		contentCol := columnVarChar(r.Fields, fieldContent)
		for i := 0; i < r.ResultCount; i++ {
			hits = append(hits, SearchHit{
				DocumentID:   valInt64(docCol, i),
				CollectionID: valInt64(collCol, i),
				ChunkIndex:   valInt64(idxCol, i),
				Content:      valVarChar(contentCol, i),
				Score:        r.Scores[i],
			})
		}
	}
	return hits, nil
}

// ---------------- 小工具 ----------------

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func columnInt64(cols []entity.Column, name string) *entity.ColumnInt64 {
	for _, c := range cols {
		if c.Name() == name {
			if col, ok := c.(*entity.ColumnInt64); ok {
				return col
			}
		}
	}
	return nil
}

func columnVarChar(cols []entity.Column, name string) *entity.ColumnVarChar {
	for _, c := range cols {
		if c.Name() == name {
			if col, ok := c.(*entity.ColumnVarChar); ok {
				return col
			}
		}
	}
	return nil
}

func valInt64(col *entity.ColumnInt64, i int) int64 {
	if col == nil {
		return 0
	}
	v, err := col.ValueByIdx(i)
	if err != nil {
		return 0
	}
	return v
}

func valVarChar(col *entity.ColumnVarChar, i int) string {
	if col == nil {
		return ""
	}
	v, err := col.ValueByIdx(i)
	if err != nil {
		return ""
	}
	return v
}
