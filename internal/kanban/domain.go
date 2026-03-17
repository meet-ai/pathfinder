package kanban

import (
	"fmt"
	"time"
)

// NewCard 创建卡片并初始化到 todo。
func NewCard(
	id string,
	boardId string,
	title string,
	description string,
	creator string,
	reviewer string,
	now time.Time,
) *Card {
	return &Card{
		Id:             id,
		BoardId:        boardId,
		Title:          title,
		Description:    description,
		Creator:        creator,
		Reviewer:       reviewer,
		Status:         CardStatusTodo,
		CreatedAt:      now,
		LastActivityAt: now,
	}
}

// Assign 指派责任人。
func (c *Card) Assign(assignee string, now time.Time) error {
	if assignee == "" {
		return fmt.Errorf("assignee is required")
	}
	c.Assignee = assignee
	c.LastActivityAt = now
	return nil
}

// MoveTo 按状态机迁移状态。
func (c *Card) MoveTo(target CardStatus, now time.Time) error {
	if !canMove(c.Status, target) {
		return fmt.Errorf("invalid status transition: %s -> %s", c.Status, target)
	}
	if target == CardStatusInReview && c.Assignee == "" {
		return fmt.Errorf("assignee is required before move to in_review")
	}
	if target == CardStatusDone {
		completedAt := now
		c.CompletedAt = &completedAt
	}
	c.Status = target
	c.LastActivityAt = now
	return nil
}

// Block 标记阻塞。
func (c *Card) Block(reason string, now time.Time) error {
	if reason == "" {
		return fmt.Errorf("block reason is required")
	}
	if !canMove(c.Status, CardStatusBlocked) {
		return fmt.Errorf("invalid status transition: %s -> %s", c.Status, CardStatusBlocked)
	}
	c.Status = CardStatusBlocked
	c.BlockReason = reason
	c.LastActivityAt = now
	return nil
}

// Unblock 解除阻塞并回到执行态。
func (c *Card) Unblock(now time.Time) error {
	if c.Status != CardStatusBlocked {
		return fmt.Errorf("card is not blocked")
	}
	c.Status = CardStatusInProgress
	c.BlockReason = ""
	c.LastActivityAt = now
	return nil
}

// ValidateWip 检查目标列 WIP 限制。
func (b *Board) ValidateWip(target CardStatus, currentCount int) error {
	limit := 0
	switch target {
	case CardStatusInProgress:
		limit = b.InProgressLimit
	case CardStatusInReview:
		limit = b.InReviewLimit
	default:
		return nil
	}
	if limit > 0 && currentCount >= limit {
		return fmt.Errorf("wip limit exceeded for %s", target)
	}
	return nil
}

func canMove(from CardStatus, to CardStatus) bool {
	switch from {
	case CardStatusTodo:
		return to == CardStatusInProgress || to == CardStatusBlocked
	case CardStatusInProgress:
		return to == CardStatusInReview || to == CardStatusBlocked
	case CardStatusInReview:
		return to == CardStatusDone || to == CardStatusBlocked
	case CardStatusBlocked:
		return to == CardStatusInProgress
	case CardStatusDone:
		return false
	default:
		return false
	}
}
