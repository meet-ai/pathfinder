package orchestration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"pathfinder/internal/agent"
	"pathfinder/internal/planning"
	"pathfinder/internal/progress"
	"pathfinder/internal/runtime"
)

// WorkflowOrchestrationApplicationService 工作流编排应用服务。
type WorkflowOrchestrationApplicationService struct {
	Planner       planning.Planner
	RunRepo       runtime.JobRepository
	PlanRepo      planning.PlanRepository
	TaskProgress  progress.TaskProgressRepository
	AgentDiscovery agent.AgentDiscovery
	Dispatcher    agent.Dispatcher
}

// SubmitGoal 提交目标：创建 Job、规划、保存、执行计划、总结。
func (s *WorkflowOrchestrationApplicationService) SubmitGoal(ctx context.Context, cmd SubmitGoalCommand) (*JobDTO, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	runId := runtime.JobId(hex.EncodeToString(b))
	streamHandle := runtime.StreamHandle("stream-" + string(runId))

	plan, err := s.Planner.PlanGoal(ctx, cmd.GoalDescription)
	if err != nil {
		return nil, err
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if err := s.PlanRepo.Save(ctx, plan); err != nil {
		return nil, err
	}

	var deadline *time.Time
	if cmd.TimeoutSecs > 0 {
		t := time.Now().UTC().Add(time.Duration(cmd.TimeoutSecs) * time.Second)
		deadline = &t
	}
	r := runtime.Create(runId, streamHandle, plan.Id, deadline)
	r.Status = runtime.JobStatusPlanning
	if err := s.RunRepo.Save(ctx, r); err != nil {
		return nil, err
	}

	for _, st := range plan.SubTasks {
		tp := progress.TaskProgress{
			RunId:     runId,
			TaskId:    st.TaskId,
			Status:    progress.TaskStatusPending,
			UpdatedAt: time.Now().UTC(),
		}
		if err := s.TaskProgress.Save(ctx, runId, &tp); err != nil {
			return nil, err
		}
	}

	r.Status = runtime.JobStatusRunning
	if err := s.RunRepo.Save(ctx, r); err != nil {
		return nil, err
	}

	if err := s.ExecutePlan(ctx, runId); err != nil {
		slog.Debug("ExecutePlan", "runId", runId, "err", err)
	}

	sum, _ := s.SummarizeJob(ctx, string(runId))
	slog.Debug("SubmitGoal done", "runId", runId, "summary", sum)

	return &JobDTO{
		JobId:        string(runId),
		StreamHandle: string(streamHandle),
		Status:       string(r.Status),
		CreatedAt:    r.CreatedAt,
	}, nil
}

// ExecutePlan 按计划执行：委托 agent.Loop 跑完“恢复→派发→写回”循环，遇取消/超时由 AbortCheck 中止；最后更新 Job 状态并持久化。
func (s *WorkflowOrchestrationApplicationService) ExecutePlan(ctx context.Context, runId runtime.JobId) error {
	r, err := s.RunRepo.Get(ctx, runId)
	if err != nil {
		return err
	}
	if r == nil || r.PlanId == "" {
		return nil
	}
	plan, err := s.PlanRepo.Get(ctx, r.PlanId)
	if err != nil || plan == nil {
		return err
	}

	abortCheck := func(ctx context.Context, id runtime.JobId) bool {
		run, err := s.RunRepo.Get(ctx, id)
		return err == nil && run != nil && (run.IsCancelRequested() || run.IsOverDeadline(time.Now().UTC()))
	}
	loop := &agent.Loop{
		Dispatcher:   s.Dispatcher,
		ProgressRepo: s.TaskProgress,
		Discovery:    s.AgentDiscovery,
		AbortCheck:   abortCheck,
	}
	if err := loop.Run(ctx, runId, plan); err != nil {
		return err
	}

	r, _ = s.RunRepo.Get(ctx, runId)
	if r == nil {
		return nil
	}
	if r.IsCancelRequested() || r.IsOverDeadline(time.Now().UTC()) {
		r.MarkAborted()
	} else {
		r.MarkCompleted()
	}
	return s.RunRepo.Save(ctx, r)
}

// SummarizeJob 汇总 job 结果，返回 SummaryDTO；Summary 为已完成步骤与结果摘要。
func (s *WorkflowOrchestrationApplicationService) SummarizeJob(ctx context.Context, runId string) (*SummaryDTO, error) {
	r, err := s.RunRepo.Get(ctx, runtime.JobId(runId))
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	pm := &progress.ProgressMaintainer{Repo: s.TaskProgress}
	tasks, err := pm.Restore(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	completed := 0
	var parts []string
	for _, t := range tasks {
		if t.Status == progress.TaskStatusCompleted {
			completed++
			parts = append(parts, "步骤 "+string(t.TaskId)+": "+t.Result)
		} else if t.Status == progress.TaskStatusFailed {
			parts = append(parts, "步骤 "+string(t.TaskId)+": 失败 - "+t.Result)
		}
	}
	var summary string
	if len(parts) > 0 {
		summary = strings.Join(parts, "\n")
	}
	return &SummaryDTO{
		JobId:     runId,
		Status:    string(r.Status),
		Summary:   summary,
		Completed: completed,
		Total:     len(tasks),
	}, nil
}

// CancelJob 标记 job 为取消请求。
func (s *WorkflowOrchestrationApplicationService) CancelJob(ctx context.Context, runId string) error {
	r, err := s.RunRepo.Get(ctx, runtime.JobId(runId))
	if err != nil {
		return err
	}
	if r == nil {
		return nil
	}
	r.Cancel()
	return s.RunRepo.Save(ctx, r)
}

// StartJob 创建 Job 并完成规划，返回 jobId；执行与总结由 ContinueJob 在后台完成。
func (s *WorkflowOrchestrationApplicationService) StartJob(ctx context.Context, cmd SubmitGoalCommand) (jobId string, err error) {
	plan, err := s.Planner.PlanGoal(ctx, cmd.GoalDescription)
	if err != nil {
		return "", err
	}
	if err := plan.Validate(); err != nil {
		return "", err
	}
	if err := s.PlanRepo.Save(ctx, plan); err != nil {
		return "", err
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	rid := runtime.JobId(hex.EncodeToString(b))
	streamHandle := runtime.StreamHandle("stream-" + string(rid))
	var deadline *time.Time
	if cmd.TimeoutSecs > 0 {
		t := time.Now().UTC().Add(time.Duration(cmd.TimeoutSecs) * time.Second)
		deadline = &t
	}
	r := runtime.Create(rid, streamHandle, plan.Id, deadline)
	r.Status = runtime.JobStatusPlanning
	if err := s.RunRepo.Save(ctx, r); err != nil {
		return "", err
	}
	return string(rid), nil
}

// ContinueJob 在后台执行计划并收尾：初始化进度、执行、更新 Job 终态。
func (s *WorkflowOrchestrationApplicationService) ContinueJob(ctx context.Context, runId string) {
	rid := runtime.JobId(runId)
	r, err := s.RunRepo.Get(ctx, rid)
	if err != nil || r == nil || r.PlanId == "" {
		return
	}
	plan, err := s.PlanRepo.Get(ctx, r.PlanId)
	if err != nil || plan == nil {
		return
	}
	for _, st := range plan.SubTasks {
		tp := progress.TaskProgress{
			RunId:     rid,
			TaskId:    st.TaskId,
			Status:    progress.TaskStatusPending,
			UpdatedAt: time.Now().UTC(),
		}
		_ = s.TaskProgress.Save(ctx, rid, &tp)
	}
	r.Status = runtime.JobStatusRunning
	_ = s.RunRepo.Save(ctx, r)
	_ = s.ExecutePlan(ctx, rid)
	r, _ = s.RunRepo.Get(ctx, rid)
	if r != nil {
		if r.IsCancelRequested() || r.IsOverDeadline(time.Now().UTC()) {
			r.MarkAborted()
		} else {
			r.MarkCompleted()
		}
		_ = s.RunRepo.Save(ctx, r)
	}
}

// GetJobState 返回当前 Job 状态，供 TUI 轮询。
func (s *WorkflowOrchestrationApplicationService) GetJobState(ctx context.Context, runId string) (*JobStateDTO, error) {
	r, err := s.RunRepo.Get(ctx, runtime.JobId(runId))
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	pm := &progress.ProgressMaintainer{Repo: s.TaskProgress}
	tasks, err := pm.Restore(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	completed := 0
	var taskDTOs []TaskProgressDTO
	var planDTO *PlanDTO
	if r.PlanId != "" {
		plan, _ := s.PlanRepo.Get(ctx, r.PlanId)
		if plan != nil {
			byId := make(map[planning.TaskId]string)
			for _, st := range plan.SubTasks {
				byId[st.TaskId] = st.Description
			}
			planDTO = &PlanDTO{
				PlanId:       string(plan.Id),
				SubTasks:     []SubTaskDTO{},
				Dependencies: []DependencyDTO{},
			}
			for _, st := range plan.SubTasks {
				planDTO.SubTasks = append(planDTO.SubTasks, SubTaskDTO{TaskId: string(st.TaskId), Description: st.Description})
			}
			for _, d := range plan.Dependencies {
				planDTO.Dependencies = append(planDTO.Dependencies, DependencyDTO{From: string(d.From), To: string(d.To)})
			}
			for _, t := range tasks {
				desc := byId[t.TaskId]
				if t.Status == progress.TaskStatusCompleted {
					completed++
				}
				taskDTOs = append(taskDTOs, TaskProgressDTO{
					TaskId:      string(t.TaskId),
					Description: desc,
					Status:      string(t.Status),
					Result:      t.Result,
					AgentId:     t.AgentId,
				})
			}
		}
	}
	if planDTO == nil {
		for _, t := range tasks {
			if t.Status == progress.TaskStatusCompleted {
				completed++
			}
			taskDTOs = append(taskDTOs, TaskProgressDTO{TaskId: string(t.TaskId), Status: string(t.Status), Result: t.Result, AgentId: t.AgentId})
		}
	}
	sum, _ := s.SummarizeJob(ctx, runId)
	out := &JobStateDTO{
		Status:    string(r.Status),
		Completed: completed,
		Total:     len(tasks),
		Tasks:     taskDTOs,
		Plan:      planDTO,
	}
	if sum != nil {
		out.Summary = sum.Summary
	}
	return out, nil
}
