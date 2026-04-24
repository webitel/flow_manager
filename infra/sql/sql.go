package sql

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Rows interface {
	// Close closes the rows, making the connection ready for use again. It is safe
	// to call Close after rows is already closed.
	Close()

	Columns() []string

	// Err returns any error that occurred while reading. Err must only be called after the Rows is closed (either by
	// calling Close or by Next returning false). If it is called early it may return nil even if there was an error
	// executing the query.
	Err() error

	// Next prepares the next row for reading. It returns true if there is another
	// row and false if no more rows are available or a fatal error has occurred.
	// It automatically closes rows when all rows are read.
	//
	// Callers should check rows.Err() after rows.Next() returns false to detect
	// whether result-set reading ended prematurely due to an error. See
	// Conn.Query for details.
	//
	// For simpler error handling, consider using the higher-level pgx v5
	// CollectRows() and ForEachRow() helpers instead.
	Next() bool

	// Scan reads the values from the current row into dest values positionally.
	// dest can include pointers to core types, values implementing the Scanner
	// interface, and nil. nil will skip the value entirely. It is an error to
	// call Scan without first calling Next() and checking that it returned true.
	Scan(dest ...any) error

	// Values returns the decoded row values. As with Scan(), it is an error to
	// call Values without first calling Next() and checking that it returned
	// true.
	Values() ([]any, error)
}

type Store interface {
	Select(ctx context.Context, out any, query string, args pgx.NamedArgs) error
	Query(ctx context.Context, sql string, args pgx.NamedArgs) (Rows, error)
	Get(ctx context.Context, out any, query string, args pgx.NamedArgs) error
	Exec(ctx context.Context, sql string, args pgx.NamedArgs) error
	Close() error
	Begin(ctx context.Context) (pgx.Tx, error)
}
