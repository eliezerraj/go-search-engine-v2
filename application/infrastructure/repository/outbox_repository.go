package repository

import (
	"context"
	"encoding/json"
	"time"
	"github.com/jackc/pgx/v5"

	"go.uber.org/zap"
	"github.com/eliezerraj/go-core/v3/database/connector"
	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/go-search-engine-v2/application/domain/entity"
	tracing "github.com/go-search-engine-v2/application/infrastructure/observability"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type OutboxRepository struct{
	dbConnector connector.IDatabaseConnector
}

type IOutboxRepository interface {
	ProcessOutbox(ctx context.Context, tx pgx.Tx, outBoxEvent entity.OutboxEvent) ([]entity.OutboxEvent, error)
	MarkEventPublished(ctx context.Context, tx pgx.Tx, outBoxEvent entity.OutboxEvent) error
	MarkEventFailed(ctx context.Context, tx pgx.Tx, outBoxEvent entity.OutboxEvent, lastError string) error
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

func NewOutboxRepository(dbConnector connector.IDatabaseConnector) IOutboxRepository {
	logger.InfoOutCtx("initializing outbox repository SUCCESSFULLY")
	
	return &OutboxRepository{
		dbConnector: dbConnector,
	}
}

func (r *OutboxRepository) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	logger.Info(ctx, "outbox repository BeginTx called")

	tx, err := r.dbConnector.Writer().BeginTx(ctx, opts)
	if err != nil {
		logger.Debug(ctx, "outbox repository BeginTx failed", zap.Error(err))
		return nil, err
	}

	return tx, nil
}

func (r *OutboxRepository) ProcessOutbox(ctx context.Context, tx pgx.Tx, outBoxEvent entity.OutboxEvent) (events []entity.OutboxEvent, err error) {
	logger.Info(ctx, "outbox repository ProcessOutbox called", zap.String("aggregate_type", outBoxEvent.AggregateType))

	ctx, span := tracing.CustomStartSpanCtx(ctx, "outboxRepository.ProcessOutbox", trace.SpanKindInternal)
	defer span.End()

	meter := otel.Meter("go-search-engine-v2.repository")
	counter, _ := meter.Int64Counter("db_outbox_consume_requests_total")
	histogram, _ := meter.Float64Histogram("db_outbox_consume_duration_seconds")
	start := time.Now()

	counter.Add(ctx, 1, metric.WithAttributes(attribute.String("operation", "ProcessOutbox")))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.Debug(ctx, "outbox repository ProcessOutbox failed", zap.Error(err))
		}
		histogram.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attribute.String("operation", "ProcessOutbox")))
	}()

	query := `select id, aggregate_type, aggregate_id, event_type, payload, created_at, published_at, attempts, last_error
			  from outbox_event
			  where aggregate_type = $1
			  order by id asc`

	rows, err := r.dbConnector.Reader().Query(ctx, query, outBoxEvent.AggregateType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		event := entity.OutboxEvent{}
		var payload []byte
		if err = rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&payload,
			&event.CreatedAt,
			&event.PublishedAt,
			&event.Attempts,
			&event.LastError,
		); err != nil {
			return nil, err
		}
		event.Payload = json.RawMessage(payload)
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *OutboxRepository) MarkEventPublished(ctx context.Context, tx pgx.Tx, outBoxEvent entity.OutboxEvent) error {
	logger.Info(ctx, "outbox repository MarkEventPublished called", zap.Int64("outbox_event_id", outBoxEvent.ID))

	ctx, span := tracing.CustomStartSpanCtx(ctx, "outboxRepository.MarkEventPublished", trace.SpanKindInternal)
	defer span.End()

	query := `update outbox_event
			  set published_at = now(),
			      attempts = attempts + 1,
			      last_error = null
			  where id = $1`

	_, err := r.dbConnector.Writer().Exec(ctx, query, outBoxEvent.ID)
	return err
}

func (r *OutboxRepository) MarkEventFailed(ctx context.Context, tx pgx.Tx, outBoxEvent entity.OutboxEvent, lastError string) error {
	logger.Info(ctx, "outbox repository MarkEventFailed called", zap.Int64("outbox_event_id", outBoxEvent.ID))

	ctx, span := tracing.CustomStartSpanCtx(ctx, "outboxRepository.MarkEventFailed", trace.SpanKindInternal)
	defer span.End()

	query := `update outbox_event
			  set attempts = attempts + 1,
			      last_error = $2
			  where id = $1`

	_, err := r.dbConnector.Writer().Exec(ctx, query, outBoxEvent.ID, lastError)
	return err
}