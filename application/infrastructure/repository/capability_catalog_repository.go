package repository

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/eliezerraj/go-core/v3/database/connector"
	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/go-search-engine-v2/application/domain/entity"
	tracing "github.com/go-search-engine-v2/application/infrastructure/observability"

	"go.opentelemetry.io/otel/trace"
)

type CapabilityCatalogRepository struct {
	dbConnector connector.IDatabaseConnector
}

type ICapabilityCatalogRepository interface {
	LoadInvocationSchema(ctx context.Context, key entity.CapabilityInvocationKey) (json.RawMessage, error)
}

func NewCapabilityCatalogRepository(dbConnector connector.IDatabaseConnector) ICapabilityCatalogRepository {
	logger.InfoOutCtx("initializing capability catalog repository SUCCESSFULLY")

	return &CapabilityCatalogRepository{
		dbConnector: dbConnector,
	}
}

func (r *CapabilityCatalogRepository) LoadInvocationSchema(ctx context.Context, key entity.CapabilityInvocationKey) (json.RawMessage, error) {
	logger.Info(
		ctx,
		"capability catalog repository LoadInvocationSchema called",
		zap.String("server_id", key.ServerID),
		zap.String("capability_type", key.CapabilityType),
		zap.String("identifier", key.Identifier),
	)

	ctx, span := tracing.CustomStartSpanCtx(ctx, "capabilityCatalogRepository.LoadInvocationSchema", trace.SpanKindInternal)
	defer span.End()

	query := `
		SELECT payload -> 'inputSchema' AS invocation
		FROM public.capability_catalog
		WHERE enabled = true
		  AND provider_id = $1
		  AND UPPER(capability_type) = UPPER($2)
		  AND identifier = $3
		LIMIT 1`

	rows, err := r.dbConnector.Reader().Query(
		ctx,
		query,
		key.ServerID,
		key.CapabilityType,
		key.Identifier,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var invocationBytes []byte
	if err := rows.Scan(&invocationBytes); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(invocationBytes) == 0 || string(invocationBytes) == "null" {
		return nil, nil
	}

	return json.RawMessage(invocationBytes), nil
}
