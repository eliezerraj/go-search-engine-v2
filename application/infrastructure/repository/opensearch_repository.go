package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/go-semantic-engine-v2/application/config"
	"github.com/go-semantic-engine-v2/application/domain/entity"
	"github.com/go-semantic-engine-v2/application/infrastructure/observability"
	"github.com/go-semantic-engine-v2/application/domain/apperrors"

	"go.opentelemetry.io/otel/trace"
)

const outboxEventIndexName = "outbox_event"

type capabilityCandidate struct {
	CapabilityID string
	Type         string
	Name         string
	ServerID     string
}

type OpenSearchRepository struct {
	endpoint string
	username string
	password string
	client   *http.Client
}

type IOpenSearchRepository interface {
	PostOutboxEvent(ctx context.Context, event entity.OutboxEvent) error
	GetOutboxEvent(ctx context.Context, id int64) (map[string]any, error)
	OpenSearchSearch(ctx context.Context, query entity.Bm25SearchQuery) ([]entity.Bm25SearchResult, error)
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
	logger.Info(ctx, "opensearch repository PostOutboxEvent called", zap.Any("outbox_event_id", event.ID))

	ctx, span := observability.CustomStartSpanCtx(ctx, "openSearchRepository.PostOutboxEvent", trace.SpanKindInternal)
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
	logger.Info(ctx, "opensearch repository GetOutboxEvent called", zap.Any("outbox_event_id", id))

	ctx, span := observability.CustomStartSpanCtx(ctx, "openSearchRepository.GetOutboxEvent", trace.SpanKindInternal)
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

func (r *OpenSearchRepository) OpenSearchSearch(ctx context.Context, query entity.Bm25SearchQuery) ([]entity.Bm25SearchResult, error) {
	logger.Info(ctx, "opensearch repository OpenSearchSearch called", zap.Any("query", query))
	
	ctx, span := observability.CustomStartSpanCtx(ctx, "openSearchRepository.OpenSearchSearch", trace.SpanKindInternal)
	defer span.End()

	limit := query.Limit
	if limit <= 0 {
		limit = 5
	}

	// Use fuzziness and avoid requiring every term to match exactly
	//  - "fuzziness": "AUTO" allows for minor typos
	//  - "minimum_should_match": "70%" ensures that not all terms need to match exactly
	body := map[string]any{
		"size": limit * 5,
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  query.Query,
				"fields": []string{
					"payload.documents.source.search_text^4",
					"payload.documents.source.description^3",
					"payload.documents.source.name^2",
					"payload.documents.source.identifier^2",
					"payload.documents.source.input_fields",
				},
				"type": "best_fields",
				"operator": "or",
				"minimum_should_match": "70%",
				"fuzziness": "AUTO",
				"prefix_length": 1,
			},
		},
	}

	requestBody, err := json.Marshal(body)
	if err != nil {
		logger.Error(ctx, "failed to marshal OpenSearch search request body", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/%s/_search", r.endpoint, outboxEventIndexName),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		logger.Error(ctx, "failed to create OpenSearch search request", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(r.username, r.password)

	fmt.Println("----------------------------- OpenSearch Search Request -----------------------------")
	fmt.Println("Headers:", req.Header)
	fmt.Println("URL:", fmt.Sprintf("%s/%s/_search", r.endpoint, outboxEventIndexName))
	fmt.Println("Body:", string(requestBody))
	fmt.Println("--------------------------------------------------------------------------------")

	resp, err := r.client.Do(req)
	if err != nil {
		logger.Error(ctx, "failed to execute OpenSearch search request", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error(ctx, "opensearch search failed", zap.Any("status", resp.StatusCode))
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("opensearch search failed: status=%d: %w", resp.StatusCode, apperrors.ErrNotFound)
		}
		return nil, fmt.Errorf("opensearch search failed: status=%d", resp.StatusCode)
	}

	fmt.Println("----------------------------- OpenSearch Search Response -----------------------------")
	fmt.Println("Status Code:", resp.StatusCode)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Response Body:")
	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Println(string(bodyBytes))
	fmt.Println("--------------------------------------------------------------------------------")
	// Reset the response body for further reading
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	type hit struct {
		Score  float64        `json:"_score"`
		Source map[string]any `json:"_source"`
	}
	type searchResponse struct {
		Hits struct {
			Hits []hit `json:"hits"`
		} `json:"hits"`
	}

	parsed := searchResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		logger.Error(ctx, "failed to parse OpenSearch search response", zap.Error(err))
		return nil, err
	}

	typeFilter := map[string]struct{}{}
	for _, t := range query.CapabilityTypes {
		typeFilter[strings.ToUpper(strings.TrimSpace(t))] = struct{}{}
	}

	serverFilter := map[string]struct{}{}
	for _, s := range query.ServerIds {
		serverFilter[strings.TrimSpace(s)] = struct{}{}
	}

	results := make([]entity.Bm25SearchResult, 0, limit)
	seen := map[string]struct{}{}
	for _, h := range parsed.Hits.Hits {
		candidates := extractCapabilities(h.Source)
		for _, candidate := range candidates {
			if candidate.CapabilityID == "" || candidate.Type == "" || candidate.Name == "" {
				continue
			}

			if len(typeFilter) > 0 {
				if _, ok := typeFilter[strings.ToUpper(candidate.Type)]; !ok {
					continue
				}
			}

			if len(serverFilter) > 0 {
				if _, ok := serverFilter[candidate.ServerID]; !ok {
					continue
				}
			}

			if _, exists := seen[candidate.CapabilityID]; exists {
				continue
			}
			seen[candidate.CapabilityID] = struct{}{}

			results = append(results, entity.Bm25SearchResult{
				CapabilityID: candidate.CapabilityID,
				Type:         strings.ToUpper(candidate.Type),
				Name:         candidate.Name,
				ServerID:     candidate.ServerID,
				Score:        h.Score,
			})

			if len(results) >= limit {
				break
			}
		}

		if len(results) >= limit {
			break
		}
	}

	return results, nil
}


func extractCapabilities(source map[string]any) []capabilityCandidate {
	result := make([]capabilityCandidate, 0, 8)

	appendCandidate := func(raw map[string]any) {
		candidate := capabilityCandidate{
			CapabilityID: firstString(raw, "identifier", "capabilityId", "capability_id", "id"),
			Type:         firstString(raw, "type", "capabilityType", "capability_type"),
			Name:         firstString(raw, "name"),
			ServerID:     firstString(raw, "serverId", "server_id"),
		}

		if candidate.ServerID == "" && candidate.CapabilityID != "" {
			if idx := strings.Index(candidate.CapabilityID, ":"); idx > 0 {
				candidate.ServerID = candidate.CapabilityID[:idx]
			}
		}

		result = append(result, candidate)
	}

	appendCandidate(source)

	payload, ok := source["payload"].(map[string]any)
	if ok {
		appendCandidate(payload)

		documents, hasDocuments := payload["documents"].([]any)
		if hasDocuments {
			for _, item := range documents {
				document, isMap := item.(map[string]any)
				if !isMap {
					continue
				}

				if sourceRaw, hasSource := document["source"].(map[string]any); hasSource {
					appendCandidate(sourceRaw)
					continue
				}

				appendCandidate(document)
			}
		}
	}

	return result
}

func firstString(source map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := source[key]
		if !ok || value == nil {
			continue
		}

		switch typed := value.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return trimmed
			}
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64)
		case int:
			return strconv.Itoa(typed)
		case int64:
			return strconv.FormatInt(typed, 10)
		}
	}

	return ""
}
