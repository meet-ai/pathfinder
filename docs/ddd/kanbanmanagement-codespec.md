## KanbanManagementContext — Codespec

> 当前聚焦：`ManageKanbanCardLifecycleUseCase`，并已落地为「前端逐步触发的分步接口」。

## 1. 已落地文件清单（Application + Adapter）

| 文件路径 | 类型 | 职责 |
| ---- | ---- | ---- |
| `internal/kanban/application.go` | Application Service | 提供分步用例方法：`CreateCard`、`AssignCard`、`MoveCard`、`BlockCard`、`UnblockCard`、`ReviewPassCard`、`ListCards`、`GetCard`。 |
| `internal/kanban/domain.go` | Domain 行为 | 状态机迁移、阻塞/解除阻塞、WIP 校验（`Card.MoveTo`、`Card.Block`、`Board.ValidateWip`）。 |
| `internal/kanban/needs.go` | Port 依赖声明 | 定义 `BoardRepository`、`CardRepository`、`AssigneeResolver`、`EventPublisher`。 |
| `internal/kanban/dtos.go` | DTO | 输出类型：`CardDTO`、`KanbanCardLifecycleDTO`。 |
| `internal/infra/persistence/kanban_repo_mem.go` | Infra 仓储实现 | 内存版看板/卡片仓储。 |
| `internal/infra/clients/kanban_support.go` | Infra 端口实现 | 事件发布器（内存占位）。 |
| `internal/infra/clients/kanban_assignee_from_sync.go` | Infra 端口实现 | AssigneeResolver：从 CanonicalStore 读拓扑（agents/bindings），按 taskType 匹配 binding 或取首 agent。 |
| `cmd/pathfinder-daemon/main.go` | HTTP 入口适配器 | 暴露 Kanban 读写 API，并装配 `kanban.ApplicationService`。 |
| `cmd/pathfinder-daemon/web/kanban/index.html` | 最小前端 | 新建卡片 + 分步推进按钮，按用例顺序驱动接口。 |

## 2. 分步接口与用例步骤映射

| 用例步骤 | HTTP 入口 | 应用层方法 | Domain/Port 参与方 |
| ---- | ---- | ---- | ---- |
| 1. 创建卡片进入 `todo` | `POST /api/kanban/cards` | `CreateCard` | `NewCard`、`CardRepository.Create`、`EventPublisher.Publish` |
| 2. 分配 assignee | `POST /api/kanban/cards/{cardId}/assign` | `AssignCard` | `AssigneeResolver.Resolve`（数据源=CanonicalStore 拓扑）、`Card.Assign`、`CardRepository.Save` |
| 3. 进入 `in_progress` | `POST /api/kanban/cards/{cardId}/move` with `to=in_progress` | `MoveCard` | `Board.ValidateWip`、`Card.MoveTo` |
| 4. 标记 `blocked` | `POST /api/kanban/cards/{cardId}/block` | `BlockCard` | `Card.Block` |
| 5. 解除阻塞回 `in_progress` | `POST /api/kanban/cards/{cardId}/unblock` | `UnblockCard` | `Card.Unblock` |
| 6. 进入 `in_review` | `POST /api/kanban/cards/{cardId}/move` with `to=in_review` | `MoveCard` | `Board.ValidateWip`、`Card.MoveTo` |
| 7. 审核通过到 `done` | `POST /api/kanban/cards/{cardId}/review-pass` | `ReviewPassCard` | `Card.MoveTo`、`EventPublisher.Publish` |
| 8. 看板展示与详情 | `GET /api/kanban/board` / `GET /api/kanban/cards/{cardId}` | `ListCards` / `GetCard` | `CardRepository.ListByBoard` / `Get` |

## 3. 入口调用顺序（前端驱动）

- `/kanban` 页面由用户操作按钮驱动，按业务节奏逐步调用写接口，而不是单次生命周期直达 `done`。
- 每次写操作后刷新 `GET /api/kanban/board` 与 `GET /api/kanban/cards/{cardId}`，形成可观测闭环。
- 默认看板 `boardId = main-board`，daemon 启动时自动初始化。

## 4. 输入/输出结构（当前）

- `POST /api/kanban/cards`
  - 入参：`boardId`、`title`、`description`、`creator`、`reviewer`
  - 出参：`CardDTO`
- `POST /api/kanban/cards/{cardId}/assign`
  - 入参：`taskType`、`operator`
  - 出参：`CardDTO`
- `POST /api/kanban/cards/{cardId}/move`
  - 入参：`to`（`in_progress` 或 `in_review`）、`operator`
  - 出参：`CardDTO`
- `POST /api/kanban/cards/{cardId}/block`
  - 入参：`reason`、`operator`
  - 出参：`CardDTO`
- `POST /api/kanban/cards/{cardId}/unblock`
  - 入参：`operator`
  - 出参：`CardDTO`
- `POST /api/kanban/cards/{cardId}/review-pass`
  - 入参：`reviewer`
  - 出参：`CardDTO`

## 5. 约束与后续

- 应用层只做编排，状态机/WIP 等规则在领域层实现。
- 当前实现是内存仓储，满足 MVP；重启进程后数据清空。
- 待补：失败路径专项测试（WIP 超限、非法迁移）与 SLA 自动升级动作。

## 6. 与 OpenClaw 融合的数据来源

- **看板/卡片数据源**：设计上来自「同步管理(G)」的 canonical 存储；当前卡片/看板仍用内存仓储，CreateCard/AssignCard 写回看板仓储；同步存储仅作为派单的**只读**数据源。
- **AssigneeResolver**：**已实现**。`KanbanAssigneeResolverFromSync` 从 CanonicalStore 读拓扑（GetTopology），按 taskType 与 binding.MatchType 匹配选 agent，否则取第一个 agent；无拓扑时回退默认 id。daemon 装配时注入同一 canonicalStore（与 sync 共用）。
- **后续用例（不改变当前编排）**：EscalateOverdueCards（升级）等读同一同步存储，与当前 ManageKanbanCardLifecycleUseCase 解耦。
