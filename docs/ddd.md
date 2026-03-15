# 多智能体目标工作流 — DDD 设计文档

> 基于 FEATURES.md 与《多智能体目标工作流设计》整理；按 DDD 阶段产出用例、限界上下文、领域模型、应用层与基础设施层设计。  
> 文档时间：2026-03

---

# 第1部分：业务理解

## 1.1 业务目标

- **主目标**：用户提交高层目标后，系统完成「规划 → 执行 → 按需生成 Skill/Tool → 进度与中止 → 总结沉淀」的完整工作流，并支持多 agent 并行/串行执行与统一反馈。
- **关键价值**：一条命令或一次 API 调用即可触发目标执行；实时进度与流式输出；缺能力时自动生成并安装 Skill/Tool；支持取消、超时与失败策略。

## 1.2 参与者

| 角色 | 说明 |
|------|------|
| **用户** | 通过 CLI（如 `pathfinder -m "目标"`）、API 或消息渠道（Chat/Slack/WebChat）提交目标；查看进度、取消任务、查看结果。 |
| **编排层** | 规划、分支、监督、总结；维护 state；不直接执行子任务。 |
| **执行层** | 按 state 派发子任务到 agent（openclaw:\<id\>/acpx）；写回执行结果。 |
| **Agent** | 执行具体子任务；可具备 Skill/Tool；由 Gateway/ClawHub 等发现。 |

## 1.3 业务流程映射

| 阶段 | 业务内容 | 对应特性 |
|------|----------|----------|
| 1. 发布任务 | 提交高层目标，获得 run_id/流式句柄；可选参数（超时、优先级、agent 池）。 | F1.1–F1.3 |
| 2. 规划与探索 | 任务分解 → 子任务列表；依赖分析 → DAG/顺序；分派建议 agent；结果写 state。 | F2.1–F2.4 |
| 3. 执行与派发 | 按依赖串行/并行执行；派发到对应 agent；结果写回 state。 | F3.1–F3.4 |
| 4. Skill/Tool 按需生成 | 检测缺失 → 路由生成；生成并安装/注册；重新调度执行。 | F4.1–F4.6 |
| 5. 进度与中止 | 任务级进度与持久化；流式/进度事件；超时/用户取消/失败策略 → 中止与清理。 | F5.1–F5.5 |
| 6. 总结与沉淀 | 汇总子任务结果；可选写回知识库；可选复盘；新 Skill/Tool 登记到目录/ClawHub。 | F6.1–F6.4 |

---

# 第2部分：用例识别

## 2.1 用例脑暴（用例卡片）

| 角色 | 操作 | 目的 | 用例名 | 类型 | 优先级 |
|------|------|------|--------|------|--------|
| 用户/系统 | 提交高层目标 | 进入工作流并获 run_id/流式句柄 | SubmitGoalUseCase | [O] | P0 |
| 用户/系统 | 通过消息渠道提交 | 从 Chat/Slack 等触发任务 | SubmitGoalViaChannelUseCase | [O] | P1 |
| 编排层 | 分解目标、分析依赖、分派 agent | 产出可执行计划 | PlanGoalUseCase | [C] | P0 |
| 编排层 | 按计划串行/并行执行子任务 | 完成子任务并写回结果 | ExecutePlanUseCase | [O] | P0 |
| 执行层 | 派发子任务到指定 agent | 由 agent 执行并返回结果 | DispatchTaskToAgentUseCase | [C] | P0 |
| 编排层 | 检测缺 Skill 并路由生成 | 不卡死、进入生成分支 | DetectMissingSkillAndRouteUseCase | [C] | P1 |
| 编排层 | 生成 Skill 包并安装到 agent | 目标 agent 可见可用 | GenerateAndInstallSkillUseCase | [C] | P1 |
| 编排层 | 检测缺 Tool 并路由生成 | 进入 Tool 生成分支 | DetectMissingToolAndRouteUseCase | [C] | P1 |
| 编排层 | 生成 Tool 并注册绑定 agent | 下一轮执行可用 | GenerateAndRegisterToolUseCase | [C] | P1 |
| 编排层 | 维护任务进度、checkpoint、恢复 | 监督与总结可读 | MaintainTaskProgressUseCase | [C] | P0 |
| 用户/监督者 | 取消 run | 走清理与总结 | CancelRunUseCase | [C] | P0 |
| 编排层 | 超时/失败策略检测并中止 | 进入中止与总结 | AbortRunOnConditionUseCase | [C] | P0 |
| 编排层 | 汇总子任务结果、生成报告 | 用户/下游获得产出 | SummarizeRunUseCase | [C] | P0 |
| 编排层 | 登记新 Skill/Tool 到目录 | 其他 run/agent 可复用 | RegisterSkillOrToolToCatalogUseCase | [C] | P2 |
| 用户/系统 | 查询可用 agent 列表 | 规划与派发时选择 agent | ListAgentsUseCase | [Q] | P0 |
| 用户/系统 | 查询某 agent 是否具备某 Skill/Tool | 检测缺失、生成前校验 | CheckAgentCapabilityUseCase | [Q] | P1 |
| 用户/系统 | 订阅 run 进度与流式输出 | 实时查看步骤、agent、进度% | StreamRunProgressUseCase | [Q] | P0 |

## 2.2 用例标准化（按类型分组）

- **[O] 编排用例**：SubmitGoalUseCase、SubmitGoalViaChannelUseCase、ExecutePlanUseCase（依赖 Plan、Dispatch、Progress、Summarize 等）。
- **[C] 命令用例**：PlanGoalUseCase、DispatchTaskToAgentUseCase、DetectMissingSkillAndRouteUseCase、GenerateAndInstallSkillUseCase、DetectMissingToolAndRouteUseCase、GenerateAndRegisterToolUseCase、MaintainTaskProgressUseCase、CancelRunUseCase、AbortRunOnConditionUseCase、SummarizeRunUseCase、RegisterSkillOrToolToCatalogUseCase。
- **[Q] 查询用例**：ListAgentsUseCase、CheckAgentCapabilityUseCase、StreamRunProgressUseCase。

## 2.3 用例分组（按业务域）

| 业务域 | 职责 | 包含用例 |
|--------|------|----------|
| **能力目录** | 谁可用、会什么；发现与登记 | ListAgentsUseCase、CheckAgentCapabilityUseCase、RegisterSkillOrToolToCatalogUseCase（登记） |
| **规划契约** | 计划结构、校验、state 约定 | PlanGoalUseCase（产出）、ExecutePlanUseCase（消费计划）中涉及的结构与校验 |
| **运行时** | run 生命周期、派发、流式、取消 | SubmitGoalUseCase、SubmitGoalViaChannelUseCase、DispatchTaskToAgentUseCase、CancelRunUseCase、StreamRunProgressUseCase |
| **执行状态** | 任务进度、checkpoint、持久化 | MaintainTaskProgressUseCase、AbortRunOnConditionUseCase（读进度/失败次数） |
| **能力生成** | Skill/Tool 缺失检测与生成安装 | DetectMissingSkillAndRouteUseCase、GenerateAndInstallSkillUseCase、DetectMissingToolAndRouteUseCase、GenerateAndRegisterToolUseCase |
| **工作流编排** | 端到端编排 | ExecutePlanUseCase、SummarizeRunUseCase（编排上述各域） |

## 2.4 用例依赖关系（编排用例分解）

```
SubmitGoalUseCase [O]
  └─ 创建 Run（运行时）
  └─ PlanGoalUseCase [C]（规划契约）
  └─ ExecutePlanUseCase [O]

ExecutePlanUseCase [O]
  ├─ MaintainTaskProgressUseCase [C]（更新进度）
  ├─ DispatchTaskToAgentUseCase [C]（运行时）
  ├─ DetectMissingSkillAndRouteUseCase [C] / GenerateAndInstallSkillUseCase [C]（能力生成）
  ├─ DetectMissingToolAndRouteUseCase [C] / GenerateAndRegisterToolUseCase [C]（能力生成）
  ├─ AbortRunOnConditionUseCase [C]（执行状态）
  └─ SummarizeRunUseCase [C]
```

---

# 第3部分：限界上下文设计

## 3.1 限界上下文列表

| 上下文 | 职责 | 核心领域对象 | 用例示例 |
|--------|------|--------------|----------|
| **CapabilityCatalogContext**（能力目录） | 执行体与能力发现、检索、登记 | Agent、Skill、Tool、AgentPool | ListAgents、CheckAgentCapability、RegisterSkillOrToolToCatalog |
| **PlanningContext**（规划契约） | 计划结构、校验、与 state 约定 | Plan、SubTask、Dependency、SuggestedAgent | PlanGoal（产出）、ExecutePlan（消费） |
| **RuntimeContext**（运行时） | run 生命周期、派发、流式、取消 | Run、Dispatch、StreamHandle | SubmitGoal、DispatchTaskToAgent、CancelRun、StreamRunProgress |
| **ExecutionStateContext**（执行状态） | 任务进度、checkpoint、恢复 | RunProgress、TaskProgress、Checkpoint | MaintainTaskProgress、AbortRunOnCondition |
| **CapabilityGenerationContext**（能力生成） | Skill/Tool 缺失检测、生成、安装/注册 | SkillPackage、ToolSpec、GenerationRequest | DetectMissingSkill/Tool、GenerateAndInstallSkill、GenerateAndRegisterTool |
| **WorkflowOrchestrationContext**（工作流编排） | 端到端编排、总结 | Run、Plan、Summary | ExecutePlan、SummarizeRun |

## 3.2 上下文映射关系

### Customer-Supplier（下游依赖上游）

- WorkflowOrchestrationContext → PlanningContext（消费 Plan）
- WorkflowOrchestrationContext → RuntimeContext（创建 Run、派发、取消、流式）
- WorkflowOrchestrationContext → ExecutionStateContext（读写进度、中止判断）
- WorkflowOrchestrationContext → CapabilityCatalogContext（ListAgents、CheckAgentCapability、登记）
- WorkflowOrchestrationContext → CapabilityGenerationContext（检测缺失、生成并安装/注册）
- RuntimeContext → CapabilityCatalogContext（按 agent_id 派发前可查 agent）
- CapabilityGenerationContext → CapabilityCatalogContext（安装/登记后更新目录）

### Shared Kernel（共享）

- PlanningContext 与 ExecutionStateContext 共享「子任务 id、状态、建议 agent」等字段约定（PlanSchema 与 Progress 读写契约一致）。

### 依赖分析（YAML 风格）

```yaml
contexts:
  - name: CapabilityCatalogContext
    responsibility: 执行体与能力发现、检索、登记
    upstream_of: [RuntimeContext, WorkflowOrchestrationContext, CapabilityGenerationContext]
    downstream_of: []

  - name: PlanningContext
    responsibility: 计划结构、校验、state 约定
    upstream_of: [WorkflowOrchestrationContext]
    downstream_of: []

  - name: RuntimeContext
    responsibility: run 生命周期、派发、流式、取消
    upstream_of: [WorkflowOrchestrationContext]
    downstream_of: [CapabilityCatalogContext]

  - name: ExecutionStateContext
    responsibility: 任务进度、checkpoint、恢复
    upstream_of: [WorkflowOrchestrationContext]
    downstream_of: []

  - name: CapabilityGenerationContext
    responsibility: Skill/Tool 缺失检测、生成、安装/注册
    upstream_of: [WorkflowOrchestrationContext]
    downstream_of: [CapabilityCatalogContext]

  - name: WorkflowOrchestrationContext
    responsibility: 端到端编排、总结
    upstream_of: []
    downstream_of: [PlanningContext, RuntimeContext, ExecutionStateContext, CapabilityCatalogContext, CapabilityGenerationContext]
```

## 3.3 上下文边界说明（摘要）

- **CapabilityCatalogContext**：只负责「谁可用、会什么」的查询与登记；不负责执行、不负责生成内容。
- **PlanningContext**：只负责计划的结构与校验；不负责执行、不负责持久化 run。
- **RuntimeContext**：只负责 run 的创建、派发、流式、取消；不负责规划、不负责进度持久化细节。
- **ExecutionStateContext**：只负责任务级进度与 checkpoint；不负责派发、不负责规划。
- **CapabilityGenerationContext**：只负责检测缺失、生成包、安装/注册；不负责编排顺序、不负责 agent 发现。
- **WorkflowOrchestrationContext**：编排上述上下文完成 SubmitGoal、ExecutePlan、SummarizeRun；不实现具体规划算法、不实现具体派发协议。

---

# 第4部分：目录结构设计

**原则**：按**职责**分包，不使用 `domain`/`application`/`infrastructure`/`vo` 等 DDD 结构名；每包内用 `provides.go` 表示对外提供接口、`needs.go` 表示唯一外部依赖接口。**pathfinder 为 Go 项目**；命名与 zeroclaw 对齐处单独标注。

**当前 pathfinder 已有**：`internal/app`、`internal/config`、`internal/provider`。配置路径：`PATHFINDER_CONFIG_DIR` 或 `PATHFINDER_WORKSPACE` 或 `~/.pathfinder`；workspace 子目录为 `workspace`。

```
pathfinder/
├── cmd/                        # 入口与支撑模块
│   ├── pathfinder/            # 主入口（CLI：pathfinder -m "目标"、run --message）
│   │   └── main.go
│   └── tui/                   # 支撑：TUI 入口（可选独立二进制或由主入口拉起）
│       └── main.go
├── internal/
│   ├── app/                   # [已有] 运行入口协调（Run(message)，加载 config、后续对接编排与 TUI）
│   │   └── run.go
│   ├── config/                # [已有] 配置加载与路径解析（PATHFINDER_CONFIG_DIR / PATHFINDER_WORKSPACE / ~/.pathfinder）
│   │   ├── config.go
│   │   └── (workspace、.env、config.toml)
│   ├── provider/              # [已有] LLM/推理提供方（factory、compatible、credential、types）；与 zeroclaw providers 对齐
│   │   ├── provider.go
│   │   ├── factory.go
│   │   ├── compatible.go
│   │   ├── credential.go
│   │   ├── types.go
│   │   ├── provides.go
│   │   └── needs.go
│   ├── agent/                 # [规划] 执行体发现、派发、循环；与 zeroclaw agent 对齐
│   │   ├── agent.go
│   │   ├── dispatcher.go
│   │   ├── loop.go
│   │   ├── provides.go
│   │   └── needs.go
│   ├── skills/                # [规划] Skill 加载、审计、目录；与 zeroclaw skills 对齐
│   ├── tools/                 # [规划] Tool 实现与规范；与 zeroclaw tools 对齐
│   ├── planning/              # [规划] 计划结构、校验、state 约定
│   ├── runtime/               # [规划] 执行环境（native/docker 等）；与 zeroclaw runtime 对齐
│   ├── gateway/               # [规划] HTTP/API、SSE/WS 流式；与 zeroclaw gateway 对齐
│   ├── channels/              # [规划] 消息渠道（Slack、Discord、Telegram 等）；与 zeroclaw channels 对齐
│   ├── progress/              # [规划] 任务进度、checkpoint、恢复
│   ├── skillforge/            # [规划] Skill 发现、评估、集成；与 zeroclaw skillforge 对齐
│   ├── orchestration/         # [规划] 端到端工作流编排、总结
│   ├── memory/                # [规划] 记忆/上下文存储；与 zeroclaw memory 对齐（可选）
│   ├── observability/         # [规划] 支撑：日志、指标、追踪
│   ├── auth/                  # [规划] 支撑：鉴权
│   ├── health/                # [规划] 支撑：健康检查
│   ├── cost/                  # [规划] 支撑：用量与成本追踪
│   └── infra/                 # [规划] 持久化、外部客户端、适配器（实现各包 needs）
│       ├── persistence/
│       ├── clients/
│       └── adapters/
└── (可选) pkg/ 或 各包内      # 对外可复用包（若有）
```

**与 zeroclaw 术语对应**：agent、skills、tools、runtime、gateway、channels、skillforge、provider（pathfinder 用单数 provider）、memory、config。progress、planning、orchestration 为 pathfinder 工作流特有。支撑模块 TUI/CLI 在 cmd/；Observability、Auth、Health、Cost 在 internal 下独立包或合并到 gateway/config。

**目录优化要点**（对齐 ddd-cleancode-dir）：  
- **入口与编排分离**：入口 = cmd/pathfinder（读参数、调 app）与 cmd/tui；编排 = orchestration（串起 planning → agent → progress → …），app 仅做「加载 config → 调 orchestration → 可选拉 TUI」，不承载用例细节。  
- **变化点独立**：会换实现（provider、channels、持久化）、被多处用（config、progress、gateway）、或即将变重（agent）的包单独保留；observability/auth/health/cost 在代码量小时可合并到 gateway 或 config，待需要再拆。  
- **大包分子包**：agent 变重时按子职责拆为 agent/loop、agent/context、agent/tools 等，子包名仍用领域概念，不按「层」拆。  
- **实现归属**：各能力包内实现本包 Port（如 provider 下 openai/compatible）；跨多包的存储/客户端放 infra；编排处（app 或 gateway 组装）new 实现并注入，不散落 import 具体实现。

## 4.2 依赖（Dependencies）

### 4.2.1 主流程与调用链

主流程（简化）：**用户提交目标** → CLI 调 app.Run → **加载 config** → **orchestration.SubmitGoal** → 创建 run、产出 plan → **planning** 产出子任务与依赖 → **progress** 写入/恢复状态 → **agent** 派发执行、**provider** 调 LLM、**skillforge** 缺能力时生成 → **gateway** 推送流式/SSE → **channels** 可选交付结果 → **orchestration.SummarizeRun** → 结束。

**谁调谁**（单向、无循环）：

```
cmd/pathfinder, cmd/tui
    → app
app
    → config, orchestration
orchestration
    → planning, agent, progress, provider, gateway(流式/取消), channels(交付), skillforge, runtime(执行环境), memory(可选)
agent
    → provider, skills, tools, progress(写结果), planning(读 plan), runtime(执行)
gateway
    → progress(读进度), orchestration(取消), config
planning
    → (无内部包依赖；needs 可为 LLM/planner 由 infra 实现)
progress
    → (needs: 存储由 infra 实现)
skillforge
    → skills, config
channels
    → config, (needs: 各渠道实现由 infra 或 channels 子包)
```

- **无循环**：orchestration 不依赖 gateway；gateway 依赖 orchestration 仅用于「取消」等控制，不反向执行业务。  
- **归属**：流式输出、取消由 gateway 暴露，背后数据来自 progress / orchestration；编排逻辑只在 orchestration，app 只做「调 orchestration + 可选起 TUI」。

### 4.2.2 各包 provides / needs 摘要

| 包 | provides（对外接口） | needs（依赖外部，由 infra 或他包实现） |
|----|----------------------|----------------------------------------|
| **config** | Load(), 路径解析 | 无（读文件/环境变量可包内实现） |
| **provider** | Provider 接口、Factory | 无或 HTTP 客户端（可包内） |
| **app** | Run(message) | config.Load, orchestration.SubmitGoal / 可选 TUI 启动 |
| **orchestration** | SubmitGoal, ExecutePlan, SummarizeRun, CancelRun | planning, agent, progress, gateway(Stream), channels(Deliver), skillforge, provider, memory, config |
| **planning** | Plan 结构、Validate、产出 SubTask/Dependency | 规划器（LLM/planner）由 infra 实现时在 needs |
| **agent** | 派发、循环、执行体发现 | provider, skills, tools, progress, planning(读 plan), runtime(执行环境) |
| **progress** | WriteProgress, ReadProgress, Checkpoint, Restore | 持久化存储（infra） |
| **gateway** | HTTP/SSE/WS 入口、Stream(run_id)、Cancel(run_id) | progress, orchestration.Cancel, config |
| **channels** | SendMessage, Deliver 等 | 各渠道实现、config |
| **skillforge** | Scout, Evaluate, Integrate | skills(登记), config |
| **skills** | 加载、审计、List | 文件系统/ClawHub（needs 或包内） |
| **tools** | Tool 规范、执行 | 无或由调用方注入 |
| **runtime** | 执行环境（native/docker） | 无或由 infra 实现具体运行时 |
| **memory** | 记忆/上下文存储 | 存储后端（infra） |
| **infra** | 无（仅实现） | 实现各包 needs：持久化、HTTP 客户端、各 channel 实现等 |

- **编排处**：app 或 gateway 的「组装」只依赖各包 **provides（接口）**，具体实现（如某 Provider 实现、某 Channel 实现）在组装处构造并注入，不散落各处 import 实现类。

### 4.2.3 依赖原则与检查清单

- **编排包（app / gateway）**：只依赖各包的接口与领域类型，不依赖具体实现类的构造细节；实现由构造处 new 并注入。  
- **能力包（agent、planning、progress、provider、channels…）**：不依赖 app、不依赖 gateway（除 gateway 作为「调用方」时的接口）；可依赖他包 **provides**；本包 **needs** 由 infra 或上层注入。  
- **实现类**：放在对应能力包内（如 provider 下）或 infra；实现 Port，由 app/gateway 或测试注入；类名用业务含义，避免 XxxImpl。  
- **无循环**：依赖图无环；若 A 调 B、B 又调 A，则合并或抽出第三包/用事件或 Port 反转。  
- **耦合**：仅保留「领域耦合」（A 因能力调 B 的接口）；避免 pass-through（只为下游传数据）、common（多包共享同一表）、content（直接碰他包内部数据结构）。

## 4.3 支撑模块（Support Modules）

支撑模块不承载核心编排/规划/执行逻辑，但为使用流程、运维与可观测性提供能力；实现时可单独目录或归入现有包。

| 支撑模块 | 职责 | 依赖/对接 | zeroclaw 参考 |
|----------|------|-----------|---------------|
| **TUI** | 本地终端 UI：绑定 run_id，订阅进度与流式输出；展示当前阶段（规划/执行/总结）、当前步骤与 agent、子任务进度（如 3/7）、流式日志；支持取消与查看最终结果。 | gateway（SSE/流式）、progress（进度）、orchestration（取消） | — |
| **CLI** | 命令行入口：`pathfinder -m "目标"` / `run --message "..."` 提交目标，可选启动 TUI。 | orchestration（SubmitGoal）、TUI（可选） | channels/cli |
| **Observability** | 日志、指标、分布式追踪；便于排障与成本/延迟分析。 | 各包埋点、config | observability/ |
| **Auth** | API、渠道、Gateway 鉴权与身份解析。 | gateway、channels、config | auth/ |
| **Health** | 健康检查端点（依赖就绪、存储连通等）。 | config、persistence、providers | health/ |
| **Cost** | 用量与成本追踪（token、调用次数、按 provider 汇总）。 | providers、orchestration/agent 调用链 | cost/ |
| **Delivery / Reply** | 结果交付：回复到指定 channel、webhook 回调、写知识库等。 | orchestration（总结产出）、channels、config | channels 的 deliver/reply 能力 |
| **Config** | 配置加载、校验、热更新（已列入目录）。 | — | config/ |

**说明**：TUI、CLI 对应 FEATURES 使用流程「直接发布 + TUI」，落地于 `cmd/tui`、`cmd/pathfinder`；Observability、Auth、Health、Cost、Delivery 在 internal 下独立包或合并到 gateway/config，pathfinder 可按需复用 zeroclaw 或实现简化版。

### 4.4 配置精简（避免过度设计）

pathfinder 当前为单用户目标工作流编排，配置仅保留工作流所需项，以下视为过度设计已移除或不做：

| 移除/不做 | 原因 |
|-----------|------|
| **active_workspace.toml** | 多 workspace 持久化切换；单用户用 PATHFINDER_CONFIG_DIR / PATHFINDER_WORKSPACE 或默认 ~/.pathfinder 即可。 |
| **config.toml 中的 api_key / api_url** | 隐私变量统一用 .env，不落盘 config.toml；APIKey/APIURL 仅从环境变量（含 .env）注入。 |
| **extra_headers** | 当前无按厂商自定义 HTTP 头需求；provider 层可选支持，config 不承载。 |

保留：路径解析（PATHFINDER_CONFIG_DIR > PATHFINDER_WORKSPACE > ~/.pathfinder）、default_provider / default_model / default_temperature、provider_timeout_secs、.env 加载。

---

# 第5部分：领域建模

## 5.1 领域元素提取（名词/动词/规则/状态）

### 名词清单（核心业务概念）

- **核心概念**：Run、Plan、SubTask、Dependency、SuggestedAgent、Agent、Skill、Tool、AgentPool、TaskProgress、RunProgress、Checkpoint、SkillPackage、ToolSpec、GenerationRequest、Summary、Report。
- **参与者**：用户、编排层、执行层、Agent、监督者。
- **属性/值**：run_id、task_id、status、agent_id、started_at、result、deadline、cancel_requested、priority、progress_percent。
- **时间/地点**：started_at、finished_at、created_at；channel（消息渠道）、workspace（skills 目录）。

### 动词清单（行为）

- 用户/系统：提交目标、取消 run、订阅进度。
- 编排层：分解目标、分析依赖、分派建议 agent、写入 state、检测缺失、路由到生成、汇总结果、登记 Skill/Tool。
- 执行层：派发子任务、写回结果。
- 运行时：创建 run、派发到 agent、流式输出、取消 run。
- 执行状态：写入任务进度、checkpoint、恢复、读取进度。
- 能力目录：列出 agent、查询能力、登记/安装 Skill/Tool。

### 条件规则清单（摘要）

- 前置：目标描述非空；Run 创建成功才有 run_id；Plan 产出后才有子任务列表与依赖。
- 后置：子任务按 DAG 执行；执行结果写回 state；取消/超时/失败策略满足后进入中止。
- 业务规则：超时后不再执行新子任务；cancel_requested 后路由到中止；失败次数超过阈值则中止；规划结果须含子任务、依赖、建议 agent。

### 状态变化记录（摘要）

- **Run**：不存在 → 已创建 → 规划中 → 执行中 → 已中止/已完成。
- **TaskProgress**：待办 → 进行中 → 已完成/失败/已跳过。
- **Plan**：无 → 已产出（含 SubTask 列表、Dependency、SuggestedAgent）。

## 5.2 名词分类（实体 vs 值对象）

- **实体（有身份、生命周期）**：Run、Plan、SubTask、Agent、TaskProgress、RunProgress、GenerationRequest。
- **值对象（由属性定义、不可变）**：Dependency、SuggestedAgent、AgentId、SkillId、ToolId、Checkpoint、SkillPackage、ToolSpec、Summary、Report、AgentPoolFilter。

## 5.3 聚合设计（事务边界）

| 聚合根 | 边界内对象 | 职责概要 | 关键不变条件 |
|--------|------------|----------|--------------|
| **Run** | Run, RunProgress 引用 | run 生命周期、取消标志、deadline | 取消后不可再派发新任务；超时后不可再执行新子任务 |
| **Plan** | Plan, SubTask[], Dependency | 计划结构、校验 | 子任务 id 唯一；依赖引用同一 Plan 内 task_id |
| **TaskProgress** | TaskProgress（按 run_id+task_id） | 单任务进度、结果 | 状态仅允许约定转换（待办→进行中→完成/失败） |
| **Agent** | Agent, Skill[], Tool[]（目录视角） | 能力目录内「谁有什么」 | 仅通过登记/安装更新能力，不在此聚合内执行 |

- **关联**：Run 引用 Plan（by ref 或 snapshot）；Run 与 TaskProgress 按 run_id 关联；Dispatch 通过 AgentId 引用 CapabilityCatalog 的 Agent。

## 5.4 行为分配（实体 vs 领域服务）

- **Run**：Create、Cancel、MarkAborted、IsCancelRequested、IsOverDeadline。
- **Plan**：Validate、SubTasks、Dependencies、SuggestedAgentFor(task_id)。
- **TaskProgress**：Start、Complete、Fail、WriteResult；领域服务 **ProgressMaintainer**：BatchUpdateProgress、Checkpoint、Restore。
- **能力目录**：领域服务 **AgentDiscovery**：ListAgents(filter)、GetAgent(id)；**CapabilityChecker**：HasSkill(agent, skill)、HasTool(agent, tool)、ListSkills(agent)；**SkillToolRegistry**：RegisterSkill、InstallSkill、RegisterTool（接口在领域层，实现在基础设施层）。
- **能力生成**：领域服务 **SkillGenerator**、**ToolGenerator**（生成规范包/spec）；安装/注册调用 CapabilityCatalog 的 Registry 与外部 ClawHub/框架适配器）。
- **派发**：领域服务 **Dispatcher**：Dispatch(run_id, task, agent_id)（接口在领域层，实现在基础设施层，调 openclaw/acpx）。

## 5.5 规则与约束建模（摘要）

- **校验规则**：Plan 必须包含至少一个 SubTask；SuggestedAgent 必须在能力目录可发现；Run 的 deadline 必须大于当前时间（创建时）。
- **状态/流程规则**：TaskProgress 仅允许 待办→进行中→已完成/失败；Run 在 cancel_requested 或超时或失败策略满足后只能进入中止分支。
- **策略规则**：失败策略（如 max_retries、风险阈值）由监督节点/应用层配置驱动，领域层提供「是否应中止」的判定接口。

---

# 第6部分：应用层设计

## 6.1 设计矩阵（用例 → Command/Query → 应用服务方法 → DTO/事件）

| 用例 | 命令/查询对象 | 应用服务与方法 | 返回 DTO / 事件 |
|------|----------------|----------------|-----------------|
| SubmitGoalUseCase | SubmitGoalCommand | WorkflowOrchestrationApplicationService.SubmitGoal() | RunDTO（run_id, stream_handle）, RunCreatedEvent |
| SubmitGoalViaChannelUseCase | SubmitGoalViaChannelCommand | WorkflowOrchestrationApplicationService.SubmitGoalViaChannel() | RunDTO, RunCreatedEvent |
| PlanGoalUseCase | PlanGoalCommand | PlanningApplicationService.PlanGoal() | PlanDTO, PlanProducedEvent |
| ExecutePlanUseCase | （内部编排） | WorkflowOrchestrationApplicationService.ExecutePlan() | RunSummaryDTO, RunCompletedEvent / RunAbortedEvent |
| DispatchTaskToAgentUseCase | DispatchTaskCommand | RuntimeApplicationService.DispatchTask() | DispatchResultDTO |
| MaintainTaskProgressUseCase | UpdateTaskProgressCommand | ExecutionStateApplicationService.UpdateTaskProgress() | — |
| CancelRunUseCase | CancelRunCommand | RuntimeApplicationService.CancelRun() | —, RunCancelledEvent |
| AbortRunOnConditionUseCase | （监督节点调用） | ExecutionStateApplicationService.EvaluateAbortCondition() | bool + 原因 |
| SummarizeRunUseCase | SummarizeRunCommand | WorkflowOrchestrationApplicationService.SummarizeRun() | SummaryDTO |
| RegisterSkillOrToolToCatalogUseCase | RegisterSkillOrToolCommand | CapabilityCatalogApplicationService.RegisterSkillOrTool() | — |
| ListAgentsUseCase | ListAgentsQuery | CapabilityCatalogQueryService.ListAgents() | AgentListDTO |
| CheckAgentCapabilityUseCase | CheckAgentCapabilityQuery | CapabilityCatalogQueryService.CheckAgentCapability() | CapabilityCheckDTO |
| StreamRunProgressUseCase | StreamRunProgressQuery | RuntimeQueryService.StreamRunProgress() | 流式事件（SSE/WS） |

## 6.2 应用服务接口（按聚合/上下文）

- **WorkflowOrchestrationApplicationService**：SubmitGoal(ctx, cmd) (*RunDTO, error)；SubmitGoalViaChannel(ctx, cmd) (*RunDTO, error)；ExecutePlan(ctx, runID) error；SummarizeRun(ctx, runID) (*SummaryDTO, error)。
- **PlanningApplicationService**：PlanGoal(ctx, cmd) (*PlanDTO, error)。
- **RuntimeApplicationService**：DispatchTask(ctx, cmd) (*DispatchResultDTO, error)；CancelRun(ctx, runID) error。
- **RuntimeQueryService**：StreamRunProgress(ctx, runID) (Stream, error)。
- **ExecutionStateApplicationService**：UpdateTaskProgress(ctx, cmd) error；EvaluateAbortCondition(ctx, runID) (shouldAbort bool, reason string, err error)。
- **CapabilityCatalogApplicationService**：RegisterSkillOrTool(ctx, cmd) error。
- **CapabilityCatalogQueryService**：ListAgents(ctx, query) (*AgentListDTO, error)；CheckAgentCapability(ctx, query) (*CapabilityCheckDTO, error)。

## 6.3 Command/Query 与 DTO（摘要）

- **SubmitGoalCommand**：GoalDescription, Timeout, Priority, AgentPoolID（可选）。
- **RunDTO**：RunID, StreamHandle, Status, CreatedAt。
- **PlanDTO**：PlanID, SubTasks[], Dependencies[], SuggestedAgents（task_id → agent_id）。
- **DispatchTaskCommand**：RunID, TaskID, AgentID, TaskDescription, Context。
- **UpdateTaskProgressCommand**：RunID, TaskID, Status, AgentID, StartedAt, Result。
- **ListAgentsQuery**：Filter（AgentPoolID, CapabilityTags）。
- **AgentListDTO**：Agents[]（ID, Name, Capabilities, Tags）。

## 6.4 领域事件（摘要）

- RunCreatedEvent、RunCancelledEvent、RunCompletedEvent、RunAbortedEvent、PlanProducedEvent、TaskProgressUpdatedEvent、SkillRegisteredEvent、ToolRegisteredEvent。

## 6.5 应用层目录结构（与第4部分一致）

- 用例编排、Command/DTO/事件由各职责包内 `service.go` 与同包下的 commands/dtos/events 文件承载，不单独设 application 子目录。

---

# 基础设施层（Infra）原则

- **端口定义位置**：各职责包内 `needs.go` 定义所需接口（如 AgentDiscovery、Dispatcher、SkillToolRegistry、ProgressRepository）；infra 实现这些接口。
- **目录组织**：按技术栈分 `clients/`（Gateway、ClawHub、acpx）、`persistence/`（Run、Plan、TaskProgress、Checkpoint）、`adapters/`（按职责包对应，如 `adapters/agent/`、`adapters/runtime/`）；与 zeroclaw 一致处：gateway、config、channels、providers 等按职责独立目录；避免顶级 `services/` 目录；命名体现技术实现（如 `task_progress_repository_sql`、`dispatcher_openclaw`）。
- **适配器前提**：仅当某能力存在多种可替换实现且含适配逻辑（如模型转换、多系统组合）时，将实现放在 `adapters/` 下；单一实现可放在 `clients/` 或 `persistence/`。
- **与 FEATURES 八对应**：能力目录端口 → M1 Agent 发现 + M2 Skill/Tool 目录；规划契约 → M5；运行时端口 → M3；执行状态端口 → M4。

---

## 文档修订

- 第1部分输入自 FEATURES.md、设计文档；第2–3部分按 ddd-2-1、ddd-3-1 执行；第5部分按 ddd-5-1、5-2-1～5-2-4 精简落地；第6部分按 ddd-6-1；Infra 按 ddd-6-3。
