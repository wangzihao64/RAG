// Package llm 提供对话生成能力（RAG 在线问答的“生成”环节）。
// 默认对接阿里云 DashScope 的 OpenAI 兼容 chat/completions 接口，走流式输出；
// 当 provider=mock 或未配置 API key 时，退化为本地 mock，便于无 key 联调。
package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"rag/config"
)

// Message 是一条对话消息。Role 取 system / user / assistant。
type Message struct {
	Role    string
	Content string
}

// 角色常量，避免拼写漂移。
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Client 以流式方式生成回答：每产生一段文本就回调一次 onDelta。
type Client interface {
	// ChatStream 发起一次流式对话，onDelta 在每个增量片段到达时被调用。
	// 若 onDelta 返回错误（如客户端断开），流应尽快中止并返回该错误。
	ChatStream(ctx context.Context, messages []Message, onDelta func(string) error) error
}

// New 按配置构造 chat 客户端。
// provider=dashscope 且存在 API key 时用真实接口，否则回退到 mock。
func New() Client {
	if config.LLMProvider == "dashscope" && config.LLMAPIKey != "" {
		cli := openai.NewClient(
			option.WithBaseURL(config.LLMBaseURL),
			option.WithAPIKey(config.LLMAPIKey),
		)
		return &openaiClient{cli: &cli, model: config.LLMModel}
	}
	return &mockClient{}
}

// ---------------- OpenAI 兼容（DashScope，流式） ----------------

type openaiClient struct {
	cli   *openai.Client
	model string
}

// toParam 把内部 Message 转成 SDK 的消息联合类型。
func toParam(messages []Message) []openai.ChatCompletionMessageParamUnion {
	params := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case RoleSystem:
			params = append(params, openai.SystemMessage(m.Content))
		case RoleAssistant:
			params = append(params, openai.AssistantMessage(m.Content))
		default:
			params = append(params, openai.UserMessage(m.Content))
		}
	}
	return params
}

func (c *openaiClient) ChatStream(ctx context.Context, messages []Message, onDelta func(string) error) error {
	stream := c.cli.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: toParam(messages),
	})
	defer stream.Close()

	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		if err := onDelta(delta); err != nil {
			return err
		}
	}
	return stream.Err()
}

// ---------------- Mock（无 key 时联调） ----------------

// mockClient 不调用真实模型，仅把最后一条 user 消息回显并分段输出，
// 用来验证 SSE 链路是否通畅。
type mockClient struct{}

func (c *mockClient) ChatStream(ctx context.Context, messages []Message, onDelta func(string) error) error {
	var question string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			question = messages[i].Content
			break
		}
	}
	segments := []string{
		"【mock 回答】",
		"这是未配置真实大模型时的占位回复。",
		"你的问题是：",
		question,
	}
	for _, seg := range segments {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := onDelta(seg); err != nil {
			return err
		}
	}
	return nil
}
