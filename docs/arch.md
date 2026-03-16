## 1. 代码目录（pathfinder）

项目为 Go 单仓，以下目录树相对仓库根（本文件所在 `docs/` 目录的上一级）。

```text
pathfinder/
├── cmd/
│   ├── pathfinder/
│   │   └── main.go
│   └── tui/
│       └── main.go
├── internal/
│   ├── app/
│   │   └── run.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── provider/
│   │   ├── provider.go
│   │   ├── factory.go
│   │   ├── compatible.go
│   │   ├── credential.go
│   │   ├── types.go
│   │   ├── from_config.go
│   │   ├── factory_test.go
│   │   └── integration_test.go
│   ├── agent/
│   │   ├── agent.go
│   │   ├── dispatcher.go
│   │   ├── loop.go
│   │   └── needs.go
│   ├── skills/
│   │   ├── skills.go
│   │   └── needs.go
│   ├── tools/
│   │   └── tools.go
│   ├── planning/
│   │   ├── plan.go
│   │   ├── errors.go
│   │   └── needs.go
│   ├── runtime/
│   │   ├── run.go
│   │   └── needs.go
│   ├── gateway/
│   │   └── gateway.go
│   ├── channels/
│   │   └── channels.go
│   ├── progress/
│   │   ├── task_progress.go
│   │   ├── progress_maintainer.go
│   │   └── needs.go
│   ├── skillforge/
│   │   ├── skillforge.go
│   │   └── needs.go
│   ├── orchestration/
│   │   ├── service.go
│   │   ├── commands.go
│   │   ├── dtos.go
│   │   └── e2e_phase0_test.go
│   ├── memory/
│   │   ├── memory.go
│   │   └── needs.go
│   ├── observability/
│   │   └── observability.go
│   ├── auth/
│   │   └── auth.go
│   ├── health/
│   │   └── health.go
│   ├── cost/
│   │   └── cost.go
│   ├── tui/
│   │   ├── tui.go
│   │   └── provides.go
│   └── infra/
│       ├── persistence/
│       │   ├── run_repo_mem.go
│       │   ├── plan_repo_mem.go
│       │   ├── task_progress_repo_mem.go
│       │   └── planner_stub.go
│       └── clients/
│           ├── agent_discovery_mem.go
│           ├── dispatcher_stub.go
│           ├── dispatcher_llm.go
│           └── agent_loop_integration_test.go
├── docs/
│   ├── ddd.md
│   ├── T-001-代码目录与文件设计-设计.md
│   └── T-001-代码目录与文件设计-任务清单.md
└── go.mod / go.sum / Makefile 等
```

> 说明：`needs.go` / `provides.go` 目前在部分包中存在，用作端口集中声明；新的设计文档已经不强制要求单独文件，但现有代码仍保留。

---

## 2. 用例与流程 → 代码位置

本节按 ddd.md 第 2 部分的核心用例，列出主步骤与对应的代码入口，方便从业务流程跳到实现。

### 2.1 SubmitGoalUseCase：提交目标并同步执行

| 步骤/流程 | 代码位置 |
|----------|---------|
| 解析 `-m` 参数，选择同步/异步模式 | [`cmd/pathfinder/main.go`](../cmd/pathfinder/main.go) — `main()` |
| 同步模式：调用 Run(message) | [`internal/app/run.go`](../internal/app/run.go) — `Run` |
| 加载配置 Config | [`internal/config/config.go`](../internal/config/config.go) — `Load` |
| 组装编排服务（仓储、规划器、AgentDiscovery、Dispatcher） | [`internal/app/run.go`](../internal/app/run.go) — `Run` 中构造 `WorkflowOrchestrationApplicationService` |
| 提交目标（规划 + 初始化进度 + 执行 + 总结） | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `SubmitGoal` |
| 规划：根据 GoalDescription 生成 Plan | [`internal/infra/persistence/planner_stub.go`](../internal/infra/persistence/planner_stub.go) — `PlannerStub.PlanGoal`（实现）；接口定义见 [`internal/planning/needs.go`](../internal/planning/needs.go) |
| 校验计划结构 | [`internal/planning/plan.go`](../internal/planning/plan.go) — `Plan.Validate` |
| 持久化 Plan | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `SubmitGoal` 内调用 `PlanRepo.Save`（实现：[`internal/infra/persistence/plan_repo_mem.go`](../internal/infra/persistence/plan_repo_mem.go)） |
| 创建 Run 并持久化 | [`internal/runtime/run.go`](../internal/runtime/run.go) — `Create`；使用位置：[`internal/orchestration/service.go`](../internal/orchestration/service.go) — `SubmitGoal`；仓储实现：[`internal/infra/persistence/run_repo_mem.go`](../internal/infra/persistence/run_repo_mem.go) |
| 初始化每个 SubTask 的 TaskProgress | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `SubmitGoal` 中循环构造 `progress.TaskProgress` 并调用 `TaskProgress.Save`；仓储实现：[`internal/infra/persistence/task_progress_repo_mem.go`](../internal/infra/persistence/task_progress_repo_mem.go) |
| 执行计划（调用 ExecutePlan） | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `SubmitGoal` 末尾 `ExecutePlan` 调用 |
| 汇总结果并打印阶段/进度/总结 | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `SummarizeRun`；输出位置：[`internal/app/run.go`](../internal/app/run.go) — `Run` |

### 2.2 ExecutePlanUseCase：按计划执行（循环派发子任务）

| 步骤/流程 | 代码位置 |
|----------|---------|
| 读取 Run 与 Plan | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `ExecutePlan` |
| 构造 AbortCheck（取消/超时检测） | 同上：`ExecutePlan` 内闭包 `abortCheck` 基于 `RunRepository.Get` + `Run.IsCancelRequested/IsOverDeadline` |
| 构造 Loop（Dispatcher + ProgressRepo + AgentDiscovery + AbortCheck） | 同上：`ExecutePlan` 内构造 `agent.Loop` |
| 循环：恢复进度（ListByRunId） | [`internal/agent/loop.go`](../internal/agent/loop.go) — `Loop.Run` 第一段调用 `ProgressRepo.ListByRunId` |
| 计算已完成任务集合 | 同上：`Loop.Run` 中构造 `done[TaskId]` |
| 依据依赖找出就绪 SubTask | 同上：`Loop.Run` 遍历 `plan.SubTasks` 和 `plan.Dependencies` |
| 选择 Agent（SuggestedAgent / AgentDiscovery.ListAgents） | 同上：`Loop.Run` 调用 `plan.SuggestedAgentFor` 与 `Discovery.ListAgents` |
| 标记任务开始并持久化 | 同上：`Loop.Run` 中 `TaskProgress.Start` + `ProgressRepo.Save` |
| 调用 Dispatcher 执行子任务 | 同上：`Loop.Run` 调用 `Dispatcher.Dispatch`；实现：[`internal/infra/clients/dispatcher_llm.go`](../internal/infra/clients/dispatcher_llm.go)、[`internal/infra/clients/dispatcher_stub.go`](../internal/infra/clients/dispatcher_stub.go) |
| 根据结果 Complete/Fail 并持久化 | 同上：`Loop.Run` 中 `TaskProgress.Complete/Fail` + `ProgressRepo.Save` |
| 根据 AbortCheck 或无可执行任务退出循环 | 同上：`Loop.Run` 顶部 AbortCheck 与 `advanced` 标志 |
| 更新 Run 状态为 Completed / Aborted | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `ExecutePlan` 尾部调用 `Run.MarkAborted/MarkCompleted` 并 Save |

### 2.3 StartRun/ContinueRun：异步执行 + TUI

