# Multi-Agent Goal Workflow Design

> Goal workflow: **publish task → plan & explore → parallel/serial execution → on-demand Skill/Tool generation → progress & cancellation → summarize**. Mapped to existing framework capabilities and implementation options.  
> Doc date: 2026-03

**Languages:** [English](workflow-design.md) · [中文](多智能体目标工作流设计.md)

---

## 1. Goal workflow (overview)

| Phase | Content |
|-------|---------|
| **1. Publish task** | User/system submits a high-level goal (e.g. “complete a requirements doc and propose an implementation”). |
| **2. Planning & exploration** | Decompose goal, analyze dependencies, explore feasible paths; produce a task DAG or execution plan (who does what, order/parallelism). |
| **3. Execution & dispatch** | Execute subtasks **in parallel or serial** per plan, assigning each to a **suitable agent** (e.g. frontend-developer, security-engineer). |
| **4a. Missing Skill** | If a required **Skill** is missing during execution, an agent **generates** it and delivers it to the right agent. |
| **4b. Missing Tool** | If a required **Tool** is missing, an agent **generates** it and delivers it to the right agent. |
| **4c. Progress & control** | **Progress**: continuous feedback; **response & feedback**; **cancel** unreasonable tasks (timeout, invalid, high risk). |
| **5. Summarize** | After completion: **organize outputs**, **capture lessons** (docs, knowledge base, register reusable Skill/Tool). |

---

## 2. Mapping to existing frameworks

### 2.1 Phases 1–3: Publish, plan, execute

| Phase | Framework support | Notes |
|-------|-------------------|------|
| **1. Publish task** | All frameworks | Entry = user message or API (LangGraph invoke, CrewAI kickoff, agency-swarm get_response, OpenClaw agent --message). |
| **2. Planning & exploration** | **LangGraph**, **CrewAI**, **AutoGen/AG2** | **LangGraph**: planning node + conditional edges, state holds plan, subtasks, dependencies. **CrewAI**: Task = subtask, Process defines order/hierarchy; add a “planning Agent” to produce Task list. **AutoGen/AG2**: Manager or Planner agent does planning via multi-turn dialogue. |
| **3. Parallel/serial execution, dispatch** | **LangGraph**, **CrewAI**, **agency-swarm**, **OpenClaw** | **LangGraph**: nodes = steps, call OpenClaw agent or acpx; graph expresses parallel/serial. **CrewAI**: Task.agent + Process. **agency-swarm**: communication_flows. **OpenClaw**: bindings; “dispatch” = call `openclaw:<agent-id>` or openclaw-sdk. |

### 2.2 Phases 4a / 4b: Generate and deliver Skill / Tool when missing

| Capability | Approach |
|-------------|----------|
| **Skill missing → generate and deliver** | **OpenClaw**: Skill has a fixed format (e.g. SKILL.md, directory); orchestration or a dedicated agent generates the Skill package, then ClawHub install or copy to agent workspace/skills; need a node/tool that can write files and run install. **Cursor**: Subagent capabilities live in prompt/agents; generation = new .md or rules on disk. Implement via LangGraph “skill generation” node or CrewAI Task: LLM + write files + `npx clawhub install` or equivalent. |
| **Tool missing → generate and deliver** | **OpenClaw / agency-swarm / CrewAI**: Tools are usually code (function/OpenAPI schema). Use a “tool generation” node/Agent to produce code or schema, write to project or register (e.g. agency-swarm BaseTool, CrewAI tools), then bind the tool to the target agent for the next run. **Cursor**: tool ≈ executable capability; generation = script or MCP config, loaded by acpx/OpenClaw. |

**Takeaway**: 4a/4b are rarely “one-click” in frameworks; design an **explicit step** (e.g. “check Skill/Tool exists → if not, enter generation subgraph → write/install → dispatch to agent”), implemented as a LangGraph subgraph or CrewAI hierarchical Process.

### 2.3 Phase 4c: Progress, feedback, cancellation

| Capability | Approach |
|-------------|----------|
| **Progress** | **LangGraph**: state holds “done / in progress / todo” subtasks; nodes update state; checkpoint for persistence. **CrewAI**: Task state; callbacks/events. **OpenClaw**: sessions on Gateway; combine with Dashboard or custom API. Maintain `task_id → status, agent, started_at, result` in orchestration. |
| **Feedback** | Streaming (LangGraph stream, CrewAI, OpenClaw) + progress events to frontend or logs; optional WebSocket/SSE for “current step, current agent, progress%”. |
| **Cancel / stop** | **LangGraph**: conditional edges or “supervisor” node route to “abort” node on timeout, error count, or user cancel; stop or skip later nodes. **CrewAI**: timeout, guardrails, or callback to interrupt. **OpenClaw**: session cancel; if using LangGraph, add a supervisor that checks and calls cancel. Keep “cancel_requested / timeout / max_retries” in state or store; check before each execution node and route to “cleanup & summarize” when needed. |

### 2.4 Phase 5: Summarize and capture lessons

| Capability | Approach |
|-------------|----------|
| **Organize outputs** | Final node/Task: aggregate subtask results from state, produce report, doc, or structured output; optionally write to knowledge base or hand off to user. |
| **Capture lessons** | Optional “retro” node: analyze success/failure, short summary or recommendations; **register new Skill/Tool** to catalog or ClawHub for reuse (persist 4a/4b). |

---

## 3. Recommended architecture (phases 1–5)

Keep the flow above; use existing components as follows.

### 3.1 Orchestration: LangGraph as controller

- **Entry**: User message or API triggers the **planning node**.
- **Planning node**: Call LLM (or OpenClaw planner agent) to decompose and explore; write “subtask list + dependencies + suggested agents” to state.
- **Execution**: **Serial**: conditional edges into “execution node” in dependency order. **Parallel**: same “layer” of subtasks via parallel nodes or branches.
- **Execution node**: From state’s “suggested agent”, call OpenClaw (`model: openclaw:<agent-id>` or openclaw-sdk) or acpx (`acpx <agent> "task"`); write result back to state.
- **Missing Skill/Tool**: When execution or a “check” node finds a missing Skill/Tool, set state to “need generation” and route to **generation node**. **Generation node**: Call “generation agent” (e.g. OpenClaw agent that writes Skill/Tool), write files and run install/register, update state “ready”, then route back to execution or new subtasks.
- **Progress & cancellation**: State holds `tasks_done`, `tasks_failed`, `cancel_requested`, `deadline`. **Supervisor** (or conditional edges): on timeout or `cancel_requested` or too many failures, route to **abort node** (cleanup, conclusion), then end. Stream/events: LangGraph stream or checkpoint callbacks emit “current step, progress, current agent” for frontend or logs.
- **Summarize node**: After all subtasks finish or abort, **summarize node** aggregates state, produces final output; optional “retro” to write summary and register new Skill/Tool.

So 1→2→3→4a/4b→4c→5 are expressed in **one graph**, easy to extend and debug.

### 3.2 Agent layer: OpenClaw + agency-agents / Cursor (acpx)

- **Plan / execute / generate Skill / generate Tool**: Different OpenClaw agents (e.g. `planner`, `frontend-developer`, `backend-architect`, `specialized-mcp-builder`); LangGraph nodes call via Gateway `openclaw:<agent-id>` or openclaw-sdk.
- **Coding tasks**: LangGraph node can call acpx (e.g. `acpx cursor exec "task"`) or an OpenClaw agent can call Cursor API via Skill/tool.
- **Deliver Skill/Tool to agent**: After generation, write to OpenClaw workspace skills or ClawHub, or project `.cursor/agents`; next run the agent can use it.

### 3.3 Simplified architecture (text)

```text
[User] publish task
    ↓
[LangGraph] planning node → explore & decompose → state: subtasks, dependencies, suggested agents
    ↓
[LangGraph] conditional edges: parallel branches | serial order
    ↓
[Execution node] pick agent from state → call OpenClaw(openclaw:<id>) or acpx
    ↓
  Missing Skill? → [Skill generation node] → write Skill, install → re-execute
  Missing Tool?  → [Tool generation node]  → write Tool, register  → re-execute
  Timeout/cancel? → [Supervisor] → [Abort node] → cleanup, conclusion
    ↓
[LangGraph] summarize node → organize output, capture lessons (optional register Skill/Tool)
    ↓
[User] result and feedback
```

---

## 4. Phase vs framework summary

| Phase | Recommended implementation |
|-------|----------------------------|
| 1. Publish task | Any framework entry (LangGraph invoke, CrewAI kickoff, OpenClaw agent --message, etc.). |
| 2. Planning & exploration | **LangGraph** planning node + state; or CrewAI “planning Agent + dynamic Task”; or AutoGen Manager + Planner agent. |
| 3. Parallel/serial execution, dispatch | **LangGraph** nodes call OpenClaw/acpx, graph expresses parallel/serial; or **CrewAI** Task.agent + Process. |
| 4a. Skill missing → generate and deliver | **LangGraph** generation node (or CrewAI Task): call “generation agent” to write SKILL.md etc., install/copy, mark state ready, then execute. |
| 4b. Tool missing → generate and deliver | Same; generation node produces Tool code/schema, register to OpenClaw/agency-swarm/CrewAI, then assign to agent. |
| 4c. Progress, feedback, cancellation | **LangGraph** state + supervisor/conditional edges + stream/checkpoint; CrewAI callbacks + timeout/guardrails. |
| 5. Summarize and capture lessons | Final node aggregates state, writes report; optional knowledge base, register Skill/Tool to ClawHub or catalog. |

---

## 5. Can OpenClaw alone implement this?

Without **LangGraph/CrewAI** (or other orchestration), can **OpenClaw only** (Gateway + multi-agent + Skill + session tools) implement phases 1–5? Summary below.

### 5.1 Capability checklist

| Phase | OpenClaw-only? | Notes |
|-------|----------------|-------|
| **1. Publish task** | ✅ **Yes** | `openclaw agent --agent <id> --message "..."` or inbound message (Slack/WebChat, etc.); bindings choose agent. |
| **2. Planning & exploration** | ⚠️ **Via a “planning Agent”** | No built-in DAG/planner. Use a **planning agent** (e.g. `planner`) and AGENTS.md to “decompose goal, produce subtasks and suggested agents”; that agent uses **sessions_send** or **sessions_spawn** to assign work. Quality depends on prompt and model, not framework. |
| **3. Parallel/serial execution, dispatch** | ✅ **Yes** | **sessions_send**: send to a session (another agent’s main or named session), wait or fire-and-forget. **sessions_spawn**: dispatch to **child agent** (`agentId`), async, `runTimeoutSeconds`; results via announce; multiple spawns = parallel. Serial = sequential send/spawn and wait. Community Skills **dispatch-multiple-agents**, **Clawflow** (DAG). |
| **4a. Skill missing → generate and deliver** | ⚠️ **Custom flow and tools** | No “one-click Skill generation”. Planning or a “generation agent” uses **write file + shell** (or existing Skill) to produce SKILL.md, then `npx clawhub install <path>` or copy to agent workspace/skills; AGENTS.md must define when/how to generate, install, and notify. |
| **4b. Tool missing → generate and deliver** | ⚠️ **Custom flow and tools** | Same: an agent generates Tool code/schema, writes to workspace or registers via Skill; OpenClaw tools come from Skill/config; **dynamic registration** may need custom Skill or plugin, or write new tool into agent’s skills and reload. |
| **4c. Progress, feedback, cancellation** | ⚠️ **Partial** | **Progress**: **sessions_list** / **sessions_history** for sessions/messages; no built-in “task-level progress table”; planning agent or each agent maintains (e.g. MEMORY.md or external API). **Feedback**: streaming and deliver to channel; for “current step/progress%”, agents must output explicitly. **Cancel**: **sessions_spawn** `runTimeoutSeconds` can timeout; **no** official “user/supervisor cancel run” session API (check Gateway). Child agents are single-level (no nested spawn); complex DAG needs multiple send/spawn rounds. |
| **5. Summarize and capture lessons** | ✅ **Yes** | Planning agent (or last agent) does final summary, writes MEMORY.md or summary; optionally register new Skill to ClawHub or team catalog. |

### 5.2 Conclusion and recommendation

- **Directly possible with OpenClaw**: 1 publish, 3 dispatch and parallel/serial (sessions_send / sessions_spawn + multiple agents), 5 summarize.
- **Requires “build inside OpenClaw”**: 2 planning (one planning agent + sessions_send/spawn), 4a/4b generation (agent with write/execute + agreed flow), 4c progress and cancel (self-maintained state + runTimeoutSeconds; active cancel depends on Gateway).
- **If you want “task DAG, progress table, unified cancel, generation nodes” as first-class abstractions**: use **LangGraph (or CrewAI) for orchestration** and OpenClaw for execution (nodes call openclaw:<agent-id> or sessions_send/spawn); then 2/4a/4b/4c are handled by the orchestration layer, OpenClaw focuses on “multi-agent execution and Skill/Tool”.
- **If you stay OpenClaw-only**: use **one planning agent + sessions_send/sessions_spawn + dispatch-multiple-agents/Clawflow-style Skill + agreed “generate Skill/Tool” flow and permissions**; use runTimeoutSeconds and self-maintained state (MEMORY or external store) for progress and cancel, accepting “no native DAG, no native task-level cancel”.

### 5.3 OpenClaw references

- **Dispatch to other agents**: [Session Tools](https://docs.openclaw.ai/concepts/session-tool) (sessions_send, sessions_spawn); spawn supports `agentId`, `runTimeoutSeconds`.
- **Multi-agent parallel**: Community Skills **dispatch-multiple-agents**, **Clawflow** (DAG).
- **Skill/Tool**: workspace `skills/`, ClawHub install, AGENTS.md for behavior.

---

## 6. References

- [OpenClaw docs](https://docs.openclaw.ai/) (Skill, Gateway, OpenAI API)
- [OpenClaw Session Tools](https://docs.openclaw.ai/concepts/session-tool) (sessions_send, sessions_spawn)
