# Provider 子系统设计：从 Zeroclaw 完整迁移

> 目标：在 pathfinder Go 端实现与 zeroclaw 等价的 LLM provider 能力，支持按 name/url/options 创建、统一 chat/stream/tools、凭证解析、OpenAI 兼容层、Router/Reliable 包装，便于从 zeroclaw 完全迁移或双端共用同一套配置与命名。

---

## 一、目标与范围

- **功能对等**：与 zeroclaw `src/providers` 对齐，包括类型、Provider 接口、工厂、凭证解析、OpenAI 兼容实现、主实现（ollama/openai/anthropic 等）、Router、Reliable。
- **配置兼容**：Provider 名、别名、环境变量与 zeroclaw 一致，便于同一份配置在 Rust/Go 间切换。
- **分层**：Provider 为「与外部 LLM 服务通信」的技术实现，归属基础设施；对上层暴露接口（Port），实现放在 `internal/provider`（或 `internal/infra/adapters/llm`），不把业务规则写进 provider。

---

## 二、类型定义（与 zeroclaw 一一对应）

| Zeroclaw | Go 类型 | 说明 |
|----------|---------|------|
| `ChatMessage` | `provider.ChatMessage` | `Role` + `Content`（system/user/assistant/tool） |
| `ToolCall` | `provider.ToolCall` | `ID`, `Name`, `Arguments`（string JSON） |
| `ChatResponse` | `provider.ChatResponse` | `Text`, `ToolCalls`, `Usage`, `ReasoningContent` |
| `ChatRequest` | `provider.ChatRequest` | `Messages`, `Tools`（可选 ToolSpec 切片） |
| `TokenUsage` | `provider.TokenUsage` | `InputTokens`, `OutputTokens`（指针表可选） |
| `ProviderCapabilities` | `provider.Capabilities` | `NativeToolCalling`, `Vision` bool |
| `ToolsPayload` | 内部使用或 `provider.ToolsPayload` | Gemini/Anthropic/OpenAI/PromptGuided 四种形态，用于 convert_tools |
| `StreamChunk` | `provider.StreamChunk` | `Delta`, `IsFinal`, `TokenCount` |
| `StreamOptions` | `provider.StreamOptions` | `Enabled`, `CountTokens` |
| `ToolSpec` | `provider.ToolSpec` | 与 tools 子系统一致：Name, Description, Parameters (JSON) |
| `ConversationMessage` | 可选 | Chat | AssistantToolCalls | ToolResults，多轮序列化用 |

- **ReasoningContent**：必须保留，thinking 模型（DeepSeek-R1、Kimi 等）需在下一轮原样回传。
- **Tool 双轨**：支持 native tool calling 的返回 API 原生格式；不支持的返回 PromptGuided，由默认逻辑把工具说明注入 system，再走 ChatWithHistory。

---

## 三、Provider 接口

```go
// Provider 与 zeroclaw Provider trait 对齐。
type Provider interface {
    // Capabilities 声明 vision、native_tool_calling。
    Capabilities() Capabilities

    // ConvertTools 将 ToolSpec 转为厂商格式；不支持 native 时返回 PromptGuided。
    ConvertTools(tools []ToolSpec) ToolsPayload

    // SimpleChat 单轮，无 system。
    SimpleChat(ctx context.Context, message, model string, temperature float64) (string, error)
    // ChatWithSystem 单轮，可选 system。
    ChatWithSystem(ctx context.Context, systemPrompt *string, message, model string, temperature float64) (string, error)
    // ChatWithHistory 多轮，无 tools。
    ChatWithHistory(ctx context.Context, messages []ChatMessage, model string, temperature float64) (string, error)
    // Chat 结构化请求（messages + 可选 tools），返回 ChatResponse（含 tool_calls）。
    Chat(ctx context.Context, req ChatRequest, model string, temperature float64) (*ChatResponse, error)
    // ChatWithTools 显式传入已序列化的 tools 负载（厂商格式）。
    ChatWithTools(ctx context.Context, messages []ChatMessage, tools []json.RawMessage, model string, temperature float64) (*ChatResponse, error)

    // SupportsNativeTools / SupportsVision 由 Capabilities() 推导，可内联实现。
    // Warmup 可选，用于 HTTP 连接池预热。
    Warmup(ctx context.Context) error

    // 流式：不支持时返回 err 或单 chunk 标记不支持。
    SupportsStreaming() bool
    StreamChatWithSystem(ctx context.Context, systemPrompt *string, message, model string, temperature float64, opts StreamOptions) (<-chan StreamResult, error)
    StreamChatWithHistory(ctx context.Context, messages []ChatMessage, model string, temperature float64, opts StreamOptions) (<-chan StreamResult, error)
}
```

- `StreamResult` 可为 `StreamChunk` 或 error；channel 关闭表示结束。
- 默认实现：不支持流式时 `SupportsStreaming() == false`，流方法返回「不支持」错误或仅发一个错误 chunk。

---

## 四、OpenAI 兼容实现（单实现覆盖多数厂商）

- **类型**：`OpenAICompatible`（或 `Compatible`），实现 `Provider`。
- **构造参数**：Name, BaseURL, Credential（可选）, AuthStyle（Bearer / XApiKey / CustomHeader(name)）；可选：SupportsVision, SupportsResponsesFallback, MergeSystemIntoUser, UserAgent, TimeoutSecs, ExtraHeaders, APIPath。
- **行为**：
  - 请求：将 `ChatMessage`/多轮/工具 序列化为 OpenAI `/v1/chat/completions` 请求体，POST 到 `BaseURL/chat/completions`（或 APIPath），按 AuthStyle 写鉴权头。
  - 响应：解析 JSON → `ChatResponse`（Text, ToolCalls, Usage, ReasoningContent）。
  - 流式：SSE 解析 → `StreamChunk` channel。
  - 404 时若 SupportsResponsesFallback 可尝试 `/v1/responses`（与 zeroclaw 一致）。
- **厂商**：通过工厂仅配置 name + base_url + auth 即可复用，不新增代码（见下表）。

---

## 五、工厂与凭证

### 5.1 工厂入口

- `CreateProvider(ctx, name string, apiKey *string, apiURL *string) (Provider, error)`
- `CreateProviderWithOptions(ctx, name string, apiKey *string, apiURL *string, opts *RuntimeOptions) (Provider, error)`
- 内部统一为 `createProviderWithOptions`；`RuntimeOptions` 含：AuthProfileOverride, ProviderAPIURL, ZeroclawDir（或 PathfinderDir）, ProviderTimeoutSecs, ExtraHeaders, APIPath, ReasoningEnabled。

### 5.2 name 解析与构造

- **主实现**（每个 name 单独构造）：
  - `openai` → OpenAI 官方实现（或兼容 + 默认 base_url）
  - `anthropic` → Anthropic 实现
  - `ollama` → Ollama 实现（base_url 来自 apiURL 或 env，无 key 也可）
  - `gemini` / `google` / `google-gemini` → Gemini 实现
  - `openrouter` → OpenRouter 实现
  - `azure_openai` / `azure` → Azure OpenAI 实现
  - `bedrock` → AWS Bedrock 实现
  - `telnyx`, `copilot`, `openai-codex` 等按需迁移
- **OpenAI 兼容**（单实现 + 配置）：
  - deepseek, groq, mistral, xai, grok, together, fireworks, novita, perplexity, cohere, venice, vercel, cloudflare, moonshot, kimi, synthetic, opencode, opencode-go, zai, glm, minimax, qianfan, doubao, qwen, lmstudio, llamacpp, sglang, vllm, osaurus, nvidia 等（与 zeroclaw 表一致）。
- **别名**：grok→xai, kimi→moonshot, zhipu→glm, baidu→qianfan, volcengine/ark→doubao 等；国内多区域用 `*_base_url(name)` 返回不同 BaseURL 再交给同一 Compatible 实现。
- **自定义**：`custom:https://...` 解析 URL 后仅用 `OpenAICompatible` 构造。

### 5.3 凭证解析顺序

1. 显式传入的 `apiKey`（非空且非占位符）
2. 按 name 查表得到厂商环境变量列表（见下表），取第一个有值的
3. 通用 `PATHFINDER_API_KEY` 或 `ZEROCLAW_API_KEY`、`API_KEY`

OAuth 等复杂逻辑（MiniMax、Qwen 等）可在后续迭代实现，首版可只做 API key。

### 5.4 环境变量表（与 zeroclaw 一致）

| name | 环境变量（顺序） |
|------|------------------|
| openai | OPENAI_API_KEY |
| anthropic | ANTHROPIC_OAUTH_TOKEN, ANTHROPIC_API_KEY |
| openrouter | OPENROUTER_API_KEY |
| ollama | OLLAMA_API_KEY（可选） |
| gemini | GEMINI_API_KEY, GOOGLE_API_KEY |
| deepseek | DEEPSEEK_API_KEY |
| groq | GROQ_API_KEY |
| mistral | MISTRAL_API_KEY |
| xai / grok | XAI_API_KEY |
| together | TOGETHER_API_KEY |
| fireworks | FIREWORKS_API_KEY |
| perplexity | PERPLEXITY_API_KEY |
| cohere | COHERE_API_KEY |
| venice | VENICE_API_KEY |
| vercel | VERCEL_API_KEY |
| cloudflare | CLOUDFLARE_API_KEY |
| moonshot / kimi | MOONSHOT_API_KEY |
| glm / zhipu | GLM_API_KEY |
| minimax | MINIMAX_OAUTH_TOKEN, MINIMAX_API_KEY |
| qianfan / baidu | QIANFAN_API_KEY |
| doubao / ark | ARK_API_KEY, DOUBAO_API_KEY |
| qwen / dashscope | DASHSCOPE_API_KEY |
| azure_openai | AZURE_OPENAI_API_KEY |
| bedrock | AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY（无单 key） |
| lmstudio / llamacpp / sglang / vllm / osaurus / nvidia / ... | 见 [providers-reference](https://github.com/zeroclaw-labs/zeroclaw/blob/main/docs/reference/api/providers-reference.md) · [Zeroclaw 中文](https://zeroclaws.io/zh/) |

### 5.5 API Key 前缀校验

- `CheckAPIKeyPrefix(providerName, key string) (likelyProvider string, ok bool)`：若 key 前缀明显属于另一厂商（如 sk-ant-→anthropic, sk-or-→openrouter, gsk_→groq），返回该厂商名，由调用方决定是否报错（与 zeroclaw 一致，防止误配）。

---

## 六、Router 与 Reliable

- **Router**：
  - 持有多组 `(name string, p Provider)` 及 defaultModel。
  - routes: `map[string]Route`，key 为 hint（如 `reasoning`），value 为 `{ ProviderName, Model }`。
  - `Resolve(model string) (Provider, resolvedModel string)`：若 model 为 `hint:xxx` 则查表得到 Provider + Model，否则用 default provider + 当前 model。
  - 所有 Provider 方法内先 Resolve 再转发到对应 provider。
- **Reliable**：
  - 主 Provider + 有序 Fallback 列表 + 重试策略（次数/间隔）。
  - 错误分类：可重试（429、408、超时）vs 不可重试（4xx 除 429/408、鉴权失败、model not found 等）；不可重试立即返回，可重试先重试主再按序 fallback。
  - 与 zeroclaw ReliableProvider 行为一致。

---

## 七、包布局（Go）

```
internal/
  provider/
    types.go          # ChatMessage, ChatRequest, ChatResponse, ToolCall, TokenUsage, Capabilities, StreamChunk, ToolSpec, ToolsPayload
    provider.go       # Provider interface
    compatible.go     # OpenAICompatible 实现
    factory.go        # CreateProvider, CreateProviderWithOptions, createProviderWithOptions
    credential.go     # ResolveCredential, env 表, CheckAPIKeyPrefix
    router.go         # Router 实现 Provider
    reliable.go       # Reliable 实现 Provider
    openai.go         # 主实现（可选，或直接用 compatible + 默认 URL）
    ollama.go         # 主实现
    anthropic.go      # 主实现（可选）
    ...
```

- 若需严格 DDD：可将 `Provider` 接口置于应用层或领域层 port，`internal/provider` 仅保留实现与工厂；首版可把接口与实现同放 `internal/provider` 以简化依赖。

---

## 八、迁移清单（Zeroclaw → Pathfinder Go）

| 项 | Zeroclaw | Pathfinder Go | 备注 |
|----|----------|----------------|------|
| 类型 | traits.rs | provider/types.go + provider.go | 1:1 字段与语义 |
| Provider trait | async_trait Provider | Provider interface | 方法一一对应，context 显式传递 |
| OpenAI 兼容 | compatible.rs OpenAiCompatibleProvider | compatible.go OpenAICompatible | base_url + AuthStyle + 可选参数 |
| 工厂 | create_provider_with_url_and_options | CreateProviderWithOptions | name → 构造 + compat 闭包等价（选项注入） |
| 凭证 | resolve_provider_credential | ResolveCredential + env 表 | 顺序与 env 表与 zeroclaw 一致 |
| 主实现 | openai, anthropic, ollama, gemini, ... | 按需实现，优先 compatible | ollama 协议不同必须单独实现 |
| 别名 | is_*_alias, *_base_url | 同逻辑函数或表 | grok→xai, kimi→moonshot 等 |
| custom | custom:URL | 解析 URL → OpenAICompatible | |
| Router | router.rs RouterProvider | router.go Router | hint → (provider, model) |
| Reliable | reliable.rs ReliableProvider | reliable.go Reliable | 主/备 + 可重试判断 |
| 流式 | StreamChunk, stream_chat_* | StreamChunk, StreamChat* 返回 channel | |
| ToolsPayload | Gemini/Anthropic/OpenAI/PromptGuided | 同四种，ConvertTools 返回 | |
| reasoning_content | ChatResponse.reasoning_content | ChatResponse.ReasoningContent | 必须保留 |

---

## 九、实现顺序建议

1. **types + Provider 接口**：定义所有类型与接口，无实现。
2. **credential**：ResolveCredential + env 表 + CheckAPIKeyPrefix。
3. **OpenAICompatible**：实现 Provider（Chat/ChatWithTools/Stream），覆盖 deepseek、groq、openai（默认 URL）等。
4. **factory**：CreateProviderWithOptions，仅支持 OpenAI 兼容 name + custom:URL。
5. **ollama**：主实现，对接本地/远程 Ollama。
6. **router / reliable**：按需。
7. **anthropic / gemini / openrouter**：按需迁移主实现。

完成后，pathfinder 可通过「配置 provider name + env」与 zeroclaw 共用同一套 provider 命名与凭证，实现从 zeroclaw 的完整迁移与双端一致行为。
