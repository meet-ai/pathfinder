package planning

import "log/slog"

// PlanId 计划唯一标识。
type PlanId string

// TaskId 子任务唯一标识（同一 Plan 内唯一）。
type TaskId string

// GoalDescription 高层目标描述。
type GoalDescription string

// Dependency 子任务依赖：From 完成后才可执行 To。
type Dependency struct {
	From TaskId
	To   TaskId
}

// SuggestedAgent 规划阶段建议执行该任务的 Agent。
type SuggestedAgent struct {
	TaskId  TaskId
	AgentId string
}

// SubTask 计划内单条可执行子任务。
type SubTask struct {
	TaskId      TaskId
	Description string
}

// Plan 规划聚合根：计划结构、子任务列表与依赖。
type Plan struct {
	Id            PlanId
	SubTasks      []SubTask
	Dependencies  []Dependency
	SuggestedAgents []SuggestedAgent // TaskId -> AgentId 建议
}

// SuggestedAgentFor 返回某 TaskId 的建议 AgentId，无则空串。
func (p *Plan) SuggestedAgentFor(taskId TaskId) string {
	for _, s := range p.SuggestedAgents {
		if s.TaskId == taskId {
			return s.AgentId
		}
	}
	return ""
}

// Validate 校验计划：至少一个子任务；依赖引用同一 Plan 内 TaskId。
func (p *Plan) Validate() error {
	if len(p.SubTasks) == 0 {
		slog.Debug("plan validate", "reason", "no subtasks")
		return ErrPlanNoSubTasks
	}
	ids := make(map[TaskId]bool)
	for _, t := range p.SubTasks {
		ids[t.TaskId] = true
	}
	for _, d := range p.Dependencies {
		if !ids[d.From] || !ids[d.To] {
			return ErrPlanInvalidDependency
		}
	}
	return nil
}
