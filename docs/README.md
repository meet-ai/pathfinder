# Pathfinder docs index

> Use this page to find design, architecture, or maintenance docs. It is the navigation and “what lives where” map.

**Languages:** [English](README.md) · [中文](README.zh-CN.md)

[Design doc](design/多智能体目标工作流设计.md) · [Features](../FEATURES.md) · [Doc maintenance](#doc-maintenance--github)

---

## Docs by purpose

### Getting started

- [Multi-agent goal workflow design](design/多智能体目标工作流设计.md) — Five phases, mapping to existing frameworks, recommended architecture, OpenClaw comparison.
- [Features (FEATURES)](../FEATURES.md) — Deliverables with IDs, descriptions, acceptance criteria, for Issue/Project.

### Design

- [Multi-agent goal workflow design](design/多智能体目标工作流设计.md) — Full flow: publish → plan → execute → Skill/Tool generation and progress/cancel → summarize.
- [Phases and framework mapping](design/多智能体目标工作流设计.md#四各阶段与框架对照小结) — Quick reference table.
- [OpenClaw-only feasibility](design/多智能体目标工作流设计.md#五仅用-openclaw-能否直接实现) — Capability comparison and conclusion.

### Architecture and integration

- [Recommended architecture](design/多智能体目标工作流设计.md#三推荐整体架构对齐你的-15) — Orchestration (LangGraph) and execution (OpenClaw/acpx).
- [F7.x: Architecture and integration](../FEATURES.md#七架构与集成) — Separation of concerns, OpenClaw/ClawHub/acpx, “OpenClaw only” mode.

### Implementation and selection

- [Zeroclaw provider analysis](zeroclaw-provider-analysis.md) — Provider abstraction, factory, OpenAI-compatible layer, Router/Reliable, reference for Go.
- [Provider migration design](design/provider-migration-design.md) — Full migration from Zeroclaw: types, interfaces, OpenAI compat, factory and credentials, Router/Reliable, layout and checklist.

### Doc maintenance

- [Doc maintenance](#doc-maintenance--github) — Where docs live, naming, Git/Issue/Project, optional GitHub Pages/MkDocs.

---

## External references

- **Zeroclaw:** [Zeroclaw GitHub](https://github.com/zeroclaw-labs/zeroclaw) · [Zeroclaw 中文](https://zeroclaws.io/zh/) — Provider, Agent, Skills, Tools, Runtime, Gateway, Channels (aligned with pathfinder).
- [OpenClaw docs](https://docs.openclaw.ai/) — Gateway, Skill, Session Tools.
- [OpenClaw Session Tools](https://docs.openclaw.ai/concepts/session-tool) — `sessions_send`, `sessions_spawn`.
- [OpenClaw GitHub README](https://github.com/openclaw/openclaw) — Project structure and doc layout.

---

## Doc maintenance — GitHub

How to organize and present docs on GitHub and how they fit with Git, Issue, and Project.

### Where to put docs

- **Repo root:** `README.md` (project intro and doc entry), `FEATURES.md` (feature list).
- **`docs/`:** This index `docs/README.md`; design docs in `docs/design/` (e.g. workflow design, optional `architecture-overview.md`, `phases-summary.md`).
- **GitHub Wiki** (optional): User-facing how-to; keep design and FEATURES in the repo for versioning and review.

The root `README.md` should have a “Docs” section linking to this index, the design doc, and `FEATURES.md`.

### Naming and layout

| Path | Content |
|------|---------|
| `docs/design/多智能体目标工作流设计.md` | Full design. |
| `docs/design/architecture-overview.md` | Simplified architecture and stack (optional). |
| `docs/design/phases-summary.md` | Phase vs framework table (optional). |
| `FEATURES.md` (root) | Feature list, IDs e.g. F2.1. |

### Git workflow

- **Branches:** Docs follow `main` or feature branches; large doc changes can go in a `feat/xxx` PR with code.
- **Tags:** Tag releases on commits that include current design and features; release notes can list implemented feature IDs.
- **Issue/Project:** Reference `FEATURES.md` IDs (Fx.x) in issues; Project columns for backlog/in progress/done; optional “Feature ID” in issue template.

### Optional enhancements

- **GitHub Pages:** Publish `docs/` or a chosen directory as a static site.
- **MkDocs / Diátaxis:** Use `docs/` as source, with navigation and search; deploy to GitHub Pages.
- **Diagrams:** Put PNG/SVG in `docs/design/assets/` and reference from docs; keep sources (Mermaid, draw.io) in the repo.

With this layout, docs are readable on GitHub via the repo + docs/ + this index, and stay aligned with features, issues, and releases.
