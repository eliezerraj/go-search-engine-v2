package usecase

import (
	"context"
	
	"go.uber.org/zap"

	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/go-search-engine-v2/application/domain/entity"
	"github.com/go-search-engine-v2/application/infrastructure/repository"
	tracing "github.com/go-search-engine-v2/application/infrastructure/observability"

	"go.opentelemetry.io/otel/trace"
)

type OpenSearchUseCase struct {
	openSearchRepository repository.IOpenSearchRepository
	capabilityCatalogRepository repository.ICapabilityCatalogRepository
}

type IOpenSearchUseCase interface {
	OpenSearchSearch(ctx context.Context, query entity.Bm25SearchQuery) ([]entity.Bm25SearchResult, error)
}

func NewOpenSearchUseCase(
	openSearchRepository repository.IOpenSearchRepository,
	capabilityCatalogRepository repository.ICapabilityCatalogRepository,
) IOpenSearchUseCase {
	logger.InfoOutCtx("initializing opensearch usecase SUCCESSFULLY")

	return &OpenSearchUseCase{
		openSearchRepository: openSearchRepository,
		capabilityCatalogRepository: capabilityCatalogRepository,
	}
}

func (u *OpenSearchUseCase) OpenSearchSearch(ctx context.Context, query entity.Bm25SearchQuery) ([]entity.Bm25SearchResult, error) {
	logger.Info(ctx, "opensearch usecase OpenSearchSearch called", zap.Any("query", query))

	ctx, span := tracing.CustomStartSpanCtx(ctx, "openSearchUsecase.OpenSearchSearch", trace.SpanKindInternal)
	defer span.End()

	results, err := u.openSearchRepository.OpenSearchSearch(ctx, query)
	if err != nil {
		return nil, err
	}

	for i := range results {
		if results[i].ServerID == "" || results[i].Type == "" || results[i].CapabilityID == "" {
			continue
		}

		invocation, err := u.capabilityCatalogRepository.LoadInvocationSchema(
			ctx,
			entity.CapabilityInvocationKey{
				ServerID:       results[i].ServerID,
				CapabilityType: results[i].Type,
				Identifier:     results[i].CapabilityID,
			},
		)
		if err != nil {
			return nil, err
		}

		results[i].Invocation = invocation
	}

	return results, nil
}