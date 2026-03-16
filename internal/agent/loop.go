package agent

import (
	"context"
	"log/slog"
	"time"

	"pathfinder/internal/planning"
	"pathfinder/internal/progress"
	"pathfinder/internal/runtime"
)

// Loop 执行体循环：按 Plan 与依赖顺序派发子任务、写回进度；参考 zeroclaw run_tool_call_loop 的“循环直至无新工作”结构。
// 依赖 Dispatcher、TaskProgressRepository、AgentDiscovery、可选 AbortCheck；needs 见 agent/needs.go。
type Loop struct {
	Dispatcher   Dispatcher
	ProgressRepo TaskProgressRepository
	Discovery    AgentDiscovery
	AbortCheck   AbortCheck
}

// Run 执行整轮循环：恢复进度 → 找出可执行任务 → 派发 → 写回进度，直到无新可执行任务或 AbortCheck 为真。
func (l *Loop) Run(ctx context.Context, runId runtime.JobId, plan *planning.Plan) error {
	if plan == nil || len(plan.SubTasks) == 0 {
		return nil
	}
	for {
		if l.AbortCheck != nil && l.AbortCheck(ctx, runId) {
			return nil
		}
		tasks, err := l.ProgressRepo.ListByRunId(ctx, runId)
		if err != nil {
			return err
		}
		done := make(map[planning.TaskId]bool)
		for _, t := range tasks {
			if t.Status == progress.TaskStatusCompleted || t.Status == progress.TaskStatusSkipped {
				done[t.TaskId] = true
			}
		}
		advanced := false
		for _, st := range plan.SubTasks {
			if done[st.TaskId] {
				continue
			}
			ready := true
			for _, d := range plan.Dependencies {
				if d.To != st.TaskId {
					continue
				}
				if !done[d.From] {
					ready = false
					break
				}
			}
			if !ready {
				continue
			}
			agentId := plan.SuggestedAgentFor(st.TaskId)
			if agentId == "" {
				agents, _ := l.Discovery.ListAgents(ctx, AgentPoolFilter{})
				if len(agents) > 0 {
					agentId = string(agents[0].Id)
				}
			}
			if agentId == "" {
				slog.Debug("agent loop: no agent for task", "taskId", st.TaskId)
				continue
			}
			advanced = true
			tp := progress.TaskProgress{
				RunId:     runId,
				TaskId:    st.TaskId,
				UpdatedAt: time.Now().UTC(),
			}
			tp.Start(agentId)
			if err := l.ProgressRepo.Save(ctx, runId, &tp); err != nil {
				return err
			}
			result, err := l.Dispatcher.Dispatch(ctx, runId, st, AgentId(agentId))
			if err != nil {
				tp.Fail(err.Error())
			} else {
				tp.Complete(result)
			}
			if err := l.ProgressRepo.Save(ctx, runId, &tp); err != nil {
				return err
			}
			done[st.TaskId] = true
		}
		if !advanced {
			break
		}
	}
	return nil
}