| 步骤/流程 | 代码位置 |
|----------|---------|
| 异步入口：根据 -m 构造 Run 并立即返回 runId | [`internal/app/run.go`](../internal/app/run.go) — `RunAsync` 调用 `StartRun` |
| 启动后台执行 ContinueRun | 同上：`go svc.ContinueRun(ctx, runId)` |
| StartRun：规划并创建 Run（不执行） | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `StartRun` |
| ContinueRun：初始化 TaskProgress | 同上：`ContinueRun` 中为每个 SubTask 创建 `TaskProgress` 并 Save |
| ContinueRun：设置 RunStatus=running 并 Save | 同上：`ContinueRun` |
| ContinueRun：调用 ExecutePlan 与终态更新 | 同上：`_ = s.ExecutePlan` 后再次读取 Run 并 MarkAborted/MarkCompleted + Save |
| TUI：根据是否 TTY 选择 RunAsync + tui.Run | [`cmd/pathfinder/main.go`](../cmd/pathfinder/main.go) — `main`；[`internal/tui/tui.go`](../internal/tui/tui.go) — `Run` |
| TUI：轮询 GetRunState 构建 RunStateDTO | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `GetRunState`；`tui.model.pollCmd` 周期性调用 `getState()` |

### 2.4 CancelRunUseCase / StreamRunProgressUseCase

| 用例/步骤 | 代码位置 |
|----------|---------|
| 取消：标记 Run.CancelRequested 并持久化 | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `CancelRun` |
| 执行循环中感知取消/超时 | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `ExecutePlan` 内 `abortCheck`；[`internal/agent/loop.go`](../internal/agent/loop.go) — `Loop.Run` 顶部调用 `AbortCheck` |
| TUI 取消：按 c 调用 cancelRun | [`internal/tui/tui.go`](../internal/tui/tui.go) — `model.Update` 中处理 key `c` 调用 `cancelRun` |
| 查询进度：GetRunState 构建 RunStateDTO | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `GetRunState` |
| TUI 展示进度/执行图 | [`internal/tui/tui.go`](../internal/tui/tui.go) — `model.Update` 处理 `RunStateMsg`；`renderPaneTopLeft` / `renderExecutionGraph` |

---

## 3. Application：主流程与归属

### 3.1 主流程（SubmitGoal 同步执行）

| 步骤 | 描述 | 归属层 | 代码位置 |
|------|------|--------|----------|
| 1 | 解析 CLI 参数 `-m` | 入口层（CLI） | [`cmd/pathfinder/main.go`](../cmd/pathfinder/main.go) — `main` |
| 2 | 加载配置 Config | 应用层（App 辅助） | [`internal/app/run.go`](../internal/app/run.go) — `Run` 调用 `config.Load` |
| 3 | 组装编排服务及 Infra 依赖 | 组合根（入口层） | [`internal/app/run.go`](../internal/app/run.go) — `Run` 中 new RunRepo/PlanRepo/TaskProgressRepo/PlannerStub/AgentDiscovery/Dispatcher |
| 4 | 调用 SubmitGoal（规划 + 初始化进度 + 执行 + 总结） | 应用层（编排服务） | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `SubmitGoal` |
| 5 | 规划、校验 Plan | 领域层（Planning） | [`internal/planning/plan.go`](../internal/planning/plan.go) — `Plan.Validate`；规划实现在 [`internal/infra/persistence/planner_stub.go`](../internal/infra/persistence/planner_stub.go) |
| 6 | 创建并持久化 Run | 领域层（Runtime）+ Infra | [`internal/runtime/run.go`](../internal/runtime/run.go) — `Create`；[`internal/infra/persistence/run_repo_mem.go`](../internal/infra/persistence/run_repo_mem.go) |
| 7 | 初始化 TaskProgress | 领域层（Progress）+ Infra | [`internal/progress/task_progress.go`](../internal/progress/task_progress.go) + [`internal/infra/persistence/task_progress_repo_mem.go`](../internal/infra/persistence/task_progress_repo_mem.go) |
| 8 | 执行计划（Loop.Run） | 领域层（Agent/Progress/Planning/Runtime）+ Infra（Dispatcher/AgentDiscovery） | [`internal/agent/loop.go`](../internal/agent/loop.go) + [`internal/infra/clients/*`](../internal/infra/clients) |
| 9 | 汇总结果并返回 DTO | 应用层（编排服务） | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `SummarizeRun` |

