## JobManagementContext — Codespec

> 当前聚焦：`SubmitGoalUseCase`（带 daemon 的主成功场景）与 `CancelJobUseCase`

## 1. 参与者与接口

- **入口层（CLI）**
  - 命令：`pathfinder -m "目标描述"`
  - 职责：读取命令行参数、调用 daemon HTTP 接口、打印返回的 URL。
- **入口层（Daemon HTTP）**
  - `POST /jobs`：提交目标，触发 `SubmitGoalUseCase`（异步执行）。
  - `GET /jobs/{jobId}`：查询 job 状态（包装 `GetJobState`）。
- **应用服务**
  - `WorkflowOrchestrationApplicationService.StartJob(ctx, SubmitGoalCommand) (jobId string, err error)`
  - `WorkflowOrchestrationApplicationService.ContinueJob(ctx, jobId string)`
  - `WorkflowOrchestrationApplicationService.GetJobState(ctx, jobId string) (*JobStateDTO, error)`
- **Command / DTO**
  - `SubmitGoalCommand`：`GoalDescription`、`TimeoutSecs`（可选）。
  - `JobStateDTO`：`JobId`、`Status`、`Completed`、`Total`、`Tasks[]`、可选 `Plan`、`Summary`。
- **参与的领域服务 / Port**
  - `Planner`（规划器）：`PlanGoal(ctx, GoalDescription) -> Plan`
  - `JobRepository`：`Save/Get` Job。
  - `PlanRepository`：`Save/Get` Plan。
  - `TaskProgressRepository`：`Save/ListByJobId` TaskProgress。
  - `Dispatcher`、`AgentDiscovery`：由 `agent.Loop` 使用。

## 2. SubmitGoalUseCase — 编排步骤表（应用层视角）

| 步骤 | 业务步骤（来自用例） | 入口/App 层动作 | 调用的应用服务方法 | 触达的领域服务 / Port |
| ---- | -------------------- | --------------- | ------------------ | ---------------------- |
| 1 | 用户在终端执行 `pathfinder -m "目标描述"` | CLI 解析 `-m` 参数，构造 `{goal: "..."} JSON` 请求 | （无，CLI 仅为入口） | — |
| 2 | CLI 通过 HTTP 调用 daemon `POST /jobs` | CLI 读取 `PATHFINDER_DAEMON_URL`（默认 `http://127.0.0.1:8080`），向 `/jobs` 发送 JSON | （无，CLI 仅为入口） | — |
| 3 | daemon 接收请求、触发 SubmitGoal 用例 | HTTP handler 解码 JSON 为 `goal` / `timeoutSec`，构造 `SubmitGoalCommand` | `StartJob(ctx, SubmitGoalCommand)` | `Planner`, `PlanRepository`, `JobRepository` |
| 4 | daemon 在后台执行计划 | `StartJob` 返回 `jobId` 后，daemon 在 goroutine 中调用 | `ContinueJob(ctx, jobId)` | `JobRepository`, `PlanRepository`, `TaskProgressRepository`, `Dispatcher`, `AgentDiscovery`（经 `agent.Loop`） |
| 5 | daemon 生成浏览器可访问的状态 URL | HTTP handler 根据 `jobId` 与 `Host` 构造 `http://{host}/jobs/{jobId}`，打包 JSON 响应 `{jobId, url}` | （封装在 handler 内部） | — |
| 6 | CLI 打印 URL 给用户 | CLI 解析响应 JSON，取出 `url`，直接打印到终端 | （无，CLI 仅为入口） | — |
| 7 | 用户在浏览器中查看 JSON 状态 | 浏览器访问 `GET /jobs/{jobId}`，daemon 调用应用服务查询 | `GetJobState(ctx, jobId)` | `JobRepository`, `TaskProgressRepository`, 可选 `PlanRepository` |

## 3. CancelJobUseCase — 编排步骤表（应用层视角）

| 步骤 | 业务步骤（来自用例） | 入口/App 层动作 | 调用的应用服务方法 | 触达的领域服务 / Port |
| ---- | -------------------- | --------------- | ------------------ | ---------------------- |
| 1 | 用户在任务状态页面决定取消某个运行中的 job | 前端在页面上渲染“取消”按钮，携带当前 `jobId` | （无，前端仅为入口） | — |
| 2 | 用户点击“取消”按钮 | 前端向 daemon 发送 `POST /jobs/{jobId}/cancel` 请求 | （无，HTTP handler 负责转发） | — |
| 3 | daemon 接收取消请求并标记 job 已请求取消 | HTTP handler 从 URL 中解析 `jobId`，构造调用 | `CancelJob(ctx, jobId)` | `JobRepository` |
| 4 | 执行循环感知取消并停止派发 | 后续循环中, `AbortCheck` 每轮从 `JobRepository` 读取 Job，发现已请求取消则立即停止派发新子任务 | `ExecutePlan` 内部使用的 `AbortCheck` | `JobRepository` |
| 5 | 用户刷新任务状态页面看到结果 | 浏览器再次访问 `GET /jobs/{jobId}`，daemon 调用 `GetJobState` 返回状态为已中止（aborted）的 Job | `GetJobState(ctx, jobId)` | `JobRepository`, `TaskProgressRepository` |

