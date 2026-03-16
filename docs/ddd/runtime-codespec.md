## RuntimeContext — Codespec

> 当前聚焦：`WatchJobProgressUseCase`（用户实时查看某个 Job 的执行过程与进度）

## 1. 参与者与接口

- **调用方（上游 CLI / TUI / Web / JobManagementContext）**
  - 调用点：
    - 在 Web/TUI 界面的 Job 列表中，用户选中某个 Job 后进入「实时查看执行过程」视图；
    - （可选）在 CLI 中执行 `pathfinder run --watch {jobId}`，效果等价于从列表中选中该 Job 后进入实时视图。
  - 职责：基于一个已知的 `jobId` 建立长连接，持续接收并展示进度事件。

- **入口层（Daemon HTTP / SSE / WebSocket）**
  - HTTP SSE 示例：`GET /jobs/{jobId}/watch` 或 `GET /jobs/{jobId}/stream`。
  - 职责：校验请求参数，建立到 Runtime 应用服务的订阅通道，将应用层产生的进度事件编码为 SSE/WebSocket 帧推送给调用方。

- **应用服务（RuntimeQueryService，名称暂定）**
  - `WatchJobProgress(ctx context.Context, jobId string, consumer JobProgressEventConsumer) error`
    - 说明：`consumer` 为推送回调接口，由入口层实现，用于将事件写入网络连接。

- **参与的领域服务 / Port**
  - `JobRepository`（RuntimeContext）：读取 Job 当前状态（queued/running/completed/aborted 等）。
  - `TaskProgressRepository`（ExecutionStateContext）：按 Job 维度读取子任务进度。
  - （可选）`ProgressStream`：若后续需要从执行循环直接订阅进度事件，可抽象为内部事件流端口，本轮先不强行落地。

## 2. WatchJobProgressUseCase — 编排步骤表（应用层视角）

| 步骤 | 业务步骤（来自用例） | 应用层动作 | 调用的应用服务方法 / Port | 触达的领域服务 / Port |
| ---- | -------------------- | ---------- | -------------------------- | ---------------------- |
| 1 | 用户在 CLI/TUI/Web 中基于一个已知的 `jobId` 选择「实时查看该 job 执行过程」。 | 入口层从 URL/命令行中解析出 `jobId`，构造订阅请求。 | （入口层内部逻辑） | — |
| 2 | 前端或 CLI 建立到 daemon 的长连接（SSE/WebSocket 或等价机制）。 | 入口层创建 `JobProgressEventConsumer` 实例，将其与底层连接绑定。 | `WatchJobProgress(ctx, jobId, consumer)` | — |
| 3 | daemon 收到订阅请求后，校验 `jobId` 是否存在且属于当前 workspace。 | Runtime 应用服务首先通过 `JobRepository` 读取 Job，若不存在则返回错误，由入口层关闭连接并告知调用方。 | `WatchJobProgress(ctx, jobId, consumer)` | `JobRepository` |
| 4 | 订阅建立成功后，Runtime 应用服务进入循环或回调模式，按时间顺序生成 Job 进度事件。 | 每次循环/回调中，从 `JobRepository` 与 `TaskProgressRepository` 读取 Job 当前状态与子任务进度，组装为 `JobProgressEvent`。 | `WatchJobProgress(ctx, jobId, consumer)` | `JobRepository`、`TaskProgressRepository` |
| 5 | 每个 `JobProgressEvent` 至少包含当前阶段、已完成/待执行子任务数量、最近一个子任务的执行摘要与 Job 整体状态。 | 应用服务调用 `consumer.Push(event)` 将事件推送给入口层，由入口层编码为 SSE/WebSocket 帧写回给调用方。 | `WatchJobProgress(ctx, jobId, consumer)` | —（纯内存转换） |
| 6 | 用户端持续消费这些事件，在界面上实时渲染执行过程。 | 入口层保持连接存活直到应用服务返回或调用方主动断开。 | （入口层内部逻辑） | — |
| 7 | 当 Job 执行完成、被取消或因条件中止时，Runtime 应用服务发送带有终态状态的最终事件并结束订阅。 | 应用服务在检测到 Job 状态为终态后，发送最后一个 `JobProgressEvent`，然后从 `WatchJobProgress` 返回；入口层关闭连接。 | `WatchJobProgress(ctx, jobId, consumer)` | `JobRepository`、`TaskProgressRepository` |

## 3. 与现有/规划代码的映射（当前设计）

- `docs/ddd/daemon-runtime-usecases.md`
  - 已定义 `WatchJobProgressUseCase` 的主成功场景，描述了 CLI/HTTP 层交互与用户视角。
- `internal/orchestration/service.go`（现有）
  - 当前已包含 Job 执行与进度更新的主循环；后续可在此基础上提炼出用于查询/订阅的进度读取逻辑。
- `internal/progress`（ExecutionStateContext，规划中）
  - 将提供 `TaskProgressRepository` 与可能的 `JobProgress` 汇总接口。
- `internal/runtime`（RuntimeContext，规划中）
  - 将提供 `JobRepository` 与 `RuntimeQueryService` 的具体实现入口。

> 约束：本轮仅为 `WatchJobProgressUseCase` 明确应用层编排与接口形状，不在本文件中决定具体的事件编码格式（SSE/JSON schema）与内部事件源实现（轮询 vs 内部事件流）；这些细节由后续设计在 gateway/runtime 包内收敛。