### 3.2 层职责总览

| 层 | 职责 | 代表文件 |
|----|------|----------|
| 入口层（CLI/TUI） | 参数解析、选择同步/异步、启动 TUI | [`cmd/pathfinder/main.go`](../cmd/pathfinder/main.go), [`cmd/tui/main.go`](../cmd/tui/main.go) |
| Application（编排） | 依赖组装、主用例编排、DTO 组装 | [`internal/app/run.go`](../internal/app/run.go), [`internal/orchestration/service.go`](../internal/orchestration/service.go), [`internal/orchestration/commands.go`](../internal/orchestration/commands.go), [`internal/orchestration/dtos.go`](../internal/orchestration/dtos.go) |
| Domain | 业务规则：Plan/Run/TaskProgress 状态、不变式、依赖判定 | [`internal/planning/plan.go`](../internal/planning/plan.go), [`internal/runtime/run.go`](../internal/runtime/run.go), [`internal/progress/task_progress.go`](../internal/progress/task_progress.go), [`internal/agent/loop.go`](../internal/agent/loop.go), [`internal/agent/agent.go`](../internal/agent/agent.go) |
| Infra | 仓储/客户端实现、LLM 调用、Agent 发现 | [`internal/infra/persistence/*.go`](../internal/infra/persistence), [`internal/infra/clients/*.go`](../internal/infra/clients), [`internal/provider/*.go`](../internal/provider) |
| UI（TUI） | Run 状态可视化、轮询进度、取消 | [`internal/tui/tui.go`](../internal/tui/tui.go) |

---

## 4. 组合根（Composition Root）

当前组合根集中在 **入口层**：

- **CLI 入口**：[`cmd/pathfinder/main.go`](../cmd/pathfinder/main.go)
  - 负责解析命令行参数，决定调用 `app.Run` 或 `app.RunAsync` + `tui.Run`。
- **应用组装点**：[`internal/app/run.go`](../internal/app/run.go)
  - `Run` / `RunAsync` 内：
    - 构造并注入：
      - `RunRepo` = `NewRunRepoMem()`（内存 RunRepository）
      - `PlanRepo` = `NewPlanRepoMem()`（内存 PlanRepository）
      - `TaskProgressRepo` = `NewTaskProgressRepoMem()`（内存 TaskProgressRepository）
      - `Planner` = `&PlannerStub{}`
      - `AgentDiscovery` = `NewAgentDiscoveryMem()`
      - `Dispatcher` = `DispatcherLLM`（成功创建 Provider 时）或 `DispatcherStub`
    - new `WorkflowOrchestrationApplicationService{...}` 并调用其用例方法。
- **TUI 独立入口**：[`cmd/tui/main.go`](../cmd/tui/main.go)
  - 单独启动 TUI，用占位模式展示 UI（无 getState/cancelRun）。

组合根只存在于 CLI/TUI 入口层；Application 与 Domain 不 new 具体 Infra 实现，只依赖接口或抽象类型。

---

## 5. 关键逻辑链条

本节列出当前实现中较核心且容易出错的机制链条，便于排查与演进。

### 5.1 关键逻辑链表

