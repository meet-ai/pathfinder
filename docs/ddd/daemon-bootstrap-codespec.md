## DaemonBootstrap — Codespec

> 聚焦：pathfinder daemon 进程启动与自检流程（BootstrapEnvironment），不展开内部 Job 执行细节。

## 1. 参与者与职责

- **操作系统 / 进程管理器**
  - 触发 daemon 进程启动（如 `pathfinder-daemon` 可执行文件，或由服务管理器拉起）。
  - 负责向进程发送停止信号（SIGINT/SIGTERM），本轮仅在文档中标注，不展开优雅退出实现。

- **daemon 入口（`cmd/pathfinder-daemon/main.go`）**
  - 负责：
    - 加载配置与 workspace 目录（`config.Load`）。
    - 组装各领域 Port 的具体实现（JobRepository、PlanRepository、TaskProgressRepository、Planner、AgentDiscovery、Dispatcher 等）。
    - 调用 RuntimeContext 暴露的 Bootstrap 能力（当前由 `BootstrapEnvironmentUseCase` 表达）。
    - 注册 HTTP 路由 `/jobs`、`/jobs/{jobId}`、`/jobs/{jobId}/cancel`。
    - 启动 HTTP 服务器，进入主事件循环。

- **RuntimeContext / 相关 Context（通过应用服务间接参与）**
  - `JobRepository`（RuntimeContext）：持久化 Job。
  - `PlanRepository`（PlanningContext）：持久化 Plan。
  - `TaskProgressRepository`（ExecutionStateContext）：持久化任务进度。
  - `AgentDiscovery`、`SkillRepository`、`ToolRepository`（CapabilityCatalogContext + 各后端适配器）。
  - `AgentDirectory`（CapabilityCatalogContext）：维护聚合后的 Agent 目录视图，供能力目录查询使用。
  - `CapabilityCatalogQueryService`（CapabilityCatalogContext）：对外提供 `ListAgents` / `DescribeAgent` 查询能力。
  - `Dispatcher`（RuntimeContext）：派发子任务到 agent。
  - `FileChangeWatcher`（Infra Port，规划中）：监听 agent/skill/tool 目录变更，触发增量重扫。

## 2. DaemonBootstrapUseCase — 编排步骤表（进程启动视角）

