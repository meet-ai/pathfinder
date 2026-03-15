# Multi-Agent Goal Workflow — Feature List

> Aligned with the workflow design; for product/engineering tracking and acceptance.  
> Each item is deliverable and testable; grouped by workflow phase and capability.

**Languages:** [English](FEATURES.en.md) · [中文](FEATURES.md)

---

## Flow (one option): Direct publish + TUI

**Contract**: User publishes one task via CLI; local TUI shows execution and results.

| Step | User action | System behavior |
|------|-------------|-----------------|
| 1 | Run `pathfinder -m "goal description"` (or `run --message "..."`) | Submit goal to orchestration, get run_id/session; process stays up. |
| 2 | — | Start **TUI** bound to run_id, subscribe to progress and stream. |
| 3 | View in TUI; cancel if needed | Show: current phase (plan/execute/summarize), current step and agent, subtask progress (e.g. 3/7), streamed log; support cancel. |
| 4 | After completion or cancel, view result | TUI shows final summary or error; process exits after TUI closes. |

**Acceptance**: One command publishes task; TUI opens and shows phase and progress in real time; user can cancel or see final result in TUI.

---

## 1. Task publish and entry

| ID | Description | Notes / acceptance |
|----|-------------|--------------------|
| F1.1 | **Unified API entry** to submit high-level goal and start workflow | HTTP/API (e.g. LangGraph invoke, OpenClaw agent --message); success returns run_id or stream handle. |
| F1.2 | **Message channels** to submit tasks (Chat, Slack, WebChat) | Bindings route user message to agent or orchestration entry; acceptance: sending on configured channel triggers task. |
| F1.3 | **Optional parameters** on submit (timeout, priority, agent pool) | Entry accepts extra params, written to state/config for later nodes; acceptance: params affect plan/execute/cancel. |

---

## 2. Planning and exploration

| ID | Description | Notes / acceptance |
|----|-------------|--------------------|
| F2.1 | **Task decomposition** of user goal into structured subtask list | Planning node/agent calls LLM/planner to split goal; acceptance: given a goal, output subtask list (title/description). |
| F2.2 | **Dependency analysis** for subtasks → DAG or execution order | Dependencies (e.g. A→B→C or A,B parallel then C); acceptance: output consumable by execution (edges or ordered list). |
| F2.3 | **Suggested agent** per subtask | Each subtask has suggested agent (e.g. frontend-developer, security-engineer); acceptance: execution node can choose openclaw:<agent-id> or acpx. |
| F2.4 | **Plan written to orchestration state** for execution and supervisor | Subtasks, dependencies, suggested agents in LangGraph state or equivalent; acceptance: execution and supervisor nodes can read and schedule. |

---

## 3. Execution and dispatch

| ID | Description | Notes / acceptance |
|----|-------------|--------------------|
| F3.1 | **Serial execution** of subtasks by dependency order | Topological order; acceptance: order matches DAG, previous result available to next. |
| F3.2 | **Parallel execution** for independent subtasks | Same “layer” via parallel nodes or branches; acceptance: concurrent execution, results merged into state. |
| F3.3 | **Dispatch to agent** per state | Via openclaw:<agent-id>, openclaw-sdk or acpx; acceptance: task runs on intended agent and returns result. |
| F3.4 | **Write execution result to state** for summarize and conditional edges | Each execution node writes result (success/fail, output) to state; acceptance: summarize and conditionals can read. |

---

## 4. On-demand Skill / Tool generation

| ID | Description | Notes / acceptance |
|----|-------------|--------------------|
| F4.1 | **Detect missing Skill** and route to generation | Execution or check node sets state “need Skill” and routes to generation; acceptance: no deadlock when Skill missing. |
| F4.2 | **Generate Skill package** (e.g. SKILL.md, layout) | Generation node/agent produces Skill files; acceptance: output matches OpenClaw/Cursor format. |
| F4.3 | **Install or deploy Skill** to target agent (clawhub install / copy to skills) | After generation, install or copy to workspace/skills; acceptance: target agent can use Skill next run. |
| F4.4 | **Detect missing Tool** and route to generation | Same as F4.1 for Tool; acceptance: route to Tool generation when missing. |
| F4.5 | **Generate Tool and register** to framework, bind to agent | Generation produces Tool code or OpenAPI schema, registers to OpenClaw/agency-swarm/CrewAI, binds to agent; acceptance: agent can call new Tool next run. |
| F4.6 | **Re-schedule after generation** (retry or new subtask) | State updated when Skill/Tool ready, route back to execution; acceptance: retry or new task uses new capability. |

---

## 5. Progress and cancellation

| ID | Description | Notes / acceptance |
|----|-------------|--------------------|
| F5.1 | **Task-level progress** (done/in progress/todo), persist | state or store: task_id → status, agent, started_at, result; checkpoint; acceptance: resume after interrupt reflects progress. |
| F5.2 | **Stream and progress events** (current step, agent, progress%) | LangGraph stream, SSE or WebSocket; acceptance: client sees progress in real time. |
| F5.3 | **Timeout and auto-abort** | Per-task or whole run deadline; route to abort node on timeout; acceptance: no new subtasks after timeout, cleanup and summarize. |
| F5.4 | **User/supervisor cancel** and cleanup | cancel_requested or equivalent; supervisor or conditional routes to abort; acceptance: cancel leads to termination and usable summary. |
| F5.5 | **Failure/risk policy**: abort on too many failures or high risk | state or supervisor decides from failure count, error type; acceptance: when condition met, abort and summarize, no infinite retry. |

---

## 6. Summarize and capture

| ID | Description | Notes / acceptance |
|----|-------------|--------------------|
| F6.1 | **Aggregate subtask results** into final report or structured output | Summarize node collects from state; acceptance: user or downstream gets full output. |
| F6.2 | **Write final output to knowledge base** or deliver | Optional: write report to KB, upload, or API callback; acceptance: when configured, output reaches target. |
| F6.3 | **Optional retro node**: short summary or recommendations | Optional: before summarize, analyze success/failure; acceptance: when enabled, retro output produced. |
| F6.4 | **Register new Skill/Tool** to catalog or ClawHub | From 4a/4b; optional at end of run; acceptance: registered Skill/Tool discoverable and reusable. |

---

## 7. Architecture and integration

| ID | Description | Notes / acceptance |
|----|-------------|--------------------|
| F7.1 | **Orchestration vs execution**: orchestration (LangGraph/CrewAI) calls execution (OpenClaw/acpx) | Docs and examples show layering; execution implementation replaceable. |
| F7.2 | **Configurable/extensible nodes** (plan, execute, generate, supervise, summarize) | At least one way to replace or extend each node. |
| F7.3 | **OpenClaw Gateway, ClawHub, Cursor acpx** integration docs and examples | How to use openclaw:<agent-id>, clawhub, acpx with orchestration; acceptance: doc allows end-to-end setup. |
| F7.4 | **“OpenClaw only” mode**: planning Agent + sessions_send/spawn subset without LangGraph | Doc and examples for what’s possible and limits; acceptance: capability vs tradeoff list. |

---

## 8. Supporting modules (by DDD)

By bounded context; each module single responsibility.

### 8.1 Capability catalog

**Responsibility**: “Who is available, what can they do”; discovery, lookup, registration.

| Submodule | Brief | Feature deps |
|-----------|-------|--------------|
| **Agent discovery** | List and metadata (id, description, tags/role); filter by pool/capability; Gateway/ClawHub/local config. | F1.3, F2.3, F3.3, F4.3/F4.5, F7.3/F7.4 |
| **Skill/Tool catalog and existence** | Query agent Skill/Tool; list and search; register/install (ClawHub install, copy to skills). | F4.1, F4.2/F4.3, F4.4/F4.5, F6.4 |

**Ports**: ListAgents(filter), GetAgent(id), HasSkill(agent, skill), ListSkills(agent), RegisterSkill/InstallSkill; symmetric Tool APIs.

### 8.2 Planning contract

**Responsibility**: Structure and validation of plan output; read/write contract with orchestration state.

| Submodule | Brief | Feature deps |
|-----------|-------|--------------|
| **Plan result contract** | Subtask list, dependencies (DAG/order), suggested agents; state read/write. | F2.1, F2.2, F2.3, F2.4 |

**Ports**: PlanSchema (subtasks, dependencies, suggested agents), Validate(plan), state contract (match orchestration).

### 8.3 Runtime

**Responsibility**: Run lifecycle, dispatch to executor, stream, cancel.

| Submodule | Brief | Feature deps |
|-----------|-------|--------------|
| **Runtime session and dispatch** | Create run, run_id/stream handle; dispatch subtask to agent (openclaw:\<id\>/acpx); stream/SSE; cancel run. | F1.1, F3.3, F5.2, F5.4 |

**Ports**: CreateRun(params), Dispatch(run_id, task, agent_id), Stream(run_id), Cancel(run_id).

### 8.4 Execution state

**Responsibility**: Task progress, checkpoint, persist and restore; for supervisor and summarize.

| Submodule | Brief | Feature deps |
|-----------|-------|--------------|
| **Progress and state persistence** | task_id → status, agent, started_at, result; checkpoint/restore. | F2.4, F3.4, F5.1, F6.1 |

**Ports**: WriteTaskProgress(run_id, task_id, status, result), ReadProgress(run_id), Checkpoint(run_id), Restore(run_id).

---

## Usage

- **ID**: For reference in Issue, Project or requirements (e.g. `F3.2`).
- **Acceptance**: Can be used as DoD or test input.
- **Priority and iteration**: Implement by phase (e.g. 1→2→3→5 first, then 4a/4b and 4c).
