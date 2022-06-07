package cachelayer

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

const (
	CacheSize   = 100
	CacheExpire = 25 * 1000
)

type ExternalStoreManager struct {
	cache utils.ObjectCache
}

type ExternalDb struct {
	db *sql.DB
}

func NewExternalStoreManager() *ExternalStoreManager {
	return &ExternalStoreManager{
		cache: utils.NewLruWithParams(CacheSize, "external", CacheExpire, ""),
	}
}

// TODO sync
func (e *ExternalStoreManager) Connect(driver, dns string) (*ExternalDb, *model.AppError) {
	if d, ok := e.cache.Get(dns); ok {
		return d.(*ExternalDb), nil
	}

	db, err := sql.Open(driver, dns)
	if err != nil {
		return nil, model.NewAppError("Cache", "cache.connect.open_err", nil, err.Error(), http.StatusInternalServerError)
	}

	e.cache.AddWithDefaultExpires(dns, &ExternalDb{
		db: db,
	})

	wlog.Info(fmt.Sprintf("store cache db %s", dns))
	return e.Connect(driver, dns)
}

func (d *ExternalDb) Query(ctx context.Context, text string, params []interface{}) (map[string]model.VariableValue, *model.AppError) {
	rows, err := d.db.QueryContext(ctx, text, params...)

	if err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.err", nil, err.Error(), http.StatusInternalServerError)
	}

	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.columns.err", nil, err.Error(), http.StatusInternalServerError)
	}

	cols := make([]interface{}, len(colNames))
	colPtrs := make([]interface{}, len(colNames))
	for i := 0; i < len(colNames); i++ {
		colPtrs[i] = &cols[i]
	}

	rows.Next()
	if err = rows.Scan(colPtrs...); err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.scan.err", nil, err.Error(), http.StatusInternalServerError)
	}

	result := make(map[string]model.VariableValue)

	for i, col := range cols {
		switch col.(type) {
		case []uint8, []uint32, []uint:
			result[colNames[i]] = fmt.Sprintf("%s", col)
		default:
			result[colNames[i]] = fmt.Sprint(col)
		}
	}

	return result, nil
}
