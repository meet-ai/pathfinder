# adapters

仅当某能力存在多种可替换实现且含适配逻辑（如模型转换、多系统组合）时，将实现放在本目录下按职责分子包，例如：

- `adapters/agent/` — 多种 AgentDiscovery/Dispatcher 实现
- `adapters/runtime/` — 多种执行环境适配

单一实现放在 `infra/persistence/` 或 `infra/clients/` 即可。
