//go:build integration

package clients

import (
	"context"
	"testing"
	"time"

	"pathfinder/internal/agent"
	"pathfinder/internal/infra/persistence"
	"pathfinder/internal/planning"
	"pathfinder/internal/progress"
	"pathfinder/internal/runtime"
)

// 集成测试：agent loop 能正常执行 toolcall（通过 DispatcherStub 模拟）并写回结果到进度仓储。
func TestAgentLoop_Run_DispatchesTasksAndWritesResults(t *testing.T) {
	ctx := context.Background()
	runId := runtime.JobId("job-integ-1")

	plan := &planning.Plan{
		Id:   planning.PlanId("plan-1"),
		SubTasks: []planning.SubTask{
			{TaskId: "t1", Description: "task one"},
			{TaskId: "t2", Description: "task two"},
		},
		Dependencies: []planning.Dependency{
			{From: "t1", To: "t2"},
		},
		SuggestedAgents: []planning.SuggestedAgent{
			{TaskId: "t1", AgentId: "agent-1"},
			{TaskId: "t2", AgentId: "agent-1"},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("plan validate: %v", err)
	}

	progressRepo := persistence.NewTaskProgressRepoMem()
	for _, st := range plan.SubTasks {
		tp := progress.TaskProgress{
			RunId:     runId,
			TaskId:    st.TaskId,
			Status:    progress.TaskStatusPending,
			UpdatedAt: time.Now().UTC(),
		}
		if err := progressRepo.Save(ctx, runId, &tp); err != nil {
			t.Fatalf("seed progress: %v", err)
		}
	}

	dispatcher := &DispatcherStub{}
	discovery := NewAgentDiscoveryMem()

	loop := &agent.Loop{
		Dispatcher:   dispatcher,
		ProgressRepo: progressRepo,
		Discovery:    discovery,
		AbortCheck:   nil,
	}
	if err := loop.Run(ctx, runId, plan); err != nil {
		t.Fatalf("loop run: %v", err)
	}

	tasks, err := progressRepo.ListByRunId(ctx, runId)
	if err != nil {
		t.Fatalf("list progress: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}

	byId := make(map[planning.TaskId]progress.TaskProgress)
	for _, tp := range tasks {
		byId[tp.TaskId] = tp
	}
	for _, st := range plan.SubTasks {
		tp, ok := byId[st.TaskId]
		if !ok {
			t.Errorf("missing progress for task %s", st.TaskId)
			continue
		}
		if tp.Status != progress.TaskStatusCompleted {
			t.Errorf("task %s status = %s, want completed", st.TaskId, tp.Status)
		}
		wantResult := "done: " + st.Description
		if tp.Result != wantResult {
			t.Errorf("task %s result = %q, want %q", st.TaskId, tp.Result, wantResult)
		}
	}
	t.Logf("测试内容 toolcall 执行和返回结果 成功: 2 个任务均 completed，结果符合 DispatcherStub 输出")
}
