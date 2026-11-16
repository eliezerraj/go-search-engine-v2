package usecase

import (
	"context"
	
	"go.uber.org/zap"

	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/go-semantic-engine-v2/application/domain/entity"
	"github.com/go-semantic-engine-v2/application/infrastructure/repository"
	tracing "github.com/go-semantic-engine-v2/application/infrastructure/observability"

	"go.opentelemetry.io/otel/trace"
)

type OpenSearchUseCase struct {
	openSearchRepository repository.IOpenSearchRepository
}

type IOpenSearchUseCase interface {
	OpenSearchSearch(ctx context.Context, query entity.Bm25SearchQuery) ([]entity.Bm25SearchResult, error)
}

func NewOpenSearchUseCase(openSearchRepository repository.IOpenSearchRepository) IOpenSearchUseCase {
	logger.InfoOutCtx("initializing opensearch usecase SUCCESSFULLY")

	return &OpenSearchUseCase{
		openSearchRepository: openSearchRepository,
	}
}

func (u *OpenSearchUseCase) OpenSearchSearch(ctx context.Context, query entity.Bm25SearchQuery) ([]entity.Bm25SearchResult, error) {
	logger.Info(ctx, "opensearch usecase OpenSearchSearch called", zap.Any("query", query))

	ctx, span := tracing.CustomStartSpanCtx(ctx, "openSearchUsecase.OpenSearchSearch", trace.SpanKindInternal)
	defer span.End()

	return u.openSearchRepository.OpenSearchSearch(ctx, query)
}