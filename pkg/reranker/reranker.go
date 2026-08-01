// Package reranker 提供文本重排能力。
package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"rag/config"
)

type Client interface {
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]Result, error)
}

type Result struct {
	Index int
	Score float32
}

type dashScopeClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

type rerankRequest struct {
	Model     string       `json:"model"`
	Input     rerankInput  `json:"input"`
	Parameter rerankParams `json:"parameters,omitempty"`
}

type rerankInput struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

type rerankParams struct {
	TopN int `json:"top_n,omitempty"`
}

type rerankResponse struct {
	Output struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float32 `json:"relevance_score"`
		} `json:"results"`
	} `json:"output"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New() Client {
	if !config.RerankEnabled || config.RerankAPIKey == "" {
		return nil
	}
	return &dashScopeClient{
		baseURL: config.RerankBaseURL,
		apiKey:  config.RerankAPIKey,
		model:   config.RerankModel,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *dashScopeClient) Rerank(
	ctx context.Context,
	query string,
	documents []string,
	topN int,
) ([]Result, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(rerankRequest{
		Model: c.model,
		Input: rerankInput{
			Query:     query,
			Documents: documents,
		},
		Parameter: rerankParams{TopN: topN},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL,
		bytes.NewReader(body),
	)
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

	var output rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("rerank 接口返回 %d: %s", resp.StatusCode, output.Message)
	}
	if output.Code != "" {
		return nil, fmt.Errorf("rerank 接口错误 %s: %s", output.Code, output.Message)
	}

	results := make([]Result, 0, len(output.Output.Results))
	for _, item := range output.Output.Results {
		results = append(results, Result{Index: item.Index, Score: item.RelevanceScore})
	}
	return results, nil
}
