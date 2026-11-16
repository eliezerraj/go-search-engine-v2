package usecase

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/go-semantic-engine-v2/application/domain/entity"

	"github.com/go-semantic-engine-v2/application/infrastructure/repository"
	"github.com/eliezerraj/go-core/v3/logger"
	tracing "github.com/go-semantic-engine-v2/application/infrastructure/observability"

	"go.opentelemetry.io/otel/trace"
)

type OutboxUseCase struct {
    outboxRepository 		repository.IOutboxRepository
	openSearchRepository 	repository.IOpenSearchRepository
}

type IOutboxUseCase interface {
	ProcessOutbox(ctx context.Context, outBoxEvent entity.OutboxEvent) (events []entity.OutboxEvent, err error)
}

func NewOutboxUseCase(outboxRepository repository.IOutboxRepository,openSearchRepository repository.IOpenSearchRepository,) IOutboxUseCase {
	logger.InfoOutCtx("initializing outbox usecase SUCCESSFULLY")
	
	return &OutboxUseCase{
		outboxRepository: 		outboxRepository,
		openSearchRepository: 	openSearchRepository,
	}
}

func (u *OutboxUseCase) ProcessOutbox(ctx context.Context, outBoxEvent entity.OutboxEvent) (events []entity.OutboxEvent, err error) {
	logger.Info(ctx, "outbox usecase ProcessOutbox called", zap.String("event_type", outBoxEvent.EventType))

	ctx, span := tracing.CustomStartSpanCtx(ctx, "outboxUsecase.ProcessOutbox", trace.SpanKindInternal)
	defer span.End()

	events, err = u.outboxRepository.ProcessOutbox(ctx, entity.OutboxEvent{
		EventType: outBoxEvent.EventType,
	})
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if err = u.openSearchRepository.PostOutboxEvent(ctx, event); err != nil {
			markErr := u.outboxRepository.MarkEventFailed(ctx, event.ID, err.Error())
			if markErr != nil {
				return nil, fmt.Errorf("post opensearch error: %v; mark event failed error: %w", err, markErr)
			}
			return nil, err
		}

		if err = u.outboxRepository.MarkEventPublished(ctx, event.ID); err != nil {
			return nil, err
		}
	}

	return events, nil
}