package app

import (
	"context"
	"fmt"
	"log/slog"

	"pathfinder/internal/agent"
	"pathfinder/internal/config"
	"pathfinder/internal/infra/clients"
	"pathfinder/internal/infra/persistence"
	"pathfinder/internal/orchestration"
	"pathfinder/internal/planning"
	"pathfinder/internal/provider"
)

// Run 接收任务描述并执行：加载 config、编排 SubmitGoal，返回 JobId 或错误。
func Run(message string) (jobId string, err error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	slog.Debug("config", "provider", cfg.DefaultProvider, "model", cfg.DefaultModel, "workspace", cfg.WorkspaceDir)

	runRepo := persistence.NewJobRepoMem()
	planRepo := persistence.NewPlanRepoMem()
	taskProgressRepo := persistence.NewTaskProgressRepoMem()
	planner := &persistence.PlannerStub{}
	agentDiscovery := clients.NewAgentDiscoveryMem()
	var dispatcher agent.Dispatcher
	if p, err := provider.CreateFromConfig(context.Background(), cfg); err == nil {
		dispatcher = &clients.DispatcherLLM{
			Provider:    p,
			Model:       cfg.DefaultModel,
			Temperature: cfg.DefaultTemperature,
		}
	} else {
		dispatcher = &clients.DispatcherStub{}
	}

	svc := &orchestration.WorkflowOrchestrationApplicationService{
		Planner:        planner,
		RunRepo:        runRepo,
		PlanRepo:       planRepo,
		TaskProgress:   taskProgressRepo,
		AgentDiscovery: agentDiscovery,
		Dispatcher:     dispatcher,
	}

	ctx := context.Background()
	dto, err := svc.SubmitGoal(ctx, orchestration.SubmitGoalCommand{
		GoalDescription: planning.GoalDescription(message),
	})
	if err != nil {
		return "", err
	}
	sum, _ := svc.SummarizeJob(ctx, dto.JobId)
	if sum != nil {
		fmt.Println("阶段:", sum.Status, "进度:", sum.Completed, "/", sum.Total)
		if sum.Summary != "" {
			fmt.Println("总结:", sum.Summary)
		}
	} else {
		fmt.Println("JobId:", dto.JobId, "Status:", dto.Status)
	}
	return dto.JobId, nil
}

// RunAsync 异步执行：StartJob 后立即返回 jobId，后台执行 ContinueJob；getState/cancelJob 供 TUI 轮询与取消。
func RunAsync(message string) (jobId string, getState func() *orchestration.JobStateDTO, cancelJob func() error, err error) {
	cfg, err := config.Load()
	if err != nil {
		return "", nil, nil, err
	}
	slog.Debug("config", "provider", cfg.DefaultProvider, "model", cfg.DefaultModel, "workspace", cfg.WorkspaceDir)

	runRepo := persistence.NewJobRepoMem()
	planRepo := persistence.NewPlanRepoMem()
	taskProgressRepo := persistence.NewTaskProgressRepoMem()
	planner := &persistence.PlannerStub{}
	agentDiscovery := clients.NewAgentDiscoveryMem()
	var dispatcher agent.Dispatcher
	if p, err := provider.CreateFromConfig(context.Background(), cfg); err == nil {
		dispatcher = &clients.DispatcherLLM{Provider: p, Model: cfg.DefaultModel, Temperature: cfg.DefaultTemperature}
	} else {
		dispatcher = &clients.DispatcherStub{}
	}

	svc := &orchestration.WorkflowOrchestrationApplicationService{
		Planner:        planner,
		RunRepo:        runRepo,
		PlanRepo:       planRepo,
		TaskProgress:   taskProgressRepo,
		AgentDiscovery: agentDiscovery,
		Dispatcher:     dispatcher,
	}

	ctx := context.Background()
	jobId, err = svc.StartJob(ctx, orchestration.SubmitGoalCommand{
		GoalDescription: planning.GoalDescription(message),
	})
	if err != nil {
		return "", nil, nil, err
	}
	go svc.ContinueJob(ctx, jobId)
	getState = func() *orchestration.JobStateDTO {
		st, _ := svc.GetJobState(ctx, jobId)
		return st
	}
	cancelJob = func() error {
		return svc.CancelJob(ctx, jobId)
	}
	return jobId, getState, cancelJob, nil
}
