package controller

import (
	"context"

	"go.uber.org/zap"

	"github.com/eliezerraj/go-core/v3/logger"

	"github.com/go-semantic-engine-v2/application/domain/usecase"
	"github.com/go-semantic-engine-v2/application/domain/external"
	"github.com/go-semantic-engine-v2/application/domain/entity"
	tracing "github.com/go-semantic-engine-v2/application/infrastructure/observability"
	"github.com/go-semantic-engine-v2/application/controller/validator"

	"go.opentelemetry.io/otel/trace"
)

type SemanticController struct {
	schema validator.Schema
	SemanticUseCase usecase.ISemanticUseCase
	OutboxUseCase usecase.IOutboxUseCase
}

// NewSemanticController creates a new instance of SemanticController with the provided login use case.
func NewSemanticController(semanticUseCase usecase.ISemanticUseCase, outboxUseCase usecase.IOutboxUseCase) *SemanticController {
	logger.InfoOutCtx("initializing semantic controller SUCCESSFULLY")

	schema := validator.Schema{
			Validate: func(ctx context.Context, data any) error {
				return nil
			},
		}

	return &SemanticController{
		schema:       schema,
		SemanticUseCase: semanticUseCase,
		OutboxUseCase:  outboxUseCase,	
	}
}

func (p *SemanticController) SearchVector(ctx context.Context, req external.VectorSearchRequest) (*entity.VectorSearch, error) {
	logger.Info(ctx, "semantic controller SearchVector called")

	// Tracing and metrics
	ctx, span := tracing.CustomStartSpanCtx(ctx, "semanticController.searchVector", trace.SpanKindInternal)
	defer span.End()

	res, err := p.SemanticUseCase.SearchVector(ctx, entity.VectorSearch{
		Text: req.Text,
	})
	if err != nil {
		logger.Debug(ctx, "error executing SearchVector controller", zap.Error(err))
		return nil, err
	}

	return res, nil
}

func (p *SemanticController) IntentDecompose(ctx context.Context, req external.IntentDecomposeRequest) (*entity.IntentDecompose, error) {
	logger.Info(ctx, "semantic controller IntentDecompose called")

	// Tracing and metrics
	ctx, span := tracing.CustomStartSpanCtx(ctx, "semanticController.intentDecompose", trace.SpanKindInternal)
	defer span.End()

	res, err := p.SemanticUseCase.IntentDecompose(ctx, entity.IntentDecompose{
		Intent: req.Intent,
	})
	if err != nil {
		logger.Debug(ctx, "error executing IntentDecompose controller", zap.Error(err))
		return nil, err
	}

	return res, nil
}

func (p *SemanticController) ProcessOutbox(ctx context.Context, req external.OutboxEventRequest) ([]entity.OutboxEvent, error) {
	logger.Info(ctx, "semantic controller ProcessOutbox called")

	// Tracing and metrics
	ctx, span := tracing.CustomStartSpanCtx(ctx, "semanticController.processOutbox", trace.SpanKindInternal)
	defer span.End()

	res, err := p.OutboxUseCase.ProcessOutbox(ctx, entity.OutboxEvent{
		EventType: req.EventType,
	})
	if err != nil {
		logger.Debug(ctx, "error executing ProcessOutbox controller", zap.Error(err))
		return nil, err
	}

	return res, nil
}