## RuntimeContext 用例：守护进程与环境自检

### BootstrapEnvironmentUseCase：daemon 启动与环境扫描

- **触发者**: 系统（pathfinder daemon 进程启动）
- **目标**: 在不修改任何外部系统的前提下，基于当前环境（openclaw、本地 workspace 等）构建一份「Agent/Skill/Tool 目录视图」，并在运行期保持与底层文件/registry 变更大致同步。

**主成功场景（Happy Path）**

1. daemon 进程启动，加载 `Config`，解析出 `WorkspaceDir` 与 provider 等运行参数。
2. daemon 基于当前环境构造端口实现：
   1. 若检测到 openclaw、Cursor、OpenCode、ClaudeCode 等后端已安装且可用（对应配置目录存在、所需 Socket/API 可连接），则为每种后端分别构造 `XxxAgentDiscovery`、`XxxSkillRepository`、`XxxToolRepository` 等实现，并组合成统一视图；
   2. 若某类后端不可用，则仅使用其余可用后端与基于本地文件系统的 `FilesystemAgentDiscovery`（如有），不再为该后端构造占位实现。
3. daemon 调用 `BootstrapEnvironment`：
   1. 通过组合后的 `AgentDiscovery.ListAgents` 读取当前环境（openclaw、Cursor、OpenCode、ClaudeCode、本地等）可用的全部 Agent，构建/更新本地「Agent 目录视图」；
   2. 通过组合后的 `SkillRepository.List(ctx, workspaceDir)` 扫描来自各后端与 workspace 的技能目录，构建/更新聚合的「Skill 目录视图」；
   3. 通过组合后的 `ToolRepository.List(ctx, workspaceDir)` 扫描来自各后端与 workspace 的工具目录，构建/更新聚合的「Tool 目录视图」。
4. 若上述扫描成功完成，daemon 将扫描结果缓存在内存（必要时写入持久化），并记录一条日志摘要当前环境（各后端启用情况以及 Agent/Skill/Tool 数量）。
5. daemon 注册文件/目录变更监听：
   1. 调用 `FileChangeWatcher.Watch(ctx, paths, onChange)`，`paths` 至少包含 workspace 的 skills/tools 目录与各后端对应的 agent/skill/tool 目录；
   2. `onChange` 回调收到文件变更事件后，根据事件类型判断是 Agent/Skill/Tool 目录发生变更；
   3. 针对变更的目录类型触发最小必要的增量重扫（如仅重扫对应的 skill 目录），更新本地目录视图。
6. 若监听注册成功，daemon 进入 HTTP 事件循环，对外暴露 run 相关 API（创建、查询、取消），后续所有规划与派发逻辑都只依赖上述目录视图与对应端口。

**失败与降级场景（只列出占位，细节后续补充）**

- F1：某个目录扫描失败 → 仅该类型视图为空或部分可用，同时打日志；不阻塞 daemon 启动。
- F2：文件变更监听注册失败 → 依然完成一次性 Bootstrap，放弃增量重扫，仅定期全量重扫或由后续用例决定。

---

### StartJobUseCase / ContinueJobUseCase：基于 daemon 的异步运行

> 这两个用例复用现有 `StartJob` / `ContinueJob` 语义，在 daemon 模式下视为 RuntimeContext 的标准用例。

- **触发者**: 外部调用者（HTTP 客户端 / 上游系统）
- **目标**: 在已经完成环境 Bootstrap 的前提下，为每个新目标创建 run，后台异步执行，并对外提供查询与取消。

**StartJobUseCase 主成功场景（简化版）**

1. 调用方通过 HTTP 向 daemon 提交目标描述与可选超时参数。
2. daemon 解析请求后构造 `SubmitGoalCommand`，并调用 Runtime 应用服务的 `StartJob(ctx, cmd)`。
3. `StartJob` 调用 Planner 产出 Plan，校验并持久化 Plan，然后创建 `Job`（状态为 planning），写入 JobRepository。
4. `StartJob` 返回 `jobId`（以及 `streamHandle`），daemon 立即将 `jobId` 与查询 URL 返回给调用方。

**ContinueJobUseCase 主成功场景（简化版）**

1. daemon 在 `StartJob` 返回后于后台启动 `ContinueJob(ctx, jobId)`。
2. `ContinueJob` 读取 Job 与对应 Plan，根据当前 TaskProgress 集合决定下一批需要执行的 SubTask。
3. 在为每个即将执行的 SubTask 派发请求前，`ContinueJob` 为该 SubTask 创建或更新对应的 `TaskProgress` 记录，使其状态与即将执行的动作一致。
4. `ContinueJob` 调用 `ExecutePlan(ctx, jobId)`，在循环中根据 Plan、Agent 目录视图与 TaskProgress 状态，持续派发子任务到 Agent 并写回结果。
5. 执行结束或被取消/超时后，`ContinueJob` 根据 Job 的取消标志与截止时间，将 Job 状态更新为 completed 或 aborted，并保存。

（完整 RuntimeContext 应用层设计见 `docs/ddd/application-layer-spec-runtime.md`。）

---

### WatchJobProgressUseCase — 主成功场景（用户实时查看任务执行与进度）

1. 用户在 CLI、TUI 或 Web 界面中，基于一个已知的 `jobId`，选择「实时查看该 job 执行过程」入口（例如在 job 列表中选中某行后按快捷键或点击“查看执行过程”）。
2. 前端或 CLI 按约定的方式发起流式订阅请求，例如：
   - 通过 HTTP SSE/WebSocket 连接到 daemon 暴露的流式接口（如 `GET /jobs/{jobId}/stream`）；
   - 或在 CLI 中使用 `pathfinder run --watch {jobId}`，由 CLI 内部与 daemon 建立长连接。
3. daemon 收到订阅请求后，校验 `jobId` 是否存在且属于当前 workspace，并为该连接注册进度事件监听，包括 Job 状态变更与 TaskProgress 更新事件。
4. 当执行引擎按计划派发子任务、写回 TaskProgress、更新 Job 状态时，RuntimeContext 按时间顺序将相应的进度事件推送到该订阅连接：
   - 每个事件至少包含当前阶段（规划/执行/总结）、已完成与待执行的子任务数量；
   - 最近一个子任务的执行结果摘要（如所属 agent、子任务描述、简要输出或状态）；
   - 当前 Job 的整体状态（queued/running/completed/aborted 等），以及必要时的提示信息（如“用户取消”“超时中止”）。
5. 前端或 CLI 持续消费这些流式事件，并以人类可读的方式渲染出来：
   - TUI/终端中按时间轴滚动显示最新的任务步骤与 agent 执行日志，同时更新当前阶段与 Job 状态；
   - Web 界面中更新进度条、任务列表与当前阶段标签，并在需要时高亮当前正在执行的子任务。
6. 当 Job 执行完成、被取消或因条件中止时，RuntimeContext 发送带有终态状态的最终事件并关闭该订阅连接，前端或 CLI 将该 job 标记为结束状态（例如“已完成”或“已中止”），并停止继续拉取，同时在界面上保留最终执行结果与进度概览，方便用户事后查看。

> TODO: 后续补充异常场景（如 `runId` 不存在、连接中断、订阅超时等）和与 `ExecutionStateContext` 的交互细节。