| 步骤 | 业务步骤（来自 `BootstrapEnvironmentUseCase` 等用例） | daemon 层动作（`main.go`） | 调用的应用服务方法 / Port | 触达的领域服务 / Port |
| ---- | ---------------------------------------------------- | -------------------------- | -------------------------- | ---------------------- |
| 1 | 进程启动，daemon 需要解析配置与 workspace 目录。 | 调用 `config.Load()`，获得 `Config`（含 `WorkspaceDir`、provider 默认值等），若失败则记录错误并 `os.Exit(1)`。 | `config.Load()` | — |
| 2 | daemon 基于当前环境构造领域 Port 的具体实现。 | 创建内存或实际实现：`JobRepoMem`、`PlanRepoMem`、`TaskProgressRepoMem`、`PlannerStub`、`AgentDiscovery`（环境感知实现）、`AgentDirectory`（如 `AgentDirectoryMem`）、`ProductMetricsProvider`（如 `ProductMetricsStub`）、`Dispatcher`（LLM 或 stub）。 | （构造函数） | `JobRepository`、`PlanRepository`、`TaskProgressRepository`、`Planner`、`AgentDiscovery`、`AgentDirectory`、`ProductMetricsProvider`、`Dispatcher` |
| 3 | 组装 RuntimeContext 与 CapabilityCatalogContext 的应用服务实例。 | 使用上一步构造的 Port 实现：new `WorkflowOrchestrationApplicationService{Planner, JobRepo, PlanRepo, TaskProgress, AgentDiscovery, Dispatcher}`；同时 new `DefaultCapabilityCatalogQueryService(AgentDirectory, ProductMetricsProvider)`，并将其注入到后续需要查询能力目录的入口层或上游 Context。 | `WorkflowOrchestrationApplicationService`、`DefaultCapabilityCatalogQueryService` 构造函数 | 上述各 Port |
| 4 | （预留）触发 RuntimeContext 的环境自检与扫描。 | 当前版本中，BootstrapEnvironment 逻辑由 `BootstrapEnvironmentUseCase` 文档约束，实际调用点可在此处或应用服务构造后添加：如 `svc.BootstrapEnvironment(ctx)`，完成 agent/skill/tool 目录视图构建与 `FileChangeWatcher` 注册；Bootstrap 完成后由 RuntimeContext 将聚合的 Agent 列表写入 `AgentDirectory` 视图。 |（未来）`BootstrapEnvironment(ctx)` | `AgentDiscovery`、`SkillRepository`、`ToolRepository`、`FileChangeWatcher`、`AgentDirectory` |
| 5 | 注册 HTTP 路由，将 URL 映射到 Job 管理用例。 | 创建 `http.NewServeMux()`，注册：`POST /jobs` → `handleCreateJob`；`GET /jobs/{jobId}` → `handleGetJob`；`POST /jobs/{jobId}/cancel` → `handleCancelJob`。 |（入口层内部逻辑）| `StartJob`、`ContinueJob`、`GetJobState`、`CancelJob`（通过 `server.svc`）|
| 6 | 监听端口并启动 HTTP 服务器，进入事件循环。 | 根据 `PATHFINDER_DAEMON_ADDR`（默认 `:8080`）创建 `net.Listen` 与 `http.Server`，记录 “listening” 日志后调用 `srv.Serve(ln)`；失败时记录错误并 `os.Exit(1)`。 | 标准库 `http.Server` | — |
| 7 | （预留）处理进程退出与优雅关停。 | 当前版本仅在 `Serve` 返回非 `ErrServerClosed` 时退出；后续可扩展：捕获 OS 信号，停止接受新请求，等待后台 `ContinueJob` 完成或标记为中止。 |（未来）`Server.Shutdown` | `JobRepository` 等（用于标记 Job 终态） |

## 3. 与现有代码的映射

- `cmd/pathfinder-daemon/main.go`
  - `config.Load()`：实现步骤 1。
  - `NewJobRepoMem` / `NewPlanRepoMem` / `NewTaskProgressRepoMem` / `PlannerStub` / `NewAgentDiscoveryMem` / `DispatcherLLM` / `DispatcherStub`：实现步骤 2。
  - `WorkflowOrchestrationApplicationService{...}`：实现步骤 3。
  - `http.NewServeMux` + `HandleFunc("/jobs", ...)` + `HandleFunc("/jobs/", ...)`：实现步骤 5。
  - `net.Listen` + `http.Server{Addr, Handler}` + `srv.Serve(ln)`：实现步骤 6。

- `docs/ddd/daemon-runtime-usecases.md`
  - `BootstrapEnvironmentUseCase`：约束步骤 4 中的环境扫描与 `FileChangeWatcher` 注册行为。
  - `StartJobUseCase / ContinueJobUseCase / WatchJobProgressUseCase`：通过应用服务方法（`StartJob` / `ContinueJob` / `GetJobState`）被 daemon 路由层间接调用。

> 约束：本 codespec 仅描述 daemon 进程级 bootstrap 与 HTTP 路由层的职责边界，不在此文件中展开 Job 执行循环、进度维护与能力发现等领域内部细节，这些由 RuntimeContext / ExecutionStateContext / CapabilityCatalogContext 各自的 codespec 与 usecases 负责定义。

---

## 4. API 网关视角（HTTP 层职责）

> 这里的「API 网关」特指 daemon 内嵌的 HTTP 层，不包含外部反向代理或统一网关组件。

### 4.1 API 列表与语义