## 4. GetJobStateUseCase — 编排步骤表（应用层视角，轮询）

| 步骤 | 业务步骤（来自用例） | 入口/App 层动作 | 调用的应用服务方法 | 触达的领域服务 / Port |
| ---- | -------------------- | --------------- | ------------------ | ---------------------- |
| 1 | 用户或脚本希望定期获取某个 job 的当前状态与进度 | 前端或脚本根据 `jobId` 周期性发起 `GET /jobs/{jobId}` 请求 | （无，HTTP handler 负责转发） | — |
| 2 | daemon 收到状态查询请求 | HTTP handler 从 URL 中解析 `jobId`，构造调用 | `GetJobState(ctx, jobId)` | `JobRepository`, `TaskProgressRepository`, 可选 `PlanRepository` |
| 3 | 应用服务组装当前状态 DTO | `GetJobState` 从 Job 与进度恢复当前状态、任务列表与 Plan（如有），组装成 `JobStateDTO` | `GetJobState(ctx, jobId)` | `JobRepository`, `TaskProgressRepository`, `PlanRepository` |
| 4 | daemon 将 `JobStateDTO` 返回给调用方 | HTTP handler 将 `JobStateDTO` 序列化为 JSON 并写入响应 | （封装在 handler 内部） | — |
| 5 | 前端或脚本据此更新 UI 或控制逻辑 | 前端根据返回数据刷新进度条、任务列表与状态标签，脚本可按状态决定是否继续轮询或触发取消 | （无，属于调用方内部逻辑） | — |

## 5. 代码映射（当前实现）

- **CLI 入口**
  - `cmd/pathfinder/main.go`
    - 解析 `-m` 参数。
    - 读取 `PATHFINDER_DAEMON_URL`，默认 `http://127.0.0.1:8080`。
    - `POST /jobs`，解析 `{ "jobId", "url" }` 并打印 `url`。
- **Daemon HTTP 入口**
  - `cmd/pathfinder-daemon/main.go`
    - `POST /jobs` → `handleCreateJob`：
      - 解码 JSON（`goal` / `timeoutSec`）。
      - 构造 `SubmitGoalCommand`，调用 `svc.StartJob`。
      - `go svc.ContinueJob(ctx, jobId)`。
      - 构造 URL `/jobs/{jobId}`，返回 `{ jobId, url }` JSON。
    - `GET /jobs/{jobId}` → `handleGetJob`：
      - 调用 `svc.GetJobState`，直接返回 `JobStateDTO` JSON。
    - `POST /jobs/{jobId}/cancel`（计划新增）：
      - 解析 `jobId`，调用 `svc.CancelJob(ctx, jobId)`，返回 204 或 200。
- **应用服务实现**
  - `internal/orchestration/service.go`
    - `StartJob`：规划 + 保存 Plan，创建 Job（状态 `planning`），保存 Job，返回 `jobId`。
    - `ContinueJob`：初始化 TaskProgress，更新 Job 状态为 `running`，执行 `ExecutePlan`，根据取消/超时结果更新终态（`aborted` 或 `completed`）。
    - `GetJobState`：读取 Job、进度与 Plan，组装 `JobStateDTO`。

## 6. 对应用层的约束

- **CLI 与 daemon 只作为入口层，不承载业务规则**：
  - CLI 不直接调 Planner/JobRepo/TaskProgressRepo，只负责 HTTP 请求与输出。
  - daemon 仅负责 HTTP ↔ 应用服务的方法调用，不直接操作 Plan/Job/TaskProgress 结构。
- **`SubmitGoalUseCase` 的业务编排集中在 `WorkflowOrchestrationApplicationService` 中**：
  - Job/Plan/TaskProgress 的创建、状态流转与执行循环，均由 `StartJob` / `ContinueJob` / `ExecutePlan` / `GetJobState` 实现。
- **对后续扩展的预留**：
  - 若增加 `CancelJobUseCase`、`StreamJobProgressUseCase`，入口层只需增加对应的 HTTP 路由（如 `POST /jobs/{id}/cancel`、SSE `/jobs/{id}/stream`），调用现有或新增的应用服务方法。  

