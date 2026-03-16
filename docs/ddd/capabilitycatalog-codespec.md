## CapabilityCatalogContext — Codespec

> 当前聚焦：`ListAgentsUseCase`（用户查看所有可用 agent 列表）、`DescribeAgentUseCase`（用户查看单个 agent 详情）

## 1. 参与者与接口

- **调用方（上游 RuntimeContext / JobManagementContext / CLI / TUI）**
  - 调用点：当用户在 CLI/TUI/Web 中选择「查看所有可用 agent」或需要为规划/派发选择 agent 时。
  - 职责：发起无参数或带简单过滤条件的查询请求，消费返回的 Agent 列表 DTO。

- **应用服务（CapabilityCatalogQueryService，后续由代码收紧到具体方法签名）**
  - `ListAgents(ctx context.Context, query ListAgentsQuery) ([]AgentSummaryDTO, error)`

- **参与的领域服务 / Port（来自 CapabilityCatalogContext 与 RuntimeContext）**
  - `AgentDirectory`（能力目录视图的读取端口，可由 RuntimeContext/daemon 维护）：  
    - `ListAll(ctx context.Context) ([]AgentRecord, error)`  
    - 说明：对上游屏蔽具体来源（openclaw/Cursor/OpenCode/ClaudeCode/本地），只提供聚合后的目录视图。

## 2. ListAgentsUseCase — 编排步骤表（应用层视角）

| 步骤 | 业务步骤（来自用例） | 应用层动作 | 调用的应用服务方法 | 触达的领域服务 / Port |
| ---- | -------------------- | ---------- | ------------------ | ---------------------- |
| 1 | 用户在 CLI 或 TUI 中选择「查看所有可用 agent」入口，发起查看请求。 | 上游入口（CLI/TUI/Web）构造 `ListAgentsQuery`（当前可为空或仅包含简单过滤条件，如标签/分组）。 | （入口仅负责转发，不实现业务） | — |
| 2 | 前端或 CLI 调用后端接口（例如 `GET /agents`），将请求转交给能力目录上下文对应的查询服务。 | HTTP handler/IPC handler 解码请求，调用 `CapabilityCatalogQueryService.ListAgents(ctx, query)`。 | `ListAgents(ctx, query)` | — |
| 3 | 查询服务从当前的 Agent 目录视图（由 RuntimeContext/daemon 维护的缓存或仓储）中读取所有可用 agent 的基础信息。 | 应用服务调用 `AgentDirectory.ListAll(ctx)` 取得聚合后的 Agent 记录列表。 | `ListAgents(ctx, query)` | `AgentDirectory` |
| 4 | 查询服务按约定的排序规则（例如按 agent 名称或分组）返回 agent 列表 DTO，包含每个 agent 的标识、名称、简要说明及关键能力标签。 | 应用服务在内存中对 `AgentRecord` 做排序与映射，生成 `[]AgentSummaryDTO` 返回给调用方。 | `ListAgents(ctx, query)` | —（纯内存转换） |
| 5 | CLI 或 TUI 根据返回的 agent 列表，在终端或界面上以列表形式展示所有可用 agent，支持用户后续选择某个 agent 进行派发或查看详情。 | 上游入口渲染 `AgentSummaryDTO` 列表（表格/列表/选择器）。 | （入口内部逻辑） | — |

## 3. DescribeAgentUseCase — 编排步骤表（应用层视角）

### 3.1 参与者与接口补充

- **应用服务（继续复用 `CapabilityCatalogQueryService`）**
  - 新增方法：`DescribeAgent(ctx context.Context, query DescribeAgentQuery) (AgentDetailDTO, error)`

- **新增/复用 Port**
  - `AgentDirectory`：
    - 新增：`GetByID(ctx context.Context, id AgentID) (AgentRecord, error)`
  - `ProductMetricsProvider`（按产品收集 agent 相关指标的 Port，存在于各产品上下文，由 CapabilityCatalog 通过 Port 调用）：
    - `CollectForAgent(ctx context.Context, productID ProductID, agentID AgentID) (ProductSpecificMetrics, error)`

### 3.2 编排步骤表

| 步骤 | 业务步骤（来自用例） | 应用层动作 | 调用的应用服务方法 | 触达的领域服务 / Port |
| ---- | -------------------- | ---------- | ------------------ | ---------------------- |
| 1 | 用户在 CLI 或 TUI 的 agent 列表界面中选择某个 agent，或在命令行中直接输入 agent 标识，发起「查看详情」请求。 | 上游入口（CLI/TUI/Web）构造 `DescribeAgentQuery`，包含 `AgentID`，可选包含 `ProductID`（当前产品标识，如 Cursor/OpenClaw）。 | （入口仅负责转发，不实现业务） | — |
| 2 | 前端或 CLI 将用户请求转交给能力目录上下文对应的查询服务。 | HTTP handler/IPC handler 解码请求，调用 `CapabilityCatalogQueryService.DescribeAgent(ctx, query)`。 | `DescribeAgent(ctx, query)` | — |
| 3 | 查询服务从当前 Agent 目录视图中读取该 agent 的基础信息与能力列表。 | 应用服务从 `DescribeAgentQuery` 中解析出 `AgentID`，调用 `AgentDirectory.GetByID(ctx, agentID)` 获取 `AgentRecord`。 | `DescribeAgent(ctx, query)` | `AgentDirectory` |
| 4 | 查询服务根据不同产品的需要，收集该 agent 在当前产品下的扩展指标（如 Cursor 的实例数/skill 数，OpenClaw 的 subagent 数量/常见编排深度等）。 | 若 `DescribeAgentQuery` 中包含 `ProductID`，则调用 `ProductMetricsProvider.CollectForAgent(ctx, productID, agentID)` 获取 `ProductSpecificMetrics`；若未提供则可跳过或返回空扩展指标。 | `DescribeAgent(ctx, query)` | `ProductMetricsProvider` |
| 5 | 查询服务将通用字段与产品特定指标整合为统一的详情 DTO。 | 应用服务在内存中将 `AgentRecord` 映射为通用部分（基础信息、能力与接口视图、运行约束与危险能力提示），并将 `ProductSpecificMetrics` 放入 `AgentDetailDTO.ProductMetrics` 或等价结构中，返回给调用方。 | `DescribeAgent(ctx, query)` | —（纯内存转换） |
| 6 | CLI 或 TUI 将 `AgentDetailDTO` 渲染为多分区详情视图，方便用户理解该 agent 的能力与在当前产品中的使用情况。 | 上游入口根据 DTO 渲染通用信息区和产品特定指标区，并可挂载推荐使用场景与快捷操作入口。 | （入口内部逻辑） | — |

## 4. DTO 形状约定（List / Describe 共享）

### 4.1 标识与基础类型

- `AgentID`：字符串，当前产品内唯一标识某个 agent，例如 `cursor-code-navigator`、`openclaw-orchestrator`。
- `ProductID`：字符串，标识当前产品，如 `cursor`, `openclaw`, `standalone` 等。

### 4.2 查询与结果 DTO

- `ListAgentsQuery`
  - `ProductID`（可选）：字符串，标记当前调用来自哪个产品，便于后续做产品特定过滤或排序。
  - `Keyword`（可选）：字符串，按名称/简介模糊匹配的关键字。
  - `Tags`（可选）：字符串数组，按标签过滤 agent（如 `["refactor", "tests"]`）。
  - `Group`（可选）：字符串，按分组过滤 agent（如「代码分析」「运行时管理」）。

- `AgentSummaryDTO`
  - `ID`：`AgentID`。
  - `Name`：字符串，用户可读名称。
  - `Description`：字符串，简短说明该 agent 用于什么场景。
  - `Version`（可选）：字符串，agent 的版本号或修订标识。
  - `Tags`：字符串数组，能力/领域标签（如 `["refactor", "go", "tests"]`）。
  - `Groups`（可选）：字符串数组，分类分组（如「代码分析」「运行时监控」）。

