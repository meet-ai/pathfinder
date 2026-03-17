package kanban

import "time"

// LifecycleEventSummary 生命周期事件摘要，供看板读模型展示。
type LifecycleEventSummary struct {
	Name       string
	OccurredAt time.Time
	Operator   string
}

// KanbanCardLifecycleDTO 生命周期用例输出。
type KanbanCardLifecycleDTO struct {
	CardId          string
	BoardId         string
	CurrentStatus   CardStatus
	Assignee        string
	Reviewer        string
	CreatedAt       time.Time
	CompletedAt     *time.Time
	LifecycleEvents []LifecycleEventSummary
}

// CardDTO 看板卡片详情/列表展示输出。
type CardDTO struct {
	ID             string     `json:"id"`
	BoardID        string     `json:"boardId"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Creator        string     `json:"creator"`
	Assignee       string     `json:"assignee"`
	Reviewer       string     `json:"reviewer"`
	Status         CardStatus `json:"status"`
	BlockReason    string     `json:"blockReason"`
	CreatedAt      time.Time  `json:"createdAt"`
	LastActivityAt time.Time  `json:"lastActivityAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
}