| API                     | 方法 | 描述                              | 对应用例                         |
|-------------------------|------|-----------------------------------|----------------------------------|
| `/jobs`                 | POST | 提交目标，创建并启动新的 job      | `SubmitGoalUseCase` + `StartJob` |
| `/jobs/{jobId}`         | GET  | 查询指定 job 的当前状态与进度     | `GetJobStateUseCase`             |
| `/jobs/{jobId}/cancel`  | POST | 取消指定 job（请求中止后续执行） | `CancelJobUseCase`               |

### 4.2 网关层编排（简化步骤表）

| API                    | 步骤 | 网关层动作                          | 调用的应用服务方法              |
|------------------------|------|-------------------------------------|---------------------------------|
| `POST /jobs`           | 1    | 解码 JSON `{goal, timeoutSec}`      | —                               |
|                        | 2    | 构造 `SubmitGoalCommand`            | —                               |
|                        | 3    | 调用 `StartJob(ctx, cmd)`           | `WorkflowOrchestrationApplicationService.StartJob` |
|                        | 4    | 在 goroutine 中调用 `ContinueJob`   | `WorkflowOrchestrationApplicationService.ContinueJob` |
|                        | 5    | 根据 `jobId` 和 `Host` 构造状态 URL | —                               |
|                        | 6    | 返回 `{jobId, url}` JSON            | —                               |
| `GET /jobs/{jobId}`    | 1    | 从 URL 提取 `jobId`                 | —                               |
|                        | 2    | 调用 `GetJobState(ctx, jobId)`      | `WorkflowOrchestrationApplicationService.GetJobState` |
|                        | 3    | 将 `JobStateDTO` 序列化为 JSON 返回 | —                               |
| `POST /jobs/{jobId}/cancel` | 1 | 从 URL 提取 `jobId`                 | —                               |
|                        | 2    | 调用 `CancelJob(ctx, jobId)`        | `WorkflowOrchestrationApplicationService.CancelJob` |
|                        | 3    | 返回 204 No Content                 | —                               |

### 4.3 职责边界

- **网关层负责**：
  - HTTP 协议解析与路由（URL、方法、状态码、Header）。
  - JSON 编解码与基础校验（参数缺失/类型错误 → 4xx）。
  - 将请求映射为应用层命令（`SubmitGoalCommand`）或标识（`jobId`），并调用对应应用服务方法。
  - 将应用层返回的 DTO 直接序列化为 JSON 回给客户端。

- **网关层不负责**：
  - 不直接操作 Job/Plan/TaskProgress 结构体字段；
  - 不实现任何业务规则（如调度顺序、重试策略、超时判定）；
  - 不维护 Job 状态机，只通过应用服务/领域服务完成状态流转。

---

## 5. Happy Path 验证方式（本轮已打通）

### 5.1 启动 daemon

```bash
go run ./cmd/pathfinder-daemon/main.go
# 预期日志：pathfinder daemon listening addr=[::]:8080
```

### 5.2 创建 Job

```bash
curl -s -X POST http://127.0.0.1:8080/jobs \
  -H 'Content-Type: application/json' \
  -d '{"goal":"test job","timeoutSec":60}'
# 预期返回：{"jobId":"...","url":"http://127.0.0.1:8080/jobs/..."}
```

### 5.3 查看所有 agent

```bash
curl -s http://127.0.0.1:8080/agents
# 预期返回：[{"ID":"agent-1","Name":"stub",...}]
```

### 5.4 查看 Job 状态

```bash
curl -s http://127.0.0.1:8080/jobs/{jobId}
# 预期返回：JobStateDTO JSON（含 status、tasks 等）
```

### 5.5 实时观看 Job 进度（SSE stub）

```bash
curl -N http://127.0.0.1:8080/jobs/{jobId}/watch
# 预期返回：至少一条 data: {...} SSE 事件（JobProgressEvent JSON）
```

### 5.6 取消 Job

```bash
curl -s -X POST http://127.0.0.1:8080/jobs/{jobId}/cancel
# 预期返回：204 No Content
```

> 本轮验证状态：5.2（创建 Job）与 5.3（查看 agent）已在本机确认通过；5.4/5.5/5.6 接口已接好，待后续迭代补充更完整的端到端验证。