- `DescribeAgentQuery`
  - `AgentID`：`AgentID`，必填，指定要查看详情的 agent。
  - `ProductID`（可选）：`ProductID`，当前所在产品，用于决定是否附加产品特定指标。

- `AgentDetailDTO`
  - 通用基础信息：
    - `ID`：`AgentID`。
    - `Name`：字符串。
    - `Description`：字符串，较 `AgentSummaryDTO` 更详细的介绍。
    - `Version`（可选）：字符串。
  - 能力与接口视图：
    - `Capabilities`：`[]AgentCapabilityDTO`，列出该 agent 暴露的主要能力。
    - `DangerousOperations`：字符串数组，列出可能具有破坏性的操作类别（如「写文件」「执行 shell」「访问网络」），用于在 UI/TUI 中高亮提示。
  - 运行约束与支持环境：
    - `SupportedContexts`：字符串数组，指明该 agent 支持的调用上下文（如 `["cli", "tui", "cursor-panel"]`）。
    - `Requirements`：字符串数组，列出使用前置条件或依赖（如「需要 git 仓库」「需要远程 LLM API Key」）。
  - 产品特定指标（扩展区）：
    - `ProductMetrics`：`ProductSpecificMetricsDTO`，承载与当前产品相关的统计与状态信息（见下文）。

- `AgentCapabilityDTO`
  - `Name`：字符串，能力名称（如「AnalyzeCodebase」「RunE2ETests」）。
  - `Description`：字符串，能力用途说明。
  - `InputSummary`：字符串，概述该能力典型需要的输入（如「需要一个目录路径和可选的文件模式」）。
  - `OutputSummary`：字符串，概述该能力典型输出（如「生成重构建议列表」「输出测试报告路径」）。

- `ProductSpecificMetricsDTO`
  - `ProductID`：`ProductID`，表明这些指标属于哪个产品。
  - `Metrics`：`[]MetricEntryDTO`，一组键值对形式的指标，具体内容由各产品上下文定义。

- `MetricEntryDTO`
  - `Key`：字符串，指标名称（例如 `cursor.instances.count`, `cursor.skills.count`, `openclaw.subagents.count`）。
  - `Value`：字符串，指标值（尽量使用可直接展示的格式，如 `"3"`, `"75%"`, `"120ms"`）。
  - `Description`（可选）：字符串，对该指标含义的简短解释，便于 UI 直接渲染为说明文本。

## 5. 与现有/规划代码的映射（当前设计）

- `internal/agent/provides.go`（规划中）
  - 将提供 `CapabilityCatalogQueryService` 及 `AgentDirectory` 等接口的具体实现入口，包含 `ListAgents` 与 `DescribeAgent` 方法。
- `internal/runtime` / `internal/daemon`（规划中）
  - 负责在 `BootstrapEnvironmentUseCase` 中维护聚合的 Agent 目录视图，并为 `AgentDirectory` 提供实现。
- 各产品上下文中的 metrics 适配层（如 `internal/cursorsmith/...`、`internal/openclaw/...`，命名待定）
  - 提供 `ProductMetricsProvider` 的具体实现，用于按产品收集 agent 相关统计数据。
- `gateway` / `channels`（规划中）
  - 暴露 `GET /agents` 与 `GET /agents/{id}` 等 HTTP/IPC 接口，分别调用 `CapabilityCatalogQueryService.ListAgents` 与 `CapabilityCatalogQueryService.DescribeAgent`，入参与出参均使用本节约定的 DTO。

> 约束：本轮仅为 `ListAgentsUseCase` 与 `DescribeAgentUseCase` 明确应用层编排与接口形状及 DTO 形状，`ProductSpecificMetricsDTO` 的具体指标键名与业务含义在各产品上下文内演进，但外层结构保持稳定。
