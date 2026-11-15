package adapter

import (
	"fmt"
	"context"
	"go.uber.org/zap"

	"github.com/go-semantic-engine-v2/application/config"
	"github.com/go-semantic-engine-v2/application/infrastructure/application"
	"github.com/go-semantic-engine-v2/application/domain/external"
	tracing "github.com/go-semantic-engine-v2/application/infrastructure/observability"

	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/eliezerraj/go-core/v3/http/utils"

	"github.com/gofiber/fiber/v3"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/attribute"
)

type ApplicationAdapter struct {
	cfg *config.Config
	application *application.Application
}

func NewApplicationAdapter(cfg *config.Config, application *application.Application) *ApplicationAdapter {
	logger.InfoOutCtx("initializing application adapter SUCCESSFULLY")

	return &ApplicationAdapter{
		cfg:         cfg,
		application: application,
	}
}

func (a *ApplicationAdapter) SearchVector(ctxFiber fiber.Ctx) error {
	logger.Info(ctxFiber.Context(), "SearchVector called")
	
	ctxWithTimeout, cancel := context.WithTimeout(ctxFiber.Context(), a.cfg.HTTP.Timeout)
	defer cancel()

	// Tracing and metrics
	ctx, span := tracing.CustomStartSpanCtx(ctxWithTimeout, "applicationAdapter.searchVector", trace.SpanKindInternal)
	defer span.End()

	logger.Debug(
		ctx,
		a.cfg.App.Name,
		zap.ByteString("headers", utils.FormatHeadersAsJSON(ctxFiber.GetReqHeaders())),
		zap.String("host", ctxFiber.Hostname()),
		zap.String("path", ctxFiber.Path()),
		zap.ByteString("query", ctxFiber.Request().URI().QueryString()),
		zap.ByteString("body", ctxFiber.Body()),
	)

	searchVectorReq := external.VectorSearchRequest{}
	if err := ctxFiber.Bind().Body(&searchVectorReq); err != nil {
		logger.Error(ctx, "failed to parse request body", zap.Error(err))
		errorResponse := external.NewResponseError(ctx,
			fiber.StatusBadRequest,
			fiber.ErrBadRequest,
			fiber.ErrBadRequest.Message,
			"failed to parse request body",
			err.Error(),
			external.BUSSINESS_ERROR)
		span.SetAttributes(
				attribute.Int("http.response.status_code", fiber.StatusBadRequest),
		)
		return ctxFiber.Status(errorResponse.StatusCode).JSON(errorResponse)
	}

	res, err := a.application.SemanticController.SearchVector(ctx, searchVectorReq)
	if err != nil {
		logger.Error(ctx, "failed to search vector", zap.Error(err))
		errorResponse := external.NewResponseError(ctx,
			fiber.StatusInternalServerError,
			fiber.ErrInternalServerError,
			fiber.ErrInternalServerError.Message,
			"failed to search vector",
			err.Error(),
			external.BUSSINESS_ERROR)
		span.SetAttributes(
				attribute.Int("http.response.status_code", fiber.StatusInternalServerError),
		)
		return ctxFiber.Status(errorResponse.StatusCode).JSON(errorResponse)
	}

	resp := external.VectorSearchResponse{
		Response: "Vector search successful",
		Vector:   res,
	}

	return ctxFiber.Status(fiber.StatusOK).JSON(resp)
}

func (a *ApplicationAdapter) IntentDecompose(ctxFiber fiber.Ctx) error {
	logger.Info(ctxFiber.Context(), "IntentDecompose called")
	
	ctxWithTimeout, cancel := context.WithTimeout(ctxFiber.Context(), a.cfg.HTTP.Timeout)
	defer cancel()

	// Tracing and metrics
	ctx, span := tracing.CustomStartSpanCtx(ctxWithTimeout, "applicationAdapter.intentDecompose", trace.SpanKindInternal)
	defer span.End()

	logger.Debug(
		ctx,
		a.cfg.App.Name,
		zap.ByteString("headers", utils.FormatHeadersAsJSON(ctxFiber.GetReqHeaders())),
		zap.String("host", ctxFiber.Hostname()),
		zap.String("path", ctxFiber.Path()),
		zap.ByteString("query", ctxFiber.Request().URI().QueryString()),
		zap.ByteString("body", ctxFiber.Body()),
	)

	intentDecomposeReq := external.IntentDecomposeRequest{}
	if err := ctxFiber.Bind().Body(&intentDecomposeReq); err != nil {
		logger.Error(ctx, "failed to parse request body", zap.Error(err))
		errorResponse := external.NewResponseError(ctx,
			fiber.StatusBadRequest,
			fiber.ErrBadRequest,
			fiber.ErrBadRequest.Message,
			"failed to parse request body",
			err.Error(),
			external.BUSSINESS_ERROR)
		span.SetAttributes(
				attribute.Int("http.response.status_code", fiber.StatusBadRequest),
		)
		return ctxFiber.Status(errorResponse.StatusCode).JSON(errorResponse)
	}

	// Pass the parsed request to the controller
	res, err := a.application.SemanticController.IntentDecompose(ctx, intentDecomposeReq)
	if err != nil {
		logger.Error(ctx, "failed to decompose intent", zap.Error(err))
		errorResponse := external.NewResponseError(ctx,
			fiber.StatusInternalServerError,
			fiber.ErrInternalServerError,
			fiber.ErrInternalServerError.Message,
			"failed to decompose intent",
			err.Error(),
			external.BUSSINESS_ERROR)
		span.SetAttributes(
				attribute.Int("http.response.status_code", fiber.StatusInternalServerError),
		)
		return ctxFiber.Status(errorResponse.StatusCode).JSON(errorResponse)
	}

	resp := external.IntentDecomposeResponse{
		Response: "Intent decomposition successful",
		Intent:   res,
	}

	return ctxFiber.Status(fiber.StatusOK).JSON(resp)
}

func (a *ApplicationAdapter) ProcessOutbox(ctxFiber fiber.Ctx) error {
	var message string
	logger.Info(ctxFiber.Context(), "ProcessOutbox called")

	ctxWithTimeout, cancel := context.WithTimeout(ctxFiber.Context(), a.cfg.HTTP.Timeout)
	defer cancel()

	// Tracing and metrics
	ctx, span := tracing.CustomStartSpanCtx(ctxWithTimeout, "applicationAdapter.processOutbox", trace.SpanKindInternal)
	defer span.End()

	logger.Debug(
		ctx,
		a.cfg.App.Name,
		zap.ByteString("headers", utils.FormatHeadersAsJSON(ctxFiber.GetReqHeaders())),
		zap.String("host", ctxFiber.Hostname()),
		zap.String("path", ctxFiber.Path()),
		zap.ByteString("query", ctxFiber.Request().URI().QueryString()),
		zap.ByteString("body", ctxFiber.Body()),
	)

	outBoxEvent := external.OutboxEventRequest{}
	if err := ctxFiber.Bind().Body(&outBoxEvent); err != nil {
		logger.Error(ctx, "failed to parse request body", zap.Error(err))
		errorResponse := external.NewResponseError(ctx,
			fiber.StatusBadRequest,
			fiber.ErrBadRequest,
			fiber.ErrBadRequest.Message,
			"failed to parse request body",
			err.Error(),
			external.BUSSINESS_ERROR)
		span.SetAttributes(
				attribute.Int("http.response.status_code", fiber.StatusBadRequest),
		)
		return ctxFiber.Status(errorResponse.StatusCode).JSON(errorResponse)
	}

	// Pass the parsed request to the controller
	res, err := a.application.SemanticController.ProcessOutbox(ctx, outBoxEvent)
	if err != nil {
		logger.Error(ctx, "failed to process outbox", zap.Error(err))
		errorResponse := external.NewResponseError(ctx,
			fiber.StatusInternalServerError,
			fiber.ErrInternalServerError,
			fiber.ErrInternalServerError.Message,
			"failed to process outbox",
			err.Error(),
			external.BUSSINESS_ERROR)
		span.SetAttributes(
				attribute.Int("http.response.status_code", fiber.StatusInternalServerError),
		)
		return ctxFiber.Status(errorResponse.StatusCode).JSON(errorResponse)
	}

	message = "Outbox processed successfully"

	if res == nil {
		message = "No outbox event to process for this request {outBoxEvent: " + fmt.Sprintf("%v", outBoxEvent) + "}"
	}
	resp := external.OutBoxEventResponse{
		Response: message,
		Event:    res,
	}

	return ctxFiber.Status(fiber.StatusOK).JSON(resp)
}