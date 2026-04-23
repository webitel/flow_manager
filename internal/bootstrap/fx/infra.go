package bsfx

import (
	"context"
	"fmt"

	otelsdk "github.com/webitel/webitel-go-kit/otel/sdk"
	"github.com/webitel/wlog"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/fx"

	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	"github.com/webitel/flow_manager/internal/session"
	postgresStorage "github.com/webitel/flow_manager/internal/storage/postgres"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"github.com/webitel/flow_manager/store/cachelayer"
	sqlstore "github.com/webitel/flow_manager/store/pg_store"
)

// AppID is a distinct named type so fx can inject it unambiguously.
type AppID string

func NewConfig() (*model.Config, error) {
	return bscfg.Load()
}

func NewAppID(cfg *model.Config) AppID {
	return AppID(fmt.Sprintf("%s-%s", model.AppServiceName, cfg.Id))
}

// NewLogger constructs the application logger, configures OpenTelemetry if
// enabled, and registers the otel shutdown via the fx lifecycle.
func NewLogger(lc fx.Lifecycle, cfg *model.Config, id AppID) (*wlog.Logger, error) {
	logConfig := &wlog.LoggerConfiguration{
		EnableConsole: cfg.Log.Console,
		ConsoleJson:   false,
		ConsoleLevel:  cfg.Log.Lvl,
	}
	if cfg.Log.File != "" {
		logConfig.FileLocation = cfg.Log.File
		logConfig.EnableFile = true
		logConfig.FileJson = true
		logConfig.FileLevel = cfg.Log.Lvl
	}
	if cfg.Log.Otel {
		logConfig.EnableExport = true
		shutdownFunc, err := otelsdk.Configure(
			context.Background(),
			otelsdk.WithResource(resource.NewSchemaless(
				semconv.ServiceName(model.AppServiceName),
				semconv.ServiceVersion(model.CurrentVersion),
				semconv.ServiceInstanceID(string(id)),
				semconv.ServiceNamespace("webitel"),
			)),
		)
		if err != nil {
			return nil, err
		}
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				shutdownFunc(ctx)
				return nil
			},
		})
	}
	log := wlog.NewLogger(logConfig)
	wlog.RedirectStdLog(log)
	wlog.InitGlobalLogger(log)
	return log, nil
}

func NewSqlSupplier(cfg *model.Config) *sqlstore.SqlSupplier {
	return sqlstore.NewSqlSupplier(cfg.SqlSettings)
}

func NewStore(s *sqlstore.SqlSupplier) store.Store {
	return store.NewLayeredStore(s)
}

// NewCheckpointRepo creates the postgres session-checkpoint repository and runs
// schema migrations before returning.
func NewCheckpointRepo(s *sqlstore.SqlSupplier) (session.Repository, error) {
	repo := postgresStorage.NewCheckpointRepository(s)
	if err := repo.Migrate(context.Background()); err != nil {
		return nil, fmt.Errorf("session checkpoint migration: %w", err)
	}
	return repo, nil
}

// NewCacheStores always provides in-memory cache; adds Redis when configured.
func NewCacheStores(cfg *model.Config) (cachelayer.CacheStores, error) {
	stores := cachelayer.CacheStores{
		cachelayer.Memory: cachelayer.NewMemoryCache(&cachelayer.MemoryCacheConfig{
			Size:          10000,
			DefaultExpiry: 10000,
		}),
	}
	if cfg.RedisSettings.IsValid() {
		redis, err := cachelayer.NewRedisCache(
			cfg.RedisSettings.Host,
			cfg.RedisSettings.Port,
			cfg.RedisSettings.Password,
			cfg.RedisSettings.Database,
		)
		if err != nil {
			return nil, err
		}
		stores[cachelayer.Redis] = redis
	}
	return stores, nil
}
