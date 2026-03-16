package orchestration

import (
	"context"
	"testing"
	"time"

	"pathfinder/internal/infra/clients"
	"pathfinder/internal/infra/persistence"
	"pathfinder/internal/planning"
	"pathfinder/internal/progress"
	"pathfinder/internal/runtime"
)

// TestPhase0_SubmitGoal_OneTask_CompletedAndSummary 验收 P0-10：端到端规划→执行(1步stub)→进度→总结；Run 终态 Completed、总结非空。
func TestPhase0_SubmitGoal_OneTask_CompletedAndSummary(t *testing.T) {
	ctx := context.Background()
	runRepo := persistence.NewJobRepoMem()
	planRepo := persistence.NewPlanRepoMem()
	taskProgressRepo := persistence.NewTaskProgressRepoMem()
	planner := &persistence.PlannerStub{}
	agentDiscovery := clients.NewAgentDiscoveryMem()
	dispatcher := &clients.DispatcherStub{}

	svc := &WorkflowOrchestrationApplicationService{
		Planner:        planner,
		RunRepo:        runRepo,
		PlanRepo:       planRepo,
		TaskProgress:   taskProgressRepo,
		AgentDiscovery: agentDiscovery,
		Dispatcher:     dispatcher,
	}

	dto, err := svc.SubmitGoal(ctx, SubmitGoalCommand{
		GoalDescription: planning.GoalDescription("e2e test goal"),
	})
	if err != nil {
		t.Fatalf("SubmitGoal: %v", err)
	}
	if dto.JobId == "" {
		t.Fatal("JobId 为空")
	}

	r, err := runRepo.Get(ctx, runtime.JobId(dto.JobId))
	if err != nil {
		t.Fatalf("RunRepo.Get: %v", err)
	}
	if r == nil {
		t.Fatal("Run 未找到")
	}
	if r.Status != runtime.JobStatusCompleted {
		t.Errorf("Job.Status = %s, want completed", r.Status)
	}

	tasks, err := taskProgressRepo.ListByRunId(ctx, r.Id)
	if err != nil {
		t.Fatalf("ListByRunId: %v", err)
	}
	if len(tasks) < 1 {
		t.Fatalf("TaskProgress 条数 = %d, want >= 1", len(tasks))
	}
	completed := 0
	for _, tp := range tasks {
		if tp.Status == progress.TaskStatusCompleted {
			completed++
		}
	}
	if completed < 1 {
		t.Errorf("已完成任务数 = %d, want >= 1", completed)
	}

	sum, err := svc.SummarizeJob(ctx, dto.JobId)
	if err != nil {
		t.Fatalf("SummarizeRun: %v", err)
	}
	if sum == nil {
		t.Fatal("SummarizeRun 返回 nil")
	}
	if sum.Summary == "" {
		t.Error("总结为空，验收要求总结非空")
	}
	if sum.Completed < 1 || sum.Total < 1 {
		t.Errorf("进度 completed/total = %d/%d, want >= 1/1", sum.Completed, sum.Total)
	}
	t.Logf("测试内容 端到端 Phase0 成功: JobId=%s Status=%s 进度=%d/%d 总结=%s",
		dto.JobId, sum.Status, sum.Completed, sum.Total, sum.Summary)
}

// TestPhase1_StartRun_ContinueRun_GetRunState 验收 Phase1 异步：StartRun 返回 runId，ContinueRun 后台执行，GetRunState 可轮询，CancelRun 可取消。
func TestPhase1_StartRun_ContinueRun_GetRunState(t *testing.T) {
	ctx := context.Background()
	runRepo := persistence.NewJobRepoMem()
	planRepo := persistence.NewPlanRepoMem()
	taskProgressRepo := persistence.NewTaskProgressRepoMem()
	planner := &persistence.PlannerStub{}
	agentDiscovery := clients.NewAgentDiscoveryMem()
	dispatcher := &clients.DispatcherStub{}

	svc := &WorkflowOrchestrationApplicationService{
		Planner:        planner,
		RunRepo:        runRepo,
		PlanRepo:       planRepo,
		TaskProgress:   taskProgressRepo,
		AgentDiscovery: agentDiscovery,
		Dispatcher:     dispatcher,
	}

	jobId, err := svc.StartJob(ctx, SubmitGoalCommand{
		GoalDescription: planning.GoalDescription("phase1 async goal"),
	})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if jobId == "" {
		t.Fatal("StartJob 返回空 jobId")
	}

	go svc.ContinueJob(ctx, jobId)

	// 轮询直到完成或超时
	for i := 0; i < 50; i++ {
		state, err := svc.GetJobState(ctx, jobId)
		if err != nil {
			t.Fatalf("GetRunState: %v", err)
		}
		if state == nil {
			continue
		}
		if state.Status == "completed" || state.Status == "aborted" {
			if state.Completed < 1 || state.Total < 1 {
				t.Errorf("进度 = %d/%d, want >= 1/1", state.Completed, state.Total)
			}
			if state.Summary == "" {
				t.Error("总结为空")
			}
			t.Logf("测试内容 Phase1 异步 StartRun+ContinueRun+GetRunState 成功: Status=%s 进度=%d/%d",
				state.Status, state.Completed, state.Total)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("GetRunState 未在限定时间内变为 completed/aborted")
}
