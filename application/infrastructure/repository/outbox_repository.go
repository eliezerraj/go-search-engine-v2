package repository

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"github.com/eliezerraj/go-core/v3/database/connector"
	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/go-semantic-engine-v2/application/domain/entity"
	tracing "github.com/go-semantic-engine-v2/application/infrastructure/observability"

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
	ProcessOutbox(ctx context.Context, outBoxEvent entity.OutboxEvent) ([]entity.OutboxEvent, error)
	MarkEventPublished(ctx context.Context, id int64) error
	MarkEventFailed(ctx context.Context, id int64, lastError string) error
}

func NewOutboxRepository(dbConnector connector.IDatabaseConnector) IOutboxRepository {
	logger.InfoOutCtx("initializing outbox repository SUCCESSFULLY")
	
	return &OutboxRepository{
		dbConnector: dbConnector,
	}
}

func (r *OutboxRepository) ProcessOutbox(ctx context.Context, outBoxEvent entity.OutboxEvent) (events []entity.OutboxEvent, err error) {
	logger.Info(ctx, "outbox repository ProcessOutbox called", zap.String("event_type", outBoxEvent.EventType))

	ctx, span := tracing.CustomStartSpanCtx(ctx, "outboxRepository.ProcessOutbox", trace.SpanKindInternal)
	defer span.End()

	meter := otel.Meter("go-semantic-engine-v2.repository")
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
			  where event_type = $1
			  order by id asc`

	rows, err := r.dbConnector.Reader().Query(ctx, query, outBoxEvent.EventType)
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

func (r *OutboxRepository) MarkEventPublished(ctx context.Context, id int64) error {
	logger.Info(ctx, "outbox repository MarkEventPublished called", zap.Int64("outbox_event_id", id))

	query := `update outbox_event
			  set published_at = now(),
			      attempts = attempts + 1,
			      last_error = null
			  where id = $1`

	_, err := r.dbConnector.Writer().Exec(ctx, query, id)
	return err
}

func (r *OutboxRepository) MarkEventFailed(ctx context.Context, id int64, lastError string) error {
	logger.Info(ctx, "outbox repository MarkEventFailed called", zap.Int64("outbox_event_id", id))

	query := `update outbox_event
			  set attempts = attempts + 1,
			      last_error = $2
			  where id = $1`

	_, err := r.dbConnector.Writer().Exec(ctx, query, id, lastError)
	return err
}