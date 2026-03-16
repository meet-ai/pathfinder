package orchestration

import "pathfinder/internal/planning"

// SubmitGoalCommand 提交目标命令。
type SubmitGoalCommand struct {
	GoalDescription planning.GoalDescription
	TimeoutSecs     uint64
	Priority        int
	AgentPoolId     string
}

// SummarizeJobCommand 总结 job 命令（内部用 JobId 即可，此处预留）。
type SummarizeJobCommand struct {
	JobId string
}
