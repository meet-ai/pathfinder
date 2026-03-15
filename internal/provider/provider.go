package provider

import (
	"context"
	"encoding/json"
)

// Provider 与 zeroclaw Provider trait 对齐，供上层仅依赖此接口。
type Provider interface {
	Capabilities() Capabilities
	ConvertTools(tools []ToolSpec) ToolsPayload

	SimpleChat(ctx context.Context, message, model string, temperature float64) (string, error)
	ChatWithSystem(ctx context.Context, systemPrompt *string, message, model string, temperature float64) (string, error)
	ChatWithHistory(ctx context.Context, messages []ChatMessage, model string, temperature float64) (string, error)
	Chat(ctx context.Context, req ChatRequest, model string, temperature float64) (*ChatResponse, error)
	ChatWithTools(ctx context.Context, messages []ChatMessage, tools []json.RawMessage, model string, temperature float64) (*ChatResponse, error)

	Warmup(ctx context.Context) error
	SupportsStreaming() bool
	StreamChatWithSystem(ctx context.Context, systemPrompt *string, message, model string, temperature float64, opts StreamOptions) (<-chan StreamResult, error)
	StreamChatWithHistory(ctx context.Context, messages []ChatMessage, model string, temperature float64, opts StreamOptions) (<-chan StreamResult, error)
}
