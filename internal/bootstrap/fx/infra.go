package bsfx

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/fx"

	otelsdk "github.com/webitel/webitel-go-kit/otel/sdk"
	"github.com/webitel/wlog"

	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	"github.com/webitel/flow_manager/internal/infrastructure/cache"
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	pgsqlImpl "github.com/webitel/flow_manager/internal/infrastructure/sql/pgsql"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/session"
	postgresStorage "github.com/webitel/flow_manager/internal/storage/postgres"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
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

func NewStore(db infraSql.Store) store.Store {
	return postgresStorage.NewStore(db)
}

// NewPgxPool creates a pgxpool connection pool from cfg and registers pool.Close in the fx lifecycle.
func NewPgxPool(lc fx.Lifecycle, cfg *model.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), *cfg.SqlSettings.DataSource)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			pool.Close()
			return nil
		},
	})
	return pool, nil
}

// NewSqlStore wraps the pgxpool as the driver-agnostic sql.Store interface.
func NewSqlStore(pool *pgxpool.Pool, log *wlog.Logger) infraSql.Store {
	return pgsqlImpl.NewFromPool(context.Background(), pool, log)
}

// NewCheckpointRepo creates the session-checkpoint repository backed by pgx.
// Goose migrations must run before this is called (see RunMigrations invoke in main).
func NewCheckpointRepo(db infraSql.Store) session.Repository {
	return postgresStorage.NewCheckpointRepository(db)
}

// NewRuntimeStateRepo creates the resumable-flow execution-state repository.
// Goose migrations must run before this is called (migration 20260426000001).
func NewRuntimeStateRepo(db infraSql.Store) persistence.Repository {
	return postgresStorage.NewRuntimeStateRepository(db)
}

// NewCacheStores always provides in-memory cache; adds Redis when configured.
func NewCacheStores(cfg *model.Config) (cache.CacheStores, error) {
	stores := cache.CacheStores{
		cache.Memory: cache.NewMemoryCache(&cache.MemoryCacheConfig{
			Size:          10000,
			DefaultExpiry: 10000,
		}),
	}
	if cfg.RedisSettings.IsValid() {
		redis, err := cache.NewRedisCache(
			cfg.RedisSettings.Host,
			cfg.RedisSettings.Port,
			cfg.RedisSettings.Password,
			cfg.RedisSettings.Database,
		)
		if err != nil {
			return nil, err
		}
		stores[cache.Redis] = redis
	}
	return stores, nil
}
