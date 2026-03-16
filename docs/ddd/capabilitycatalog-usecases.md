## CapabilityCatalogContext 用例：查看 Agent 能力目录

### ListAgentsUseCase — 主成功场景（用户查看所有 agent）

1. 用户在 CLI 或 TUI 中选择「查看所有可用 agent」入口，发起查看请求，可以选择性地输入简单过滤条件（如按标签、分组或名称关键字筛选）。
2. 前端或 CLI 将用户输入封装为 `ListAgentsQuery`（当前可为空或只包含标签/分组/关键字），调用后端接口（例如 `GET /agents` 或等价的 IPC 调用），将请求转交给能力目录上下文对应的查询服务。
3. 查询服务调用 `AgentDirectory.ListAll` 或等价端口，从当前的 Agent 目录视图（由 RuntimeContext/daemon 在 `BootstrapEnvironment` 等用例中维护的缓存或仓储）中读取所有可用 agent 的基础信息。
4. 查询服务在内存中根据 `ListAgentsQuery` 进行过滤与排序（例如先按是否匹配标签/分组，再按 agent 名称或优先级排序），将结果映射为 agent 列表 DTO，包含每个 agent 的标识、名称、简要说明及关键能力标签。
5. CLI 或 TUI 根据返回的 agent 列表，在终端或界面上以列表形式展示所有可用 agent，支持用户使用光标/快捷键/鼠标选择某个 agent 查看详情或作为后续派发目标。

> TODO: 后续补充失败与异常场景（如目录视图尚未初始化、后端不可用等）。

### DescribeAgentUseCase — 主成功场景（用户查看单个 agent 详情）

1. 用户在 CLI 或 TUI 的 agent 列表界面中，通过光标/快捷键/鼠标选择某个感兴趣的 agent，或在命令行中直接输入 agent 标识，发起「查看详情」请求。
2. 前端或 CLI 将用户选择封装为 `DescribeAgentQuery`，其中至少包含 agent 标识（ID/名称），可选包含当前产品标识（如 `product=cursorsmith`、`product=openclaw` 等），并调用后端接口（例如 `GET /agents/{id}` 或等价的 IPC 调用）。
3. 能力目录上下文的查询服务调用 `AgentDirectory.GetByID` 或等价端口，从当前 Agent 目录视图中读取该 agent 的基础元信息（标识、名称、版本、简要说明）以及该 agent 所暴露的能力列表（每个能力的名称、用途简述、输入输出概要）。
4. 查询服务在内存中根据 agent 在不同产品中的绑定关系，组装与产品无关的通用字段（基础信息、能力列表、支持的调用入口、是否具有写文件/执行命令等危险能力），并调用产品特定的 metrics 端口（如 `ProductMetricsProvider.CollectForAgent`）获取当前产品下与该 agent 相关的扩展指标（例如 Cursor 中的实例数、已注册的 skill 数，或 OpenClaw 中的 subagent 数量、常见编排深度等）。
5. 查询服务将通用字段与产品特定指标合并映射为 `AgentDetailDTO`，其中包含：agent 基础信息、能力与接口视图、运行约束与风险等级、产品特定指标（以结构化字段或键值对形式承载），并将该 DTO 返回给前端或 CLI。
6. CLI 或 TUI 将 `AgentDetailDTO` 渲染为分区展示的详情视图：上半部分显示通用信息（ID/名称/版本、简介、能力列表、危险操作提示），下半部分按当前产品以模块化方式展示扩展指标（如「Cursor 统计」「OpenClaw 统计」小节），并在适当位置展示 1～3 条推荐使用场景示例或快捷操作入口（例如「以此 agent 作为后续任务的默认执行者」）。

> TODO: 后续补充该用例下的异常场景（如指定 agent 不存在、产品特定 metrics 提供方不可用等），以及产品特定指标字段的最小统一模型。

