// Package service 实现文档处理流水线：提取 → 切分 → 向量化 → 写入 Milvus。
package service

import (
	"context"
	"fmt"
	"log"

	"rag/config"
	"rag/internal/model"
	"rag/internal/repository"
	"rag/pkg/chunker"
	"rag/pkg/embedding"
	"rag/pkg/extractor"
	"rag/pkg/vectordb"
)

// Pipeline 封装处理一个文档所需的全部依赖。
type Pipeline struct {
	embedClient embedding.Client
	vectorStore *vectordb.Store
}

// NewPipeline 构造处理流水线（需要 Milvus 与 embedding 客户端就绪）。
func NewPipeline(ctx context.Context) (*Pipeline, error) {
	store, err := vectordb.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("初始化向量库失败: %w", err)
	}
	return &Pipeline{
		embedClient: embedding.New(),
		vectorStore: store,
	}, nil
}

// Close 释放资源。
func (p *Pipeline) Close() error {
	if p.vectorStore != nil {
		return p.vectorStore.Close()
	}
	return nil
}

// ProcessDocument 处理一个文档：提取 → 切分 → 向量化 → 写入 Milvus → 更新状态。
// 整个流程原子性地更新 document 状态，失败时记录 error_msg。
func (p *Pipeline) ProcessDocument(ctx context.Context, docID uint) error {
	dao := repository.NewDocumentDao(ctx)
	doc, err := dao.FindByID(docID)
	if err != nil {
		return fmt.Errorf("查询文档失败: %w", err)
	}
	if doc == nil {
		return fmt.Errorf("文档 %d 不存在", docID)
	}

	// 执行处理流水线
	if err := p.process(ctx, doc); err != nil {
		log.Printf("文档 %d 处理失败: %v", docID, err)
		_ = dao.UpdateStatus(docID, model.DocStatusFailed, err.Error())
		return err
	}

	// 标记为就绪
	return dao.UpdateStatus(docID, model.DocStatusReady, "")
}

// process 执行提取 → 切分 → 向量化 → 写入的核心流程。
func (p *Pipeline) process(ctx context.Context, doc *model.Document) error {
	// 1. 提取文本
	text, err := extractor.Extract(doc.FilePath, doc.FileType)
	if err != nil {
		return fmt.Errorf("提取文本失败: %w", err)
	}
	if text == "" {
		return fmt.Errorf("文档内容为空")
	}

	// 2. 切分文本块
	chunks := chunker.Split(text, config.ChunkSize, config.ChunkOverlap)
	if len(chunks) == 0 {
		return fmt.Errorf("切分后无有效文本块")
	}

	// 3. 向量化
	vectors, err := p.embedClient.Embed(ctx, chunks)
	if err != nil {
		return fmt.Errorf("向量化失败: %w", err)
	}
	if len(vectors) != len(chunks) {
		return fmt.Errorf("向量数量不符：期望 %d 得到 %d", len(chunks), len(vectors))
	}

	// 4. 写入 Milvus（先清理同文档的旧数据，再写入新数据）
	if err := p.vectorStore.DeleteByDocument(ctx, int64(doc.ID)); err != nil {
		log.Printf("清理文档 %d 旧向量时出错（可能首次写入）: %v", doc.ID, err)
	}

	var vdbChunks []vectordb.Chunk
	for i := range chunks {
		vdbChunks = append(vdbChunks, vectordb.Chunk{
			DocumentID:   int64(doc.ID),
			CollectionID: int64(doc.CollectionID),
			ChunkIndex:   int64(i),
			Content:      chunks[i],
			Vector:       vectors[i],
		})
	}
	if err := p.vectorStore.Insert(ctx, vdbChunks); err != nil {
		return fmt.Errorf("写入向量库失败: %w", err)
	}

	// 5. 更新文档的 chunk_count
	doc.ChunkCount = len(chunks)
	if err := repository.DB.Save(doc).Error; err != nil {
		return fmt.Errorf("更新 chunk_count 失败: %w", err)
	}

	return nil
}
