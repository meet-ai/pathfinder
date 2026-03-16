package planning

import "context"

// Planner 规划器端口：根据目标产出计划，由 infra 实现（如 LLM 规划器）；实现可依赖 PlanLearning 做自动学习。
type Planner interface {
	PlanGoal(ctx context.Context, goal GoalDescription) (*Plan, error)
}

// PlanRepository 计划仓储端口：按 PlanId 持久化/加载 Plan，由 infra 实现。
type PlanRepository interface {
	Save(ctx context.Context, p *Plan) error
	Get(ctx context.Context, id PlanId) (*Plan, error)
}

// RunOutcome 单次 Run 的结果摘要，供规划学习使用。
type RunOutcome struct {
	RunId     string
	Completed int
	Total     int
	Success   bool   // 未取消、未超时、无致命错误
	Summary   string
}

// PastRunForPlanning 历史 Run 的规划侧摘要，供 Planner 实现做相似目标参考（如 few-shot）。
type PastRunForPlanning struct {
	RunId     string
	Goal      GoalDescription
	PlanId    PlanId
	SubTasks  []SubTask
	Outcome   RunOutcome
}

// PlanLearning 规划学习端口：供 Planner 实现读取历史、写入结果，实现自动学习。
type PlanLearning interface {
	// SimilarPast 返回与当前目标可参考的历史 Run（如最近 N 条），由实现决定排序与筛选。
	SimilarPast(ctx context.Context, goal GoalDescription, limit int) ([]PastRunForPlanning, error)
	// RecordOutcome 记录本次 Run 的目标、计划与结果，供后续学习。
	RecordOutcome(ctx context.Context, runId string, goal GoalDescription, plan *Plan, outcome *RunOutcome) error
}
