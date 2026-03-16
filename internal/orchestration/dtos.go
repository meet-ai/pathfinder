package orchestration

import "time"

// JobDTO 提交目标返回：JobId、StreamHandle、状态、创建时间。
type JobDTO struct {
	JobId        string
	StreamHandle string
	Status       string
	CreatedAt    time.Time
}

// PlanDTO 计划 DTO（规划产出）。
type PlanDTO struct {
	PlanId          string
	SubTasks        []SubTaskDTO
	Dependencies    []DependencyDTO
	SuggestedAgents map[string]string // TaskId -> AgentId
}

// SubTaskDTO 子任务 DTO。
type SubTaskDTO struct {
	TaskId      string
	Description string
}

// DependencyDTO 依赖 DTO。
type DependencyDTO struct {
	From string
	To   string
}

// SummaryDTO 运行总结 DTO。
type SummaryDTO struct {
	JobId     string
	Status    string
	Summary   string
	Completed int
	Total     int
}

// JobStateDTO 供 TUI 轮询的 Job 状态：阶段、进度、任务列表、执行图、总结与错误。
type JobStateDTO struct {
	Status    string
	Completed int
	Total     int
	Tasks     []TaskProgressDTO
	Plan      *PlanDTO
	Summary   string
	ErrMsg    string
}

// TaskProgressDTO 单任务进度 DTO（含计划中的描述与执行该任务的 Agent）。
type TaskProgressDTO struct {
	TaskId      string
	Description string
	Status      string
	Result      string
	AgentId     string // 执行该任务的 Agent，供 TUI 显示「当前 Agent」
}
