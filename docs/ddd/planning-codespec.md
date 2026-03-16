## PlanningContext — Codespec

> 当前聚焦：`PlanGoalUseCase`（输入 GoalDescription，输出 Plan/子任务/依赖/建议 agent）

## 1. 参与者与接口

- **调用方（上游 JobManagementContext）**
  - 调用点：在 `SubmitGoalUseCase.StartJob` 中，根据用户提交的 `GoalDescription` 触发规划。
  - 职责：将 `GoalDescription` 作为输入传入规划应用服务，拿到 `Plan` 后与 Job 一起持久化并驱动后续执行。
- **应用服务（PlanningApplicationService，后续由代码收紧到具体方法签名）**
  - `PlanGoal(ctx, goal GoalDescription) (*Plan, error)`
- **参与的领域服务 / Port（来自 PlanningContext）**
  - `Planner`：`PlanGoal(ctx, GoalDescription) (*Plan, error)`
  - `PlanRepository`：`Save(ctx, *Plan) error`、`Get(ctx, PlanId) (*Plan, error)`
  - `PlanLearning`（可选）：提供相似历史计划与 Run 结果，支持 Planner 自学习。

## 2. PlanGoalUseCase — 编排步骤表（应用层视角）

| 步骤 | 业务步骤（来自用例） | 应用层动作 | 调用的应用服务方法 | 触达的领域服务 / Port |
| ---- | -------------------- | ---------- | ------------------ | ---------------------- |
| 1 | 用户或上游用例提交包含目标描述（GoalDescription）的规划请求。 | JobManagement 的 `StartJob` 构造 `GoalDescription`，调用 Planning 应用服务。 | `PlanGoal(ctx, goal)` | `Planner`，可选 `PlanLearning` |
| 2 | 系统根据 GoalDescription 先生成整体任务规划思路，识别主要阶段与关键里程碑。 | 应用服务将 `goal` 转交给 `Planner`，由 Planner 内部完成研究与高层规划思路生成。 | `PlanGoal(ctx, goal)` | `Planner`，可选 `PlanLearning` |
| 3 | 系统在规划思路基础上，将目标分解为若干子任务，并为每个子任务补充说明与预期产出。 | `Planner` 返回包含 `SubTasks` 的 `Plan` 结构，每个子任务带有 `TaskId` 与描述。 | —（在同一次 `PlanGoal` 调用内完成） | `Planner` |
| 4 | 系统在子任务之间显式标注依赖关系，构建带依赖边的任务图结构（Task Graph）。 | `Planner` 在返回的 `Plan` 中填充 `Dependencies`（`From` → `To`）。 | — | `Planner` |
| 5 | 系统为任务图中的各个子任务绑定建议负责的 agent 或技能能力，形成可执行的计划（Plan）。 | `Planner` 在 `Plan.SuggestedAgents` 中为部分或全部子任务绑定 `AgentId` 建议。 | — | `Planner` |
| 6 | 在后续执行过程中，系统根据子任务执行结果和环境变化，按需对任务图中的子任务、依赖关系和顺序做动态调整与重规划。 | 规划应用服务预留接口（未来如 `ReplanForRun`），当前版本仅返回静态 `Plan`；动态调整由执行侧（Runtime/WorkflowOrchestration）在后续版本通过再次调用 Planner 实现。 | （暂不实现，仅在 docs 中预留） | `Planner`，`PlanLearning` |
| 7 | 当用户追加或修改目标或约束时，系统基于最新输入与当前任务图状态，增删或重组相关子任务并更新 Plan，保持计划与用户意图对齐。 | 由上游用例在检测到用户修改目标时，再次调用规划应用服务生成新 Plan，并与旧 Plan 对比（本轮仅在文档中描述，不立刻落地代码）。 | （暂不实现，仅在 docs 中预留） | `Planner`，`PlanLearning` |

## 3. 与现有代码的映射（当前设计）

- `internal/planning/plan.go`
  - 定义 `Plan` 聚合根：`PlanId`、`SubTasks`、`Dependencies`、`SuggestedAgents`。
  - 提供 `Validate()`，校验子任务与依赖合法。
- `internal/planning/needs.go`
  - 定义 `Planner`、`PlanRepository`、`PlanLearning` 端口。
- `internal/orchestration/service.go`（上游用例位置，仅作为参考，不在本上下文内实现）
  - `StartJob` 将在内部调用 Planning 应用服务的 `PlanGoal`，拿到 `Plan` 后持久化并关联到 Job。

> 约束：本轮迭代仅在 PlanningContext 中明确 `PlanGoalUseCase` 的应用层编排与接口形状，不修改 Planner 领域接口签名；后续收紧由对照此 spec 与现有实现逐步完成。

