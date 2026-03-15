package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AuthStyle 鉴权方式（与 zeroclaw AuthStyle 对应）。
type AuthStyle int

const (
	AuthBearer AuthStyle = iota
	AuthXApiKey
)

const defaultTimeoutSecs = 120

// OpenAICompatible 实现 OpenAI /v1/chat/completions 协议的 provider（DeepSeek、Groq 等）。
type OpenAICompatible struct {
	name       string
	baseURL    string
	credential string
	authStyle  AuthStyle
	timeout    time.Duration
	client     *http.Client
}

// NewOpenAICompatible 构造 OpenAI 兼容 provider。baseURL 不含尾部斜杠，不含 /v1。
func NewOpenAICompatible(name, baseURL, credential string, authStyle AuthStyle) *OpenAICompatible {
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	timeout := defaultTimeoutSecs * time.Second
	return &OpenAICompatible{
		name:       name,
		baseURL:    baseURL,
		credential: credential,
		authStyle:  authStyle,
		timeout:    timeout,
		client:     &http.Client{Timeout: timeout},
	}
}

// WithTimeout 覆盖默认超时。
func (c *OpenAICompatible) WithTimeout(secs int) *OpenAICompatible {
	if secs > 0 {
		c.timeout = time.Duration(secs) * time.Second
		c.client = &http.Client{Timeout: c.timeout}
	}
	return c
}

func (c *OpenAICompatible) Capabilities() Capabilities {
	return Capabilities{NativeToolCalling: true, Vision: false}
}

func (c *OpenAICompatible) ConvertTools(tools []ToolSpec) ToolsPayload {
	if len(tools) == 0 {
		return PromptGuidedPayload{}
	}
	out := make([]json.RawMessage, 0, len(tools))
	for _, t := range tools {
		raw := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
		b, _ := json.Marshal(raw)
		out = append(out, b)
	}
	return OpenAIToolsPayload{Tools: out}
}

func (c *OpenAICompatible) SimpleChat(ctx context.Context, message, model string, temperature float64) (string, error) {
	return c.ChatWithSystem(ctx, nil, message, model, temperature)
}

func (c *OpenAICompatible) ChatWithSystem(ctx context.Context, systemPrompt *string, message, model string, temperature float64) (string, error) {
	msgs := []ChatMessage{}
	if systemPrompt != nil && *systemPrompt != "" {
		msgs = append(msgs, ChatMessage{Role: "system", Content: *systemPrompt})
	}
	msgs = append(msgs, ChatMessage{Role: "user", Content: message})
	resp, err := c.chat(ctx, msgs, nil, model, temperature, false)
	if err != nil {
		return "", err
	}
	if resp.Text != nil {
		return *resp.Text, nil
	}
	return "", nil
}

func (c *OpenAICompatible) ChatWithHistory(ctx context.Context, messages []ChatMessage, model string, temperature float64) (string, error) {
	resp, err := c.chat(ctx, messages, nil, model, temperature, false)
	if err != nil {
		return "", err
	}
	if resp.Text != nil {
		return *resp.Text, nil
	}
	return "", nil
}

func (c *OpenAICompatible) Chat(ctx context.Context, req ChatRequest, model string, temperature float64) (*ChatResponse, error) {
	var tools []json.RawMessage
	if len(req.Tools) > 0 {
		payload := c.ConvertTools(req.Tools)
		if p, ok := payload.(OpenAIToolsPayload); ok {
			tools = p.Tools
		}
	}
	return c.chat(ctx, req.Messages, tools, model, temperature, false)
}

func (c *OpenAICompatible) ChatWithTools(ctx context.Context, messages []ChatMessage, tools []json.RawMessage, model string, temperature float64) (*ChatResponse, error) {
	return c.chat(ctx, messages, tools, model, temperature, false)
}

func (c *OpenAICompatible) Warmup(ctx context.Context) error { return nil }

func (c *OpenAICompatible) SupportsStreaming() bool { return false }

func (c *OpenAICompatible) StreamChatWithSystem(ctx context.Context, _ *string, _, _ string, _ float64, _ StreamOptions) (<-chan StreamResult, error) {
	return nil, fmt.Errorf("%s: streaming not implemented", c.name)
}

func (c *OpenAICompatible) StreamChatWithHistory(ctx context.Context, _ []ChatMessage, _ string, _ float64, _ StreamOptions) (<-chan StreamResult, error) {
	return nil, fmt.Errorf("%s: streaming not implemented", c.name)
}

// --- 内部：OpenAI 请求/响应结构 ---

type openAIReq struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64        `json:"temperature"`
	Stream      bool            `json:"stream"`
	Tools       []json.RawMessage `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResp struct {
	Choices []struct {
		Message struct {
			Role            string `json:"role"`
			Content         string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
			ToolCalls       []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     uint64 `json:"prompt_tokens"`
		CompletionTokens uint64 `json:"completion_tokens"`
	} `json:"usage,omitempty"`
}

func (c *OpenAICompatible) chat(ctx context.Context, messages []ChatMessage, tools []json.RawMessage, model string, temperature float64, stream bool) (*ChatResponse, error) {
	msgs := make([]openAIMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}
	body := openAIReq{
		Model:       model,
		Messages:    msgs,
		Temperature: temperature,
		Stream:      stream,
		Tools:       tools,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	path := c.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.credential != "" {
		switch c.authStyle {
		case AuthBearer:
			req.Header.Set("Authorization", "Bearer "+c.credential)
		case AuthXApiKey:
			req.Header.Set("x-api-key", c.credential)
		}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		return nil, fmt.Errorf("%s API error (%d): %s", c.name, resp.StatusCode, string(b))
	}
	var out openAIResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("%s: empty choices", c.name)
	}
	msg := out.Choices[0].Message
	r := &ChatResponse{}
	if msg.Content != "" {
		r.Text = &msg.Content
	}
	if msg.ReasoningContent != "" {
		r.ReasoningContent = &msg.ReasoningContent
	}
	for _, tc := range msg.ToolCalls {
		r.ToolCalls = append(r.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	if out.Usage != nil {
		r.Usage = &TokenUsage{
			InputTokens:  &out.Usage.PromptTokens,
			OutputTokens: &out.Usage.CompletionTokens,
		}
	}
	return r, nil
}
