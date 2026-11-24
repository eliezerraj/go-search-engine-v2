package application

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/eliezerraj/go-core/v3/logger"
	"github.com/eliezerraj/go-core/v3/database/connector"
	"github.com/eliezerraj/go-core/v3/httpclient"

	"github.com/go-semantic-engine-v2/application/infrastructure/repository"
	"github.com/go-semantic-engine-v2/application/config"
	"github.com/go-semantic-engine-v2/application/controller"
	"github.com/go-semantic-engine-v2/application/domain/usecase"
	"github.com/go-semantic-engine-v2/application/infrastructure/module"
)

type Application struct {
	SemanticController *controller.SemanticController
}

type UseCase struct {
	SemanticUseCase usecase.ISemanticUseCase
}

type Repository struct {
	SemanticRepository repository.ISemanticRepository
}

func NewApplication(cfg *config.Config) (*Application, error) {
	logger.InfoOutCtx("initializing application SUCCESSFULLY")

	// Initialize vector database connector
	readerConfigVector := connector.ConnectorConfig{
		DSN:              "postgres://" + cfg.Database.Username + ":" + cfg.Database.Password + "@" + cfg.Database.Host + ":" + cfg.Database.Port + "/" + cfg.Database.Name,
		MaxConnIdleTime:  cfg.Database.ConnIdleTime * time.Minute,
		MaxConnLifeTime:  cfg.Database.ConnLifetime * time.Minute,
		MaxConns:         cfg.Database.MaxConnections,
		MinConns:         cfg.Database.MinConnections,
		DBConnTimeout:    cfg.Database.ConnTimeout,
		HealthCheckPeriod: cfg.Database.ConnIdleTime * time.Minute / 2,
	}

	writerConfigVector := connector.ConnectorConfig{
		DSN:              "postgres://" + cfg.Database.Username + ":" + cfg.Database.Password + "@" + cfg.Database.Host + ":" + cfg.Database.Port + "/" + cfg.Database.Name,
		MaxConnIdleTime:  cfg.Database.ConnIdleTime * time.Minute,
		MaxConnLifeTime:  cfg.Database.ConnLifetime * time.Minute,
		MaxConns:         cfg.Database.MaxConnections,
		MinConns:         cfg.Database.MinConnections,
		DBConnTimeout:    cfg.Database.ConnTimeout,
		HealthCheckPeriod: cfg.Database.ConnIdleTime * time.Minute / 2,
	}

	// Initialize primary database connector
	readerConfig := connector.ConnectorConfig{
		DSN:              "postgres://" + cfg.Database2.Username + ":" + cfg.Database2.Password + "@" + cfg.Database2.Host + ":" + cfg.Database2.Port + "/" + cfg.Database2.Name,
		MaxConnIdleTime:  cfg.Database2.ConnIdleTime * time.Minute,
		MaxConnLifeTime:  cfg.Database2.ConnLifetime * time.Minute,
		MaxConns:         cfg.Database2.MaxConnections,
		MinConns:         cfg.Database2.MinConnections,
		DBConnTimeout:    cfg.Database2.ConnTimeout,
		HealthCheckPeriod: cfg.Database2.ConnIdleTime * time.Minute / 2,
	}

	writerConfig := connector.ConnectorConfig{
		DSN:              "postgres://" + cfg.Database2.Username + ":" + cfg.Database2.Password + "@" + cfg.Database2.Host + ":" + cfg.Database2.Port + "/" + cfg.Database2.Name,
		MaxConnIdleTime:  cfg.Database2.ConnIdleTime * time.Minute,
		MaxConnLifeTime:  cfg.Database2.ConnLifetime * time.Minute,
		MaxConns:         cfg.Database2.MaxConnections,
		MinConns:         cfg.Database2.MinConnections,
		DBConnTimeout:    cfg.Database2.ConnTimeout,
		HealthCheckPeriod: cfg.Database2.ConnIdleTime * time.Minute / 2,
	}

	logger.InfoOutCtx("Vector database connector initialized SUCCESSFULLY", zap.Any("readerConfig", readerConfigVector), zap.Any("writerConfig", writerConfigVector))
	logger.InfoOutCtx("Primary database connector initialized SUCCESSFULLY", zap.Any("readerConfig", readerConfig), zap.Any("writerConfig", writerConfig))

	dbConnectorVector, err := connector.NewDatabaseConnector(cfg.App.Name, readerConfigVector, writerConfigVector)
	if err != nil {
		logger.FatalOutCtx("failed to initialize database connector")
		return nil, err
	}
	
	logger.InfoOutCtx("dbConnectorVector initialized SUCCESSFULLY", zap.Any("dbConnector", dbConnectorVector))

	dbConnectorPrimary, err := connector.NewDatabaseConnector(cfg.App.Name, readerConfig, writerConfig)
	if err != nil {
		logger.FatalOutCtx("failed to initialize database connector")
		return nil, err
	}

	logger.InfoOutCtx("Primary dbConnectorPrimary initialized SUCCESSFULLY", zap.Any("dbConnector", dbConnectorPrimary))
	
	pgConnectionVector := &connector.PgConnection{}
	_, err = pgConnectionVector.NewPool(context.Background(), readerConfigVector)
	if err != nil {
		logger.FatalOutCtx("failed to create database vector pool")
		return nil, err
	}
	err = pgConnectionVector.Ping(context.Background())
	if err != nil {
		logger.FatalOutCtx("failed to ping pg pgConnectionVector")
		return nil, err
	}
	
	pgConnectionPrimary := &connector.PgConnection{}
	_, err = pgConnectionPrimary.NewPool(context.Background(), readerConfig)
	if err != nil {
		logger.FatalOutCtx("failed to create primary database pool")
		return nil, err
	}
	err = pgConnectionPrimary.Ping(context.Background())
	if err != nil {
		logger.FatalOutCtx("failed to ping pg pgConnectionPrimary")
		return nil, err
	}

	// Repository initialization (where the RSA keys are loaded and managed)
	semanticRepository := repository.NewSemanticRepository(dbConnectorVector)

	// Create the forwards modules.
	httpConfig := &httpclient.HttpConfig{
		Timeout:             cfg.HTTP.Timeout * time.Second,
		KeepAlive:           cfg.HTTP.KeepAlive * time.Second,
		IdleConnTimeout:     cfg.HTTP.IdleConnTimeout * time.Second,
		MaxIdleConns:        cfg.HTTP.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.HTTP.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.HTTP.MaxConnsPerHost,
		ServiceName:         cfg.App.Name,
	}

	modelHttpClient := httpclient.NewHttpClient(httpConfig)
	modelModule := module.NewModelModule(cfg, modelHttpClient, nil)

	// Repository initialization for the outbox module.
	outboxRepository := repository.NewOutboxRepository(dbConnectorPrimary)
	openSearchRepository := repository.NewOpenSearchRepository(cfg.OpenSearch)
	capabilityCatalogRepository := repository.NewCapabilityCatalogRepository(dbConnectorPrimary)

	// UseCase initialization	
	semanticUsecase := usecase.NewSemanticUseCase(semanticRepository, modelModule)
	outboxUsecase := usecase.NewOutboxUseCase(outboxRepository, openSearchRepository)
	openSearchUsecase := usecase.NewOpenSearchUseCase(openSearchRepository, capabilityCatalogRepository)

	// Controller initialization
	semanticController := controller.NewSemanticController(semanticUsecase, outboxUsecase, openSearchUsecase)

	return &Application{
		SemanticController: semanticController,
	}, nil
}
