## RuntimeContext 用例：守护进程与环境自检

### BootstrapEnvironmentUseCase：daemon 启动与环境扫描

- **触发者**: 系统（pathfinder daemon 进程启动）
- **目标**: 在不修改任何外部系统的前提下，基于当前环境（openclaw、本地 workspace 等）构建一份「Agent/Skill/Tool 目录视图」，并在运行期保持与底层文件/registry 变更大致同步。

**主成功场景（Happy Path）**

1. daemon 进程启动，加载 `Config`，解析出 `WorkspaceDir` 与 provider 等运行参数。
2. daemon 基于当前环境构造端口实现：
   1. 若检测到 openclaw、Cursor、OpenCode、ClaudeCode 等后端已安装且可用（对应配置目录存在、所需 Socket/API 可连接），则为每种后端分别构造 `XxxAgentDiscovery`、`XxxSkillRepository`、`XxxToolRepository` 等实现，并组合成统一视图；
   2. 若某类后端不可用，则仅使用其余可用后端与基于本地文件系统的 `FilesystemAgentDiscovery`（如有），必要时回退到内存占位实现。
3. daemon 调用 `BootstrapEnvironment`：
   1. 分别调用各已启用后端的 `XxxAgentDiscovery.ListAgents`（如 OpenClaw、Cursor、OpenCode、ClaudeCode、本地文件系统等），将返回的 Agent 列表在内存中合并，构建/更新本地「Agent 目录视图」；
   2. 分别调用各已启用后端的 `XxxSkillRepository.List(ctx, workspaceDir)` 扫描其各自的技能目录（包括 workspace），将结果在内存中合并为聚合的「Skill 目录视图」；
   3. 分别调用各已启用后端的 `XxxToolRepository.List(ctx, workspaceDir)` 扫描其各自的工具目录（包括 workspace），将结果在内存中合并为聚合的「Tool 目录视图」。
4. 若上述扫描成功完成，daemon 将扫描结果缓存在内存（必要时写入持久化），并记录一条日志摘要当前环境（各后端启用情况以及 Agent/Skill/Tool 数量）。
5. daemon 注册文件/目录变更监听：
   1. 调用 `FileChangeWatcher.Watch(ctx, paths, onChange)`，`paths` 至少包含 workspace 的 skills/tools 目录与各后端对应的 agent/skill/tool 目录；
   2. `onChange` 回调收到文件变更事件后，根据事件类型判断是 Agent/Skill/Tool 目录发生变更；
   3. 针对变更的目录类型触发最小必要的增量重扫（如仅重扫对应的 skill 目录），更新本地目录视图。
6. 若监听注册成功，daemon 进入 HTTP 事件循环，对外暴露 job 相关 API（创建、查询、取消），后续所有规划与派发逻辑都只依赖上述目录视图与对应端口。

**失败与降级场景（只列出占位，细节后续补充）**

- F1：某个目录扫描失败 → 仅该类型视图为空或部分可用，同时打日志；不阻塞 daemon 启动。
- F2：文件变更监听注册失败 → 依然完成一次性 Bootstrap，放弃增量重扫，仅定期全量重扫或由后续用例决定。

---

### StartJobUseCase / ContinueJobUseCase：基于 daemon 的异步运行

> 这两个用例复用现有 `StartJob` / `ContinueJob` 语义，在 daemon 模式下视为 RuntimeContext 的标准用例。

- **触发者**: 外部调用者（HTTP 客户端 / 上游系统）
- **目标**: 在已经完成环境 Bootstrap 的前提下，为每个新目标创建 job，后台异步执行，并对外提供查询与取消。

**StartJobUseCase 主成功场景（简化版）**

1. 调用方通过 HTTP 向 daemon 提交目标描述与可选超时参数。
2. daemon 解析请求后构造 `SubmitGoalCommand`，并调用 Runtime 应用服务的 `StartJob(ctx, cmd)`。
3. `StartJob` 调用 Planner 产出 Plan，校验并持久化 Plan，然后创建 `Job`（状态为 planning），写入 JobRepository。
4. `StartJob` 返回 `jobId`（以及 `streamHandle`），daemon 立即将 `jobId` 与查询 URL 返回给调用方。

**ContinueJobUseCase 主成功场景（简化版）**

1. daemon 在 `StartJob` 返回后于后台启动 `ContinueJob(ctx, jobId)`。
2. `ContinueJob` 读取 Job 与对应 Plan，为每个 SubTask 初始化 `TaskProgress`，并将 Job 状态更新为 running。
3. `ContinueJob` 调用 `ExecutePlan(ctx, jobId)`，在循环中根据 Plan、Agent 目录视图与 TaskProgress 状态，持续派发子任务到 Agent 并写回结果。
4. 执行结束或被取消/超时后，`ContinueJob` 根据 Job 的取消标志与截止时间，将 Job 状态更新为 completed 或 aborted，并保存。

（完整 RuntimeContext 应用层设计见 `docs/ddd/application-layer-spec-runtime.md`。）

---

### WatchJobProgressUseCase — 主成功场景（用户实时查看任务执行与进度）

1. 用户在 CLI、TUI 或 Web 界面中选择某个已创建的 job，希望实时观看该 job 的执行过程与进度变化。
2. 前端或 CLI 按约定的方式发起流式订阅请求，例如：
- 通过 HTTP SSE/WebSocket 连接到 daemon 暴露的流式接口（如 `GET /jobs/{jobId}/stream`）；
- 或在 CLI 中使用 `pathfinder run --watch {jobId}`，由 CLI 内部与 daemon 建立长连接。
3. daemon 收到订阅请求后，校验 `jobId` 是否存在，并为该连接注册进度事件监听，包括 Job 状态变更与 TaskProgress 更新事件。
4. 当执行引擎按计划派发子任务、写回 TaskProgress、更新 Job 状态时，RuntimeContext 将相应的进度事件推送到该订阅连接：
   - 包含当前阶段（规划/执行/总结）、已完成与待执行的子任务数量；
   - 最近一个子任务的执行结果摘要（如所属 agent、简要输出）；
   - 当前 Job 的 整体状态（queued/running/completed/aborted 等）。
5. 前端或 CLI 持续消费这些流式事件，并以人类可读的方式渲染出来：
   - TUI/终端中滚动显示最新的任务步骤与 agent 执行日志；
   - Web 界面中更新进度条、任务列表与当前阶段标签。
6. 当 Job 执行完成、被取消或因条件中止时，RuntimeContext 发送最终状态事件并关闭该订阅连接，前端或 CLI 将该 job 标记为结束状态并停止继续拉取。

> TODO: 后续补充异常场景（如 `runId` 不存在、连接中断、订阅超时等）和与内部执行状态维护逻辑的交互细节。

