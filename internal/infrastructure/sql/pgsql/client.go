package pgsql

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/infrastructure/sql"
)

type DB struct {
	ctx  context.Context
	pool *pgxpool.Pool
	log  *wlog.Logger
}

type rows struct {
	pgx.Rows
}

func New(ctx context.Context, dsn string, log *wlog.Logger) (sql.Store, error) {
	dbCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		return nil, err
	}

	return NewFromPool(ctx, pool, log), nil
}

// NewFromPool wraps an existing pgxpool as a sql.Store without creating a new connection.
func NewFromPool(ctx context.Context, pool *pgxpool.Pool, log *wlog.Logger) sql.Store {
	return &DB{ctx: ctx, pool: pool, log: log}
}

func (db *DB) Select(ctx context.Context, out any, query string, args pgx.NamedArgs) error {
	return pgxscan.Select(ctx, db.pool, out, query, args)
}

func (db *DB) SelectArgs(ctx context.Context, out any, query string, args ...any) error {
	return pgxscan.Select(ctx, db.pool, out, query, args...)
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *DB) Get(ctx context.Context, out any, query string, args pgx.NamedArgs) error {
	return pgxscan.Get(ctx, db.pool, out, query, args)
}

func (db *DB) Query(ctx context.Context, sql string, args pgx.NamedArgs) (sql.Rows, error) {
	r, err := db.pool.Query(ctx, sql, args)
	if err != nil {
		return nil, err
	}

	return &rows{
		Rows: r,
	}, nil
}

func (db *DB) Exec(ctx context.Context, sql string, args pgx.NamedArgs) error {
	_, err := db.pool.Exec(ctx, sql, args)
	return err
}

func (db *DB) ExecResult(ctx context.Context, sql string, args pgx.NamedArgs) (int64, error) {
	tag, err := db.pool.Exec(ctx, sql, args)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (db *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

func (db *DB) Close() error {
	db.pool.Close()

	return nil
}

func (r *rows) Columns() []string {
	c := make([]string, 0, len(r.FieldDescriptions()))
	for _, v := range r.FieldDescriptions() {
		c = append(c, v.Name)
	}

	return c
}
