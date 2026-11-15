package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/go-semantic-engine-v2/application/config"
	"github.com/go-semantic-engine-v2/application/domain/entity"
	tracing "github.com/go-semantic-engine-v2/application/infrastructure/observability"

	"go.opentelemetry.io/otel/trace"
)

const outboxEventIndexName = "outbox_event"

type OpenSearchRepository struct {
	endpoint string
	username string
	password string
	client   *http.Client
}

type IOpenSearchRepository interface {
	PostOutboxEvent(ctx context.Context, event entity.OutboxEvent) error
	GetOutboxEvent(ctx context.Context, id int64) (map[string]any, error)
}

func NewOpenSearchRepository(cfg config.OpenSearch) IOpenSearchRepository {
	logger.InfoOutCtx("initializing opensearch repository SUCCESSFULLY")

	return &OpenSearchRepository{
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
		username: cfg.Username,
		password: cfg.Password,
		client:   &http.Client{},
	}
}

func (r *OpenSearchRepository) PostOutboxEvent(ctx context.Context, event entity.OutboxEvent) error {
	ctx, span := tracing.CustomStartSpanCtx(ctx, "openSearchRepository.PostOutboxEvent", trace.SpanKindInternal)
	defer span.End()

	var payload any
	if len(event.Payload) > 0 {
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			logger.Error(ctx, "failed to unmarshal outbox payload", zap.Error(err), zap.Int64("outbox_event_id", event.ID))
			return err
		}
	}

	body := map[string]any{
		"id":             event.ID,
		"aggregate_type": event.AggregateType,
		"aggregate_id":   event.AggregateID,
		"event_type":     event.EventType,
		"payload":        payload,
		"created_at":     event.CreatedAt,
		"published_at":   event.PublishedAt,
		"attempts":       event.Attempts,
		"last_error":     event.LastError,
	}

	requestBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/%s/_doc/%d", r.endpoint, outboxEventIndexName, event.ID),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(r.username, r.password)

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opensearch post outbox event failed: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (r *OpenSearchRepository) GetOutboxEvent(ctx context.Context, id int64) (map[string]any, error) {
	ctx, span := tracing.CustomStartSpanCtx(ctx, "openSearchRepository.GetOutboxEvent", trace.SpanKindInternal)
	defer span.End()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/%s/_doc/%d", r.endpoint, outboxEventIndexName, id),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(r.username, r.password)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("opensearch get outbox event failed: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	response := map[string]any{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil
}
