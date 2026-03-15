# Zeroclaw Provider 实现分析

> 分析 zeroclaw 中 LLM provider 的抽象、工厂、兼容层与扩展方式，供 pathfinder Go 侧参考。  
> 代码位置：`zeroclaw/src/providers/`。

---

## 一、整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│  Agent / 调用方                                                   │
│  (传入 provider: Box<dyn Provider>, model, messages, tools)       │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│  Provider trait (traits.rs)                                      │
│  - capabilities() / convert_tools() / chat() / chat_with_tools()  │
│  - simple_chat / chat_with_system / chat_with_history             │
│  - stream_chat_with_system / stream_chat_with_history             │
│  - warmup()                                                       │
└───────────────────────────────┬─────────────────────────────────┘
                                │
        ┌──────────────────────┼──────────────────────┐
        │                      │                        │
┌───────▼───────┐    ┌─────────▼─────────┐    ┌────────▼────────┐
│ 主实现         │    │ OpenAiCompatible  │    │ 包装器           │
│ (anthropic,   │    │ (compatible.rs)   │    │ RouterProvider  │
│  openai,      │    │ 单实现覆盖多数     │    │ ReliableProvider│
│  ollama,      │    │ OpenAI 协议厂商    │    │ (多 provider     │
│  gemini,      │    │                   │    │  路由/故障转移)   │
│  openrouter,  │    │                   │    │                  │
│  bedrock,     │    │                   │    │                  │
│  azure_openai,│    │                   │    │                  │
│  telnyx,      │    │                   │    │                  │
│  copilot,     │    │                   │    │                  │
│  openai_codex)│    │                   │    │                  │
└───────────────┘    └──────────────────┘    └──────────────────┘
                                │
                    create_provider(name, api_key?, api_url?)
                    create_provider_with_options(...)
                    (mod.rs 工厂)
```

- **统一入口**：所有调用方只依赖 `Provider` trait，不依赖具体实现。
- **两种实现路径**：  
  1）**主实现**：每个厂商一个模块（如 `anthropic.rs`、`ollama.rs`），实现完整 trait，处理该厂商特有协议与鉴权。  
  2）**OpenAI 兼容**：`OpenAiCompatibleProvider` 一个实现，通过 `base_url + AuthStyle` 覆盖所有「OpenAI 协议」的厂商（DeepSeek、Groq、Mistral、xAI、Together、Fireworks、Moonshot、Qwen、GLM、MiniMax 等）。
- **包装器**：`RouterProvider` 按 model 前缀（如 `hint:reasoning`）路由到不同 provider+model；`ReliableProvider` 做主/备链与重试。

---

## 二、核心类型（traits.rs）

| 类型 | 用途 |
|------|------|
| `ChatMessage` | 单条消息：role + content（system/user/assistant/tool） |
| `ToolCall` | LLM 请求的工具调用：id, name, arguments |
| `ChatResponse` | 单次 chat 返回：text?, tool_calls[], usage?, reasoning_content? |
| `ChatRequest` | 请求：messages, tools?（ToolSpec 切片） |
| `ConversationMessage` | 多轮枚举：Chat \| AssistantToolCalls \| ToolResults |
| `TokenUsage` | input_tokens?, output_tokens? |
| `ProviderCapabilities` | native_tool_calling, vision |
| `ToolsPayload` | 厂商格式：Gemini \| Anthropic \| OpenAI \| PromptGuided（兜底：工具说明注入 system） |
| `StreamChunk` / `StreamOptions` / `StreamResult` | 流式输出 |
| `Provider` | 异步 trait：能力声明、工具转换、chat 系列、流式、warmup |

要点：

- **Tool 双轨**：支持 native tool calling 的 provider 实现 `convert_tools()` 返回 API 原生格式（Gemini/Anthropic/OpenAI）；不支持的返回 `PromptGuided`，由 trait 默认逻辑把工具说明注入 system prompt，再走 `chat_with_history`。
- **reasoning_content**：为 thinking 模型（DeepSeek-R1、Kimi 等）保留推理内容，便于下一轮请求原样回传。
- **capabilities()**：声明 vision、native_tool_calling，调用方可据此决定是否发图、是否用 native tools。

---

## 三、OpenAiCompatibleProvider（compatible.rs）

- **职责**：对接所有「OpenAI `/v1/chat/completions` 风格」的 API，通过配置区分厂商。
- **构造**：`name`、`base_url`、`credential?`、`AuthStyle`（Bearer / x-api-key / Custom(header)）。  
  可选：`supports_vision`、`supports_responses_fallback`（404 时是否试 `/v1/responses`）、`merge_system_into_user`（无 system 的厂商）、`user_agent`、`timeout_secs`、`extra_headers`、`api_path`。
- **请求**：将 `ChatMessage`/多轮/工具 序列化为 OpenAI 请求体，POST 到 `base_url/chat/completions`（或自定义 path），按 AuthStyle 写鉴权头。
- **响应**：解析 JSON，映射为 `ChatResponse`（text、tool_calls、usage、reasoning_content）。
- **流式**：SSE 解析，产出 `StreamChunk`。

工厂里多数厂商只需一行，例如：

- `"deepseek" => compat(OpenAiCompatibleProvider::new("DeepSeek", "https://api.deepseek.com", key, AuthStyle::Bearer))`
- `"groq"`、`"mistral"`、`"xai"`、`"together"`、`"fireworks"`、`"perplexity"`、`"cohere"` 等同理；国内厂商用 `glm_base_url(name)`、`moonshot_base_url(name)` 等返回不同 base_url，再交给同一 compatible 实现。

---

## 四、工厂与凭证（mod.rs）

- **入口**：  
  - `create_provider(name, api_key?)`  
  - `create_provider_with_options(name, api_key?, options)`  
  - `create_provider_with_url(name, api_key?, api_url?)`  
  内部统一到 `create_provider_with_url_and_options`。
- **name 解析**：  
  - 主 provider：`"openai"`、`"anthropic"`、`"ollama"`、`"gemini"`、`"openrouter"`、`"azure_openai"`、`"bedrock"`、`"telnyx"`、`"copilot"`、`"openai-codex"` 等。  
  - 别名：如 `grok`→xai、`kimi`→moonshot、`zhipu`→glm；国内多区域/多端点用 `is_*_alias()` + `*_base_url(name)`。  
  - 自定义：`custom:https://...` 解析 URL 后交给 `OpenAiCompatibleProvider`。
- **凭证**：`resolve_provider_credential(name, api_key?)`：  
  1）显式传入的 api_key（且非占位符）；  
  2）按 name 查表得到厂商 env 列表（如 `openai`→OPENAI_API_KEY，`anthropic`→ANTHROPIC_OAUTH_TOKEN/ANTHROPIC_API_KEY）；  
  3）通用 ZEROCLAW_API_KEY、API_KEY。  
  部分厂商（MiniMax、Qwen 等）有 OAuth 刷新逻辑。
- **API key 前缀校验**：`check_api_key_prefix(provider_name, key)` 防止误用（如把 Anthropic key 配给 openai），不匹配时直接 bail。
- **运行时选项**：`ProviderRuntimeOptions`：auth_profile_override、provider_api_url、zeroclaw_dir、provider_timeout_secs、extra_headers、api_path、reasoning_enabled 等，在工厂里注入到对应 provider（如 timeout/headers 通过 `compat` 闭包套在 OpenAiCompatibleProvider 上）。

---

## 五、主实现示例：Ollama（ollama.rs）

- **结构**：`OllamaProvider { base_url, api_key?, reasoning_enabled? }`；base_url 可来自环境或 `api_url` 参数，默认本地。
- **请求体**：自有格式（model, messages, stream, options, think?, tools?），与 OpenAI 不同，故单独实现。
- **实现 trait**：  
  - `capabilities()`：native_tool_calling、vision 按需为 true。  
  - `convert_tools()`：返回 OpenAI 风格（Ollama 接受类似格式）。  
  - `chat_with_system` / `chat_with_history` / `chat` / `chat_with_tools`：组 body，POST，解析 `ApiChatResponse` 为 `ChatResponse`（含 tool_calls、thinking→reasoning_content）。  
  - 流式：实现 `stream_chat_with_*`，解析 SSE。
- **与工厂对接**：工厂 match `"ollama"` 时 `OllamaProvider::new_with_reasoning(api_url, key, options.reasoning_enabled)`，无 key 也可（本地默认无鉴权）。

主实现（OpenAI、Anthropic、Gemini、OpenRouter、Bedrock、Azure、Telnyx、Copilot、OpenAI Codex）同理：各自模块内实现 `Provider`，工厂里按 name 构造并 `Box::new(...)` 或 `Ok(Box::new(...))`。

---

## 六、Router 与 Reliable

- **RouterProvider**：  
  - 持有多组 `(name, Box<dyn Provider>)` 和 `default_model`。  
  - `routes`：hint → (provider_index, model)，例如 `"reasoning"` → (1, "claude-sonnet-4").  
  - `resolve(model)`：若 model 为 `hint:xxx` 则查表得到 (provider_index, resolved_model)，否则用 default provider + 当前 model。  
  - 所有 `Provider` 方法内先 `resolve(model)` 再转发到对应 provider，对调用方透明。
- **ReliableProvider**：  
  - 主 provider + 有序 fallback 列表 + 重试策略。  
  - 错误分类：可重试（如 429、超时）vs 不可重试（4xx 除 429/408、鉴权失败、model not found 等）。  
  - 先重试主 provider，再按序 fallback，任一层成功即返回。

---

## 七、对 Go 实现的参考

| 维度 | Zeroclaw 做法 | Go 可借鉴 |
|------|----------------|-----------|
| 抽象 | 单一 `Provider` trait，chat/stream/tools/capabilities | 定义 `Provider` interface，方法：Chat、ChatStream、Capabilities、ConvertTools（或统一请求/响应结构） |
| 兼容层 | 一个 OpenAiCompatibleProvider，base_url + auth 覆盖多数厂商 | 一个 `OpenAICompatible` 实现，BaseURL + APIKey + 可选 Header 映射；新厂商只加配置或别名 |
| 工厂 | 单函数 create_provider(name, key?, url?)，内部大 match | 注册表：name → 构造函数；或 switch + 显式构造；支持 custom:URL |
| 凭证 | 显式 key 优先，再按 name 查 env 表，再通用 env | 同序：参数 → 厂商 env → 通用 env；可选 API key 前缀校验 |
| 工具 | ToolsPayload 多态 + 默认 PromptGuided 注入 system | 支持「原生 tools」与「把工具说明写进 system」两种路径；响应统一为 text + tool_calls |
| 能力 | ProviderCapabilities（vision, native_tool_calling） | 接口返回能力位，调用方决定是否发图、是否用 native tools |
| 流式 | StreamChunk + StreamOptions，默认返回“不支持” | 接口返回 stream 或 error“不支持”；消费端统一按 chunk 处理 |
| 多厂商/容错 | Router + Reliable 包装 | 可选：路由表（model/hint → provider+model）；主/备 + 重试策略 |

pathfinder 若在 Go 侧需要直连 LLM（而非只调 Shawkeye），可先做「Provider interface + OpenAI 兼容实现 + 简单工厂（name → base_url + auth）」，再按需加主实现（如 Anthropic、Ollama）与 Router/Reliable 等价逻辑。Zeroclaw 的 provider 列表与凭证 env 表可直接参考 [providers-reference](zeroclaw 仓库 docs/reference/api/providers-reference.md)。
