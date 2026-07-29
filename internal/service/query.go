// Package service ——本文件实现 RAG 在线查询：检索 + 大模型流式生成。
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"rag/config"
	"rag/internal/repository"
	"rag/pkg/embedding"
	"rag/pkg/llm"
	"rag/pkg/vectordb"
)

// ErrQueryUnavailable 表示在线查询资源（Milvus 等）未就绪。
// 由 InitQuery 失败（如启动时 Milvus 不可用）导致，handler 应据此返回 503。
var ErrQueryUnavailable = errors.New("在线查询服务不可用：向量库未就绪")

// 在线查询所需资源的包级单例。与 worker 的 Pipeline 各持一份 Milvus 连接，
// 刻意解耦：避免把 worker 内部资源暴露成共享单例。由 InitQuery 在启动时初始化。
var (
	queryStore *vectordb.Store
	queryEmbed embedding.Client
	queryLLM   llm.Client
)

// RetrievedChunk 是一次检索命中的片段，附带其所属文档信息，用于展示引用来源。
type RetrievedChunk struct {
	DocumentID   int64   `json:"document_id"`
	DocumentName string  `json:"document_name"`
	ChunkIndex   int64   `json:"chunk_index"`
	Content      string  `json:"content"`
	Score        float32 `json:"score"`
}

// InitQuery 初始化在线查询资源（Milvus + embedding + llm）。
// 与 worker 一样硬依赖 Milvus，失败应由调用方决定是否致命。
func InitQuery(ctx context.Context) error {
	store, err := vectordb.New(ctx)
	if err != nil {
		return fmt.Errorf("初始化查询向量库失败: %w", err)
	}
	queryStore = store
	queryEmbed = embedding.New()
	queryLLM = llm.New()
	return nil
}

// QueryAvailable 报告在线查询资源是否已就绪（Milvus + embedding + llm 均已初始化）。
func QueryAvailable() bool {
	return queryStore != nil && queryEmbed != nil && queryLLM != nil
}

// CloseQuery 释放在线查询占用的资源。
func CloseQuery() error {
	if queryStore != nil {
		return queryStore.Close()
	}
	return nil
}

// Retrieve 在指定知识库内检索与 query 最相关的 topK 片段。
// 复用 GetCollection 的可见性鉴权（owner 或 public 可访问）。
func Retrieve(ctx context.Context, collectionID, userID uint, query string, topK int) ([]RetrievedChunk, error) {
	if !QueryAvailable() {
		return nil, ErrQueryUnavailable
	}
	if _, err := GetCollection(ctx, collectionID, userID); err != nil {
		return nil, err
	}
	if topK <= 0 {
		topK = config.QueryTopK
	}

	vectors, err := queryEmbed.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("查询向量化失败: %w", err)
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("查询向量化返回为空")
	}

	hits, err := queryStore.Search(ctx, int64(collectionID), vectors[0], topK)
	if err != nil {
		return nil, err
	}

	return enrichWithDocNames(ctx, hits), nil
}

// enrichWithDocNames 批量补充命中片段的文档名，按 document_id 去重查询，避免 N+1。
func enrichWithDocNames(ctx context.Context, hits []vectordb.SearchHit) []RetrievedChunk {
	dao := repository.NewDocumentDao(ctx)
	nameCache := make(map[int64]string)
	chunks := make([]RetrievedChunk, 0, len(hits))
	for _, h := range hits {
		name, ok := nameCache[h.DocumentID]
		if !ok {
			if doc, err := dao.FindByID(uint(h.DocumentID)); err == nil && doc != nil {
				name = doc.Name
			}
			nameCache[h.DocumentID] = name
		}
		chunks = append(chunks, RetrievedChunk{
			DocumentID:   h.DocumentID,
			DocumentName: name,
			ChunkIndex:   h.ChunkIndex,
			Content:      h.Content,
			Score:        h.Score,
		})
	}
	return chunks
}

// StreamAnswer 基于已构造好的消息做大模型流式生成，每段文本经 onDelta 回调返回。
// 与 Retrieve 拆开是为了让调用方（handler）能在生成之前先把检索来源下发给前端。
func StreamAnswer(ctx context.Context, messages []llm.Message, onDelta func(string) error) error {
	if queryLLM == nil {
		return ErrQueryUnavailable
	}
	return queryLLM.ChatStream(ctx, messages, onDelta)
}

// BuildRAGMessages 依据检索到的片段构造对话消息：system 约束“仅依据资料回答”，user 为原始问题。
func BuildRAGMessages(query string, chunks []RetrievedChunk) []llm.Message {
	var sb strings.Builder
	sb.WriteString("你是一个严谨的知识库问答助手。请仅依据下面提供的资料回答用户的问题；")
	sb.WriteString("如果资料中没有相关信息，请直接说明“根据现有资料无法回答”，不要编造。\n\n")
	if len(chunks) == 0 {
		sb.WriteString("（未检索到任何相关资料）\n")
	} else {
		sb.WriteString("资料片段：\n")
		for i, c := range chunks {
			fmt.Fprintf(&sb, "[%d] 来源《%s》：%s\n", i+1, c.DocumentName, c.Content)
		}
	}

	return []llm.Message{
		{Role: llm.RoleSystem, Content: sb.String()},
		{Role: llm.RoleUser, Content: query},
	}
}
