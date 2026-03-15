# Pathfinder 文档索引

> 需要查设计、架构或维护方式时，从下面入口找。本页是文档导航与「什么在哪里」。

[设计文档](design/多智能体目标工作流设计.md) · [特性列表](../FEATURES.md) · [文档维护说明](#文档维护说明--github-展示)

---

## 文档（按用途）

### 入门与总览

- [多智能体目标工作流设计](design/多智能体目标工作流设计.md) — 目标工作流五阶段、与现有框架的能力映射、推荐架构与 OpenClaw 能力对照。
- [特性列表（FEATURES）](../FEATURES.md) — 按阶段/能力域分组的可交付特性，带 ID、描述与验收标准，便于 Issue/Project 引用。

### 设计

- [多智能体目标工作流设计](design/多智能体目标工作流设计.md) — 全文：发布任务 → 规划探索 → 执行派发 → Skill/Tool 按需生成与进度/中止 → 整理总结。
- [各阶段与框架对照小结](design/多智能体目标工作流设计.md#四各阶段与框架对照小结) — 表格速查。
- [仅用 OpenClaw 能否实现](design/多智能体目标工作流设计.md#五仅用-openclaw-能否直接实现) — 能力逐项对照与结论。

### 架构与集成

- 设计文档内 [推荐整体架构](design/多智能体目标工作流设计.md#三推荐整体架构对齐你的-15) — 编排层（LangGraph）与执行层（OpenClaw/acpx）、简化架构图。
- [特性 F7.x：架构与集成](../FEATURES.md#七架构与集成) — 编排与执行分离、节点可扩展、OpenClaw/ClawHub/acpx 集成、「仅 OpenClaw」模式说明。

### 实现与选型

- [Zeroclaw Provider 实现分析](zeroclaw-provider-analysis.md) — zeroclaw 中 provider 抽象、工厂、OpenAI 兼容层、Router/Reliable 及对 Go 实现的参考。
- [Provider 迁移设计](design/provider-migration-design.md) — 从 zeroclaw 完整迁移的 provider 功能设计：类型、接口、OpenAI 兼容、工厂与凭证、Router/Reliable、包布局与迁移清单。

### 文档维护与 GitHub 展示

- 本页下方 [文档维护说明](#文档维护说明--github-展示) — 文档放在哪、命名与层级、与 Git/Issue/Project 的配合、可选展示增强（GitHub Pages、MkDocs 等）。

---

## 参考（外部）

- [OpenClaw 文档](https://docs.openclaw.ai/) — Gateway、Skill、Session Tools 等。
- [OpenClaw Session Tools](https://docs.openclaw.ai/concepts/session-tool) — `sessions_send`、`sessions_spawn`。
- [OpenClaw GitHub README](https://github.com/openclaw/openclaw) — 项目结构与文档组织可作参考。

---

## 文档维护说明 — GitHub 展示

以下说明在 GitHub 上如何组织与展示文档，以及如何与 Git 工作流、Issue/Project 配合。

### 一、文档应放在哪些位置

- **仓库根目录**：`README.md`（项目简介与文档入口）、`FEATURES.md`（特性列表）。
- **`docs/`**：本索引 `docs/README.md`；设计类放入 `docs/design/`（如 `多智能体目标工作流设计.md`、可选 `architecture-overview.md`、`phases-summary.md`）。
- **GitHub Wiki**（可选）：用户向/操作向内容；设计文档与特性列表建议保留在主仓便于版本与 Code Review。

根目录 `README.md` 中建议有「文档」小节，链接到本页、设计文档与 `FEATURES.md`。

### 二、文件命名与层级

| 建议路径 | 内容 |
|----------|------|
| `docs/design/多智能体目标工作流设计.md` | 设计全文。 |
| `docs/design/architecture-overview.md` | 简化架构图与推荐技术栈（可选）。 |
| `docs/design/phases-summary.md` | 各阶段与框架对照表格（可选）。 |
| `FEATURES.md`（根目录） | 特性列表，ID 如 F2.1。 |

### 三、与 Git 工作流的配合

- **分支**：文档随 `main` 或功能分支维护；大改可在 `feat/xxx` 上文档与代码一起 PR。
- **版本与 Tag**：按版本打 tag 时打在包含当前设计+特性的提交上；Release Notes 中可写「本版本实现的特性 ID」。
- **Issue/Project**：Issue 描述中引用 `FEATURES.md` 的 Fx.x；Project 列对应需求/进行中/已完成；可选 Issue 模板字段「对应特性 ID」。

### 四、可选的展示增强

- **GitHub Pages**：将 `docs/` 或指定目录发布为静态站。
- **MkDocs / Diátaxis**：用 `docs/` 作源目录，多级导航与搜索；可部署到 GitHub Pages。
- **架构图**：PNG/SVG 放 `docs/design/assets/`，在文档中相对路径引用；图源（Mermaid、draw.io）建议进仓。

按上述方式组织后，在 GitHub 上可通过「仓库 + docs/ + README 索引」直接阅读，并与特性列表、Issue、版本管理对齐。