| 链条 | 链条描述（顺序） | 触发条件/前提 | 代码跳转 |
|------|------------------|---------------|----------|
| 提交目标端到端链 | CLI 解析 `-m` → `app.Run` 加载 Config + 组装依赖 → `WorkflowOrchestrationApplicationService.SubmitGoal` → `Planner.PlanGoal` + `Plan.Validate` + `PlanRepo.Save` → `Run.Create` + `RunRepo.Save`（planning → running）→ 初始化 `TaskProgress` 并 Save → `ExecutePlan`（见下一条链）→ `SummarizeRun` → CLI 打印阶段/进度/总结 | 用户在命令行执行 `pathfinder -m \"...\"`（非 TTY 或同步模式） | [`cmd/pathfinder/main.go`](../cmd/pathfinder/main.go) → [`internal/app/run.go`](../internal/app/run.go) → [`internal/orchestration/service.go`](../internal/orchestration/service.go) → [`internal/planning/plan.go`](../internal/planning/plan.go) → [`internal/runtime/run.go`](../internal/runtime/run.go) → [`internal/progress/task_progress.go`](../internal/progress/task_progress.go) |
| 执行计划循环链 | `ExecutePlan` 取 Run+Plan → 构造 `AbortCheck`（读 RunRepo，检查 CancelRequested/Deadline）→ new `agent.Loop`（注入 Dispatcher/ProgressRepo/AgentDiscovery/AbortCheck）→ `Loop.Run`：ListByRunId 恢复进度 → 计算 done 集合 → 依据 Dependencies 选就绪 SubTask → 用 SuggestedAgent 或 AgentDiscovery.Select 第一个 Agent → `TaskProgress.Start` + Save → `Dispatcher.Dispatch` 执行 → `TaskProgress.Complete/Fail` + Save → 如有新进展则继续循环，否则退出 → `ExecutePlan` 再次检查 Run 终态并 MarkCompleted/MarkAborted + Save | 任何进入 `ExecutePlan` 的同步或异步执行路径 | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `ExecutePlan` → [`internal/agent/loop.go`](../internal/agent/loop.go) → [`internal/infra/clients/*.go`](../internal/infra/clients) |
| 取消/中止链 | TUI 或上层调用 `CancelRun(runId)` → `Run.Cancel` + Save → 后续 `ExecutePlan` / `Loop.Run` 每轮开头通过 `AbortCheck` 读取 Run 并检查 CancelRequested/Deadline → true 时立即返回，不再派发新任务 → `ExecutePlan`/`ContinueRun` 末尾根据 CancelRequested/Deadline 调用 `Run.MarkAborted` + Save → GetRunState/SummarizeRun 看到状态为 aborted，TUI 显示“已取消” | 运行中用户在 TUI 按 `c` 或调用 CancelRun API；Run 设置了 Deadline 且已过期 | [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `CancelRun` / `ExecutePlan` / `ContinueRun` → [`internal/runtime/run.go`](../internal/runtime/run.go) → [`internal/tui/tui.go`](../internal/tui/tui.go) — 处理 key `c` 与 `RunStateMsg` |
| TUI 轮询与可视化链 | CLI 在 TTY 下调用 `RunAsync` → `StartRun` 返回 runId → 后台 `ContinueRun` 执行 → `tui.Run(runId, getState, cancelRun)` 启动 Bubbletea 程序 → `model.Init`/`pollCmd` 周期性调用 getState（内部是 `GetRunState`）→ `GetRunState` 读取 Run + Plan + Tasks，并构建 `RunStateDTO`（含 Plan.SubTasks/Dependencies 与 TaskProgress 列表）→ TUI `Update(RunStateMsg)` 更新 phase/completed/total/Summary 与执行图 → `View` 渲染四宫格（状态/子任务、思考过程、执行图、输入） | 在 TTY 下运行 `pathfinder -m \"...\"`，并成功进入异步模式 | [`cmd/pathfinder/main.go`](../cmd/pathfinder/main.go) → [`internal/app/run.go`](../internal/app/run.go) — `RunAsync` → [`internal/orchestration/service.go`](../internal/orchestration/service.go) — `StartRun` / `ContinueRun` / `GetRunState` → [`internal/tui/tui.go`](../internal/tui/tui.go) — `Run` / `model.Update` / `View` |

