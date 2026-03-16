package persistence

import (
	"context"

	"pathfinder/internal/planning"
)

// PlannerStub 占位规划器：根据目标返回单条子任务的计划，供联调与测试。
type PlannerStub struct{}

// PlanGoal 返回仅含一条子任务的 Plan。
func (p *PlannerStub) PlanGoal(ctx context.Context, goal planning.GoalDescription) (*planning.Plan, error) {
	planId := planning.PlanId("plan-" + string(goal))
	if len(planId) > 64 {
		planId = planId[:64]
	}
	return &planning.Plan{
		Id:             planId,
		SubTasks:       []planning.SubTask{{TaskId: "task-1", Description: string(goal)}},
		Dependencies:   []planning.Dependency{},
		SuggestedAgents: []planning.SuggestedAgent{},
	}, nil
}
