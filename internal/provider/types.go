// Package provider 提供与 zeroclaw 对等的 LLM provider 抽象与实现，便于完整迁移。
// 类型与 zeroclaw traits.rs 一一对应。
package provider

import "encoding/json"

// ChatMessage 单条对话消息（system/user/assistant/tool）。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ToolCall LLM 请求的工具调用。
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// TokenUsage 单次响应的 token 统计。
type TokenUsage struct {
	InputTokens  *uint64 `json:"input_tokens,omitempty"`
	OutputTokens *uint64 `json:"output_tokens,omitempty"`
}

// ChatResponse 单次 chat 的返回（text、tool_calls、usage、reasoning_content）。
type ChatResponse struct {
	Text             *string    `json:"text,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	Usage            *TokenUsage `json:"usage,omitempty"`
	ReasoningContent *string    `json:"reasoning_content,omitempty"`
}

// ChatRequest 结构化 chat 请求（agent 循环用）。
type ChatRequest struct {
	Messages []ChatMessage
	Tools    []ToolSpec
}

// ToolSpec 工具规格（与 tools 子系统一致）。
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage  `json:"parameters,omitempty"`
}

// Capabilities 声明 provider 能力（与 zeroclaw ProviderCapabilities 对应）。
type Capabilities struct {
	NativeToolCalling bool
	Vision            bool
}

// StreamChunk 流式输出的一块内容。
type StreamChunk struct {
	Delta     string
	IsFinal   bool
	TokenCount int
}

// StreamOptions 流式请求选项。
type StreamOptions struct {
	Enabled     bool
	CountTokens bool
}

// StreamResult 流式结果：成功为 *StreamChunk，失败为 error；channel 关闭表示结束。
type StreamResult struct {
	Chunk *StreamChunk
	Err   error
}

// ToolsPayload 厂商工具负载形态（与 zeroclaw ToolsPayload 对应）。
type ToolsPayload interface {
	isToolsPayload()
}

// PromptGuidedPayload 兜底：将工具说明注入 system prompt。
type PromptGuidedPayload struct{ Instructions string }

func (PromptGuidedPayload) isToolsPayload() {}

// OpenAIToolsPayload OpenAI 格式 tools 数组。
type OpenAIToolsPayload struct{ Tools []json.RawMessage }

func (OpenAIToolsPayload) isToolsPayload() {}

// RuntimeOptions 工厂与运行时选项（与 zeroclaw ProviderRuntimeOptions 对应）。
type RuntimeOptions struct {
	ProviderAPIURL     string
	ProviderTimeoutSecs int // 0 表示用默认 120
	ExtraHeaders       map[string]string
	APIPath            string
}
