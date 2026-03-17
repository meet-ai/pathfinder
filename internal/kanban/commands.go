package kanban

import "fmt"

// ManageKanbanCardLifecycleCommand 看板卡片生命周期用例输入。
type ManageKanbanCardLifecycleCommand struct {
	BoardId        string
	Title          string
	Description    string
	Creator        string
	TaskType       string
	Reviewer       string
	NeedBlockCycle bool
	BlockReason    string
}

// Validate 校验命令基础字段；业务规则细节留给领域层。
func (c ManageKanbanCardLifecycleCommand) Validate() error {
	if c.BoardId == "" {
		return fmt.Errorf("boardId is required")
	}
	if c.Title == "" {
		return fmt.Errorf("title is required")
	}
	if c.Creator == "" {
		return fmt.Errorf("creator is required")
	}
	if c.TaskType == "" {
		return fmt.Errorf("taskType is required")
	}
	if c.Reviewer == "" {
		return fmt.Errorf("reviewer is required")
	}
	if c.NeedBlockCycle && c.BlockReason == "" {
		return fmt.Errorf("blockReason is required when needBlockCycle is true")
	}
	return nil
}
