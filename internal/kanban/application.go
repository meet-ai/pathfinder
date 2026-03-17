package kanban

import (
	"context"
	"fmt"
	"time"
)

// ApplicationService 看板应用服务：仅做流程编排，不承载业务规则细节。
type ApplicationService struct {
	BoardRepo        BoardRepository
	CardRepo         CardRepository
	AssigneeResolver AssigneeResolver
	EventPublisher   EventPublisher
}

// CreateCard 创建卡片并进入 todo。
func (s *ApplicationService) CreateCard(
	ctx context.Context,
	boardId string,
	title string,
	description string,
	creator string,
	reviewer string,
) (*CardDTO, error) {
	board, err := s.BoardRepo.Get(ctx, boardId)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, fmt.Errorf("board not found: %s", boardId)
	}
	card := NewCard(newCardId(), boardId, title, description, creator, reviewer, time.Now().UTC())
	if err := s.CardRepo.Create(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardCreated", card, creator, nil); err != nil {
		return nil, err
	}
	return toCardDTO(card), nil
}

// AssignCard 指派责任人。
func (s *ApplicationService) AssignCard(ctx context.Context, cardId string, taskType string, operator string) (*CardDTO, error) {
	card, err := s.CardRepo.Get(ctx, cardId)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, fmt.Errorf("card not found: %s", cardId)
	}
	assignee, err := s.AssigneeResolver.Resolve(ctx, card.BoardId, taskType)
	if err != nil {
		return nil, err
	}
	if err := card.Assign(assignee, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardAssigned", card, operator, nil); err != nil {
		return nil, err
	}
	return toCardDTO(card), nil
}

// MoveCard 状态迁移。
func (s *ApplicationService) MoveCard(ctx context.Context, cardId string, target CardStatus, operator string) (*CardDTO, error) {
	card, board, err := s.loadCardAndBoard(ctx, cardId)
	if err != nil {
		return nil, err
	}
	if target == CardStatusInProgress || target == CardStatusInReview {
		count, err := s.CardRepo.CountByBoardAndStatus(ctx, board.Id, target)
		if err != nil {
			return nil, err
		}
		if err := board.ValidateWip(target, count); err != nil {
			return nil, err
		}
	}
	if err := card.MoveTo(target, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardMoved", card, operator, nil); err != nil {
		return nil, err
	}
	return toCardDTO(card), nil
}

// BlockCard 标记阻塞。
func (s *ApplicationService) BlockCard(ctx context.Context, cardId string, reason string, operator string) (*CardDTO, error) {
	card, _, err := s.loadCardAndBoard(ctx, cardId)
	if err != nil {
		return nil, err
	}
	if err := card.Block(reason, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardBlocked", card, operator, nil); err != nil {
		return nil, err
	}
	return toCardDTO(card), nil
}

// UnblockCard 解除阻塞。
func (s *ApplicationService) UnblockCard(ctx context.Context, cardId string, operator string) (*CardDTO, error) {
	card, _, err := s.loadCardAndBoard(ctx, cardId)
	if err != nil {
		return nil, err
	}
	if err := card.Unblock(time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardUnblocked", card, operator, nil); err != nil {
		return nil, err
	}
	return toCardDTO(card), nil
}

// ReviewPassCard 审核通过后进入 done。
func (s *ApplicationService) ReviewPassCard(ctx context.Context, cardId string, reviewer string) (*CardDTO, error) {
	card, _, err := s.loadCardAndBoard(ctx, cardId)
	if err != nil {
		return nil, err
	}
	if reviewer != "" {
		card.Reviewer = reviewer
	}
	if err := card.MoveTo(CardStatusDone, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardCompleted", card, card.Reviewer, nil); err != nil {
		return nil, err
	}
	return toCardDTO(card), nil
}

// ListCards 列出看板全部卡片。
func (s *ApplicationService) ListCards(ctx context.Context, boardId string) ([]CardDTO, error) {
	cards, err := s.CardRepo.ListByBoard(ctx, boardId)
	if err != nil {
		return nil, err
	}
	out := make([]CardDTO, 0, len(cards))
	for i := range cards {
		out = append(out, *toCardDTO(&cards[i]))
	}
	return out, nil
}

// GetCard 获取卡片详情。
func (s *ApplicationService) GetCard(ctx context.Context, cardId string) (*CardDTO, error) {
	card, err := s.CardRepo.Get(ctx, cardId)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, nil
	}
	return toCardDTO(card), nil
}

// ManageKanbanCardLifecycle 起步用例：打通卡片生命周期 Happy Path。
func (s *ApplicationService) ManageKanbanCardLifecycle(
	ctx context.Context,
	cmd ManageKanbanCardLifecycleCommand,
) (*KanbanCardLifecycleDTO, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	board, err := s.BoardRepo.Get(ctx, cmd.BoardId)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, fmt.Errorf("board not found: %s", cmd.BoardId)
	}

	now := time.Now().UTC()
	card := NewCard(
		newCardId(),
		cmd.BoardId,
		cmd.Title,
		cmd.Description,
		cmd.Creator,
		cmd.Reviewer,
		now,
	)
	if err := s.CardRepo.Create(ctx, card); err != nil {
		return nil, err
	}

	events := make([]LifecycleEventSummary, 0, 8)
	if err := s.publishEvent(ctx, "CardCreated", card, cmd.Creator, &events); err != nil {
		return nil, err
	}

	assignee, err := s.AssigneeResolver.Resolve(ctx, cmd.BoardId, cmd.TaskType)
	if err != nil {
		return nil, err
	}
	if err := card.Assign(assignee, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardAssigned", card, "system", &events); err != nil {
		return nil, err
	}

	inProgressCount, err := s.CardRepo.CountByBoardAndStatus(ctx, board.Id, CardStatusInProgress)
	if err != nil {
		return nil, err
	}
	if err := board.ValidateWip(CardStatusInProgress, inProgressCount); err != nil {
		return nil, err
	}
	if err := card.MoveTo(CardStatusInProgress, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardMovedToInProgress", card, assignee, &events); err != nil {
		return nil, err
	}

	if cmd.NeedBlockCycle {
		if err := card.Block(cmd.BlockReason, time.Now().UTC()); err != nil {
			return nil, err
		}
		if err := s.CardRepo.Save(ctx, card); err != nil {
			return nil, err
		}
		if err := s.publishEvent(ctx, "CardBlocked", card, assignee, &events); err != nil {
			return nil, err
		}

		if err := card.Unblock(time.Now().UTC()); err != nil {
			return nil, err
		}
		if err := s.CardRepo.Save(ctx, card); err != nil {
			return nil, err
		}
		if err := s.publishEvent(ctx, "CardUnblocked", card, assignee, &events); err != nil {
			return nil, err
		}
	}

	inReviewCount, err := s.CardRepo.CountByBoardAndStatus(ctx, board.Id, CardStatusInReview)
	if err != nil {
		return nil, err
	}
	if err := board.ValidateWip(CardStatusInReview, inReviewCount); err != nil {
		return nil, err
	}
	if err := card.MoveTo(CardStatusInReview, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardMovedToInReview", card, assignee, &events); err != nil {
		return nil, err
	}

	if err := card.MoveTo(CardStatusDone, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.CardRepo.Save(ctx, card); err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "CardCompleted", card, cmd.Reviewer, &events); err != nil {
		return nil, err
	}

	return &KanbanCardLifecycleDTO{
		CardId:          card.Id,
		BoardId:         card.BoardId,
		CurrentStatus:   card.Status,
		Assignee:        card.Assignee,
		Reviewer:        card.Reviewer,
		CreatedAt:       card.CreatedAt,
		CompletedAt:     card.CompletedAt,
		LifecycleEvents: events,
	}, nil
}

func (s *ApplicationService) publishEvent(
	ctx context.Context,
	name string,
	card *Card,
	operator string,
	events *[]LifecycleEventSummary,
) error {
	event := DomainEvent{
		Name:       name,
		BoardId:    card.BoardId,
		CardId:     card.Id,
		Operator:   operator,
		OccurredAt: time.Now().UTC(),
	}
	if err := s.EventPublisher.Publish(ctx, event); err != nil {
		return err
	}
	if events == nil {
		return nil
	}
	*events = append(*events, LifecycleEventSummary{
		Name:       event.Name,
		OccurredAt: event.OccurredAt,
		Operator:   event.Operator,
	})
	return nil
}

func newCardId() string {
	return fmt.Sprintf("card-%d", time.Now().UTC().UnixNano())
}

func toCardDTO(card *Card) *CardDTO {
	return &CardDTO{
		ID:             card.Id,
		BoardID:        card.BoardId,
		Title:          card.Title,
		Description:    card.Description,
		Creator:        card.Creator,
		Assignee:       card.Assignee,
		Reviewer:       card.Reviewer,
		Status:         card.Status,
		BlockReason:    card.BlockReason,
		CreatedAt:      card.CreatedAt,
		LastActivityAt: card.LastActivityAt,
		CompletedAt:    card.CompletedAt,
	}
}

func (s *ApplicationService) loadCardAndBoard(ctx context.Context, cardId string) (*Card, *Board, error) {
	card, err := s.CardRepo.Get(ctx, cardId)
	if err != nil {
		return nil, nil, err
	}
	if card == nil {
		return nil, nil, fmt.Errorf("card not found: %s", cardId)
	}
	board, err := s.BoardRepo.Get(ctx, card.BoardId)
	if err != nil {
		return nil, nil, err
	}
	if board == nil {
		return nil, nil, fmt.Errorf("board not found: %s", card.BoardId)
	}
	return card, board, nil
}
