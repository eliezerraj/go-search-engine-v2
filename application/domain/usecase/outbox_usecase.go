package usecase

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"

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

// BeginTx starts a new database transaction with the specified options.
func (u	*OutboxUseCase) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	logger.Info(ctx, "outbox usecase BeginTx called")

	tx, err := u.outboxRepository.BeginTx(ctx, opts)
	if err != nil {
		logger.Error(ctx, "outbox usecase BeginTx failed", zap.Error(err))
		return nil, err
	}

	return tx, nil
}

func (u *OutboxUseCase) ProcessOutbox(ctx context.Context, outBoxEvent entity.OutboxEvent) (events []entity.OutboxEvent, err error) {
	logger.Info(ctx, "outbox usecase ProcessOutbox called", zap.String("aggregate_type", outBoxEvent.AggregateType))

	ctx, span := tracing.CustomStartSpanCtx(ctx, "outboxUsecase.ProcessOutbox", trace.SpanKindInternal)
	defer span.End()

	// Start a new transaction
	tx, err := u.outboxRepository.BeginTx(ctx, pgx.TxOptions{ IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite })
	if err != nil {
		logger.Error(ctx, "outbox usecase ProcessOutbox failed to begin transaction", zap.Error(err))
		return nil, err
	}

	// Ensure that the transaction is either committed or rolled back
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
				logger.Error(ctx, "outbox usecase ProcessOutbox failed to rollback transaction", zap.Error(rollbackErr))
			}
		} else {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				logger.Error(ctx, "outbox usecase ProcessOutbox failed to commit transaction", zap.Error(commitErr))
				err = commitErr
			}
		}
	}()

	events, err = u.outboxRepository.ProcessOutbox(ctx, tx, entity.OutboxEvent{
		AggregateType: outBoxEvent.AggregateType,
	})
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if err = u.openSearchRepository.PostOutboxEvent(ctx, event); err != nil {
			markErr := u.outboxRepository.MarkEventFailed(ctx, tx, event, err.Error())
			if markErr != nil {
				return nil, fmt.Errorf("post opensearch error: %v; mark event failed error: %w", err, markErr)
			}
			return nil, err
		}

		if err = u.outboxRepository.MarkEventPublished(ctx, tx, event); err != nil {
			return nil, err
		}
	}

	return events, nil
}