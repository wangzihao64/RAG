// Package embedding 提供文本向量化能力。
// 默认对接阿里云 DashScope 的 OpenAI 兼容 embeddings 接口；
// 当 provider=mock 或未配置 API key 时，退化为本地确定性 mock，便于打通流水线与测试。
package embedding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"rag/config"
)

// Client 把一批文本转成等长的向量。
type Client interface {
	// Embed 返回与 texts 一一对应、维度为 Dim() 的向量切片。
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Dim 返回向量维度。
	Dim() int
}

// New 按配置构造 embedding 客户端。
// provider=dashscope 且存在 API key 时用真实接口，否则回退到 mock。
func New() Client {
	dim := config.EmbeddingDim
	if dim <= 0 {
		dim = 1024
	}
	if config.EmbeddingProvider == "dashscope" && config.EmbeddingAPIKey != "" {
		return &dashScopeClient{
			baseURL: config.EmbeddingBaseURL,
			apiKey:  config.EmbeddingAPIKey,
			model:   config.EmbeddingModel,
			dim:     dim,
			http:    &http.Client{Timeout: 30 * time.Second},
		}
	}
	return &mockClient{dim: dim}
}

// ---------------- DashScope（OpenAI 兼容） ----------------

type dashScopeClient struct {
	baseURL string
	apiKey  string
	model   string
	dim     int
	http    *http.Client
}

func (c *dashScopeClient) Dim() int { return c.dim }

type embedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	// 出错时 DashScope 返回的错误信息
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *dashScopeClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(embedRequest{
		Model:      c.model,
		Input:      texts,
		Dimensions: c.dim,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding 接口返回 %d: %s", resp.StatusCode, string(raw))
	}

	var out embedResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Code != "" {
		return nil, fmt.Errorf("embedding 接口错误 %s: %s", out.Code, out.Message)
	}
	if len(out.Data) != len(texts) {
		return nil, fmt.Errorf("embedding 返回数量不符：期望 %d 得到 %d", len(texts), len(out.Data))
	}

	// 按 index 归位，保证与输入顺序一致
	vectors := make([][]float32, len(texts))
	for _, d := range out.Data {
		if d.Index < 0 || d.Index >= len(texts) {
			return nil, errors.New("embedding 返回了越界的 index")
		}
		vectors[d.Index] = d.Embedding
	}
	for i, v := range vectors {
		if len(v) != c.dim {
			return nil, fmt.Errorf("第 %d 个向量维度为 %d，期望 %d", i, len(v), c.dim)
		}
	}
	return vectors, nil
}

// ---------------- Mock（无 key 时打通流水线） ----------------

// mockClient 用文本哈希生成确定性的归一化向量，同文本得到同向量。
type mockClient struct {
	dim int
}

func (c *mockClient) Dim() int { return c.dim }

func (c *mockClient) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i, t := range texts {
		vectors[i] = c.hashVector(t)
	}
	return vectors, nil
}

// hashVector 由文本内容派生一个确定性单位向量。
func (c *mockClient) hashVector(text string) []float32 {
	v := make([]float32, c.dim)
	var sum float64
	seed := []byte(text)
	for i := 0; i < c.dim; i++ {
		h := sha256.Sum256(append(seed, byte(i), byte(i>>8)))
		u := binary.LittleEndian.Uint32(h[:4])
		// 映射到 [-1, 1]
		f := float64(u)/float64(math.MaxUint32)*2 - 1
		v[i] = float32(f)
		sum += f * f
	}
	// L2 归一化
	norm := math.Sqrt(sum)
	if norm > 0 {
		for i := range v {
			v[i] = float32(float64(v[i]) / norm)
		}
	}
	return v
}
