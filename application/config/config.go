package config

import (
	"time"
	"fmt"

	"github.com/eliezerraj/go-core/v3/logger"

	"github.com/caarlos0/env/v8"
	"github.com/joho/godotenv"
)

type HTTP struct {
	Timeout               time.Duration `env:"HTTP_TIMEOUT" envDefault:"30s"`
	KeepAlive             time.Duration `env:"HTTP_KEEP_ALIVE" envDefault:"30s"`
	ReadTimeout           time.Duration `env:"HTTP_READTIMEOUT"`
	WriteTimeout          time.Duration `env:"HTTP_WRITETIMEOUT"`
	IdleConnTimeout       time.Duration `env:"HTTP_IDLECONNTIMEOUT"`
	MaxIdleConns          int           `env:"HTTP_MAXIDLECONNS"`
	MaxIdleConnsPerHost   int           `env:"HTTP_MAXIDLECONNSPERHOST"`
	MaxConnsPerHost       int           `env:"HTTP_MAXCONNSPERHOST"`
	Port                  string        `env:"HTTP_PORT,required,notEmpty"`
	ReadBufferSize        int           `env:"READ_BUFFER_SIZE" envDefault:"12288"`
	DisableStartupMessage bool          `env:"DISABLE_STARTUP_MESSAGE" envDefault:"false"`
}

type Database struct {
	DSN                   string
	Host                  string        `env:"DB_HOST,required,notEmpty"`
	Port                  string        `env:"DB_PORT,required,notEmpty"`
	Username              string        `env:"DB_USERNAME,required,notEmpty"`
	Password              string        `env:"DB_PASSWORD,required,notEmpty"`
	Name                  string        `env:"DB_NAME,required,notEmpty"`
	Schema                string        `env:"DB_SCHEMA" envDefault:"postgres"`
	QueryTracer           bool          `env:"DATABASE_QUERY_TRACER" envDefault:"false"`
	MaxConnections        int32         `env:"DB_MAX_CONNS" envDefault:"5"`
	MinConnections        int32         `env:"DB_MIN_CONNS" envDefault:"5"`
	ConnTimeout           time.Duration `env:"DB_CONNECTION_TIMEOUT_DURATION" envDefault:"5s"`
	ConnLifetime          time.Duration `env:"DB_LIFE_TIME_CONNS" envDefault:"28s"`
	ConnIdleTime          time.Duration `env:"DB_IDLE_TIME_CONNS" envDefault:"28s"`
	MaxConnLifetimeJitter time.Duration `env:"DB_MAX_LIFE_TIME_JITTER_CONNS" envDefault:"2m"`
	HealthCheckTimeout    time.Duration `env:"DB_HEALTH_CHECK_TIMEOUT" envDefault:"15s"`
}

type Database2 struct {
	DSN                   string
	Host                  string        `env:"DB_HOST_2,required,notEmpty"`
	Port                  string        `env:"DB_PORT_2,required,notEmpty"`
	Username              string        `env:"DB_USERNAME_2,required,notEmpty"`
	Password              string        `env:"DB_PASSWORD_2,required,notEmpty"`
	Name                  string        `env:"DB_NAME_2,required,notEmpty"`
	Schema                string        `env:"DB_SCHEMA_2" envDefault:"postgres"`
	QueryTracer           bool          `env:"DATABASE_QUERY_TRACER_2" envDefault:"false"`
	MaxConnections        int32         `env:"DB_MAX_CONNS_2" envDefault:"5"`
	MinConnections        int32         `env:"DB_MIN_CONNS_2" envDefault:"5"`
	ConnTimeout           time.Duration `env:"DB_CONNECTION_TIMEOUT_DURATION_2" envDefault:"5s"`
	ConnLifetime          time.Duration `env:"DB_LIFE_TIME_CONNS_2" envDefault:"28s"`
	ConnIdleTime          time.Duration `env:"DB_IDLE_TIME_CONNS_2" envDefault:"28s"`
	MaxConnLifetimeJitter time.Duration `env:"DB_MAX_LIFE_TIME_JITTER_CONNS_2" envDefault:"2m"`
	HealthCheckTimeout    time.Duration `env:"DB_HEALTH_CHECK_TIMEOUT_2" envDefault:"15s"`
}

type OpenSearch struct {
	Endpoint string `env:"OPENSEARCH_ENDPOINT"`
	Username string `env:"OPENSEARCH_USERNAME"`
	Password string `env:"OPENSEARCH_PASSWORD"`
}

type Authorization struct {
	PolicyPath			string `env:"AUTHORIZATION_POLICY_PATH"`
	HeaderKey 			string `env:"AUTHORIZATION_HEADER_KEY,required"`
	DryRun  			bool   `env:"AUTHORIZATION_DRY_RUN,required"`
	AuthorizationServer string `env:"AUTHORIZATION_SERVER"`
	JwksURL 			string `env:"AUTHORIZATION_JWKS_URL"`
	RequiredAuthHeader  bool   `env:"AUTHORIZATION_REQUIRED_HEADER"`
	Audience 			string `env:"AUTHORIZATION_AUDIENCE"`
	ResourceMetadata 	string `env:"AUTHORIZATION_OAUTH_PROTECTED_RESOURCE_URL"`
	Timeout             time.Duration `env:"AUTHORIZATION_TIMEOUT" envDefault:"15s"`
}

type App struct {
	Name        string `env:"APP_NAME"`
	Version     string `env:"VERSION" envDefault:"no-version"`
	Env		  	string `env:"ENV" envDefault:"dev"`
	Account	 	string `env:"ACCOUNT" envDefault:"local:localhost"`
	Type      	string `env:"TYPE" envDefault:"webserver"`
}

type Log struct {
	Level string             `env:"LOG_LEVEL" envDefault:"INFO"`
	Mode  logger.EncoderType `env:"LOG_MODE" envDefault:"json"`
}

type VectorEndpoint struct {
	Endpoint string        `env:"VECTOR_ENDPOINT"`
	UrlPath  string        `env:"VECTOR_URL_PATH"`
	Timeout  time.Duration `env:"VECTOR_ENDPOINT_TIMEOUT"`
}

type Config struct {
	App         App
	HTTP        HTTP
	Database    Database
	Database2   Database2
	Authorization Authorization
	Log         Log
	OtelEnv		OtelEnv
	Vector      VectorEndpoint
	OpenSearch  OpenSearch
}

type OtelEnv struct {
	OtelExportEndpoint			string	`env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"127.0.0.1:4317"`
	OtelMetricsPort				string	`env:"OTEL_METRICS_PORT"`
}

func Load() (cfg *Config, err error) {
	cfg = &Config{}
	_ = godotenv.Load(".env")

	err = env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	cfg.Database.DSN = fmt.Sprintf(
		`host=%s port=%s database=%s search_path=%s user=%s password=%s connect_timeout=%d pool_min_conns=%d pool_max_conns=%d pool_max_conn_idle_time=%s pool_max_conn_lifetime=%s pool_max_conn_lifetime_jitter=%s application_name=%s`,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.Schema,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.ConnTimeout,
		cfg.Database.MinConnections,
		cfg.Database.MaxConnections,
		cfg.Database.ConnIdleTime,
		cfg.Database.ConnLifetime,
		cfg.Database.MaxConnLifetimeJitter,
		cfg.App.Name,
	)

	return cfg, nil
}