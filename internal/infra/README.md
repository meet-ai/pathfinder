# infra

持久化、外部客户端、适配器（实现各包 needs）。

## 职责

- 实现各包 needs.go 中定义的端口
- 不对外提供业务接口；仅被 app/gateway 组装并注入

## 子目录与文件说明

| 路径 | 职责 | 文件 | 说明 |
|------|------|------|------|
| **persistence/** | 仓储实现 | run_repo_mem.go | RunRepository 内存实现 |
| | | plan_repo_mem.go | PlanRepository 内存实现 |
| | | task_progress_repo_mem.go | TaskProgressRepository 内存实现 |
| | | planner_stub.go | Planner 占位实现 |
| **clients/** | 外部客户端/占位 | agent_discovery_mem.go | AgentDiscovery 内存实现 |
| | | dispatcher_stub.go | Dispatcher 占位实现 |
| **adapters/** | 多实现适配器 | README.md | 仅当多套可替换实现且含适配逻辑时在此分子包 |

## 实现约定

- 实现类命名体现技术（如 run_repo_mem、dispatcher_stub），避免 XxxImpl 后缀
- 单一实现放 persistence/ 或 clients/；多实现适配放 adapters/ 下按职责分子包
