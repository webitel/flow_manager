package sqlstore

import (
	"database/sql"
	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"net/http"
)

const ForeignKeyViolationErrorCode = pq.ErrorCode("23503")
const DuplicationViolationErrorCode = pq.ErrorCode("23505")

type PostgresJSONDialect struct {
	gorp.PostgresDialect
}

func extractCodeFromErr(err error) int {
	code := http.StatusInternalServerError

	if err == sql.ErrNoRows {
		code = http.StatusNotFound
	} else if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ForeignKeyViolationErrorCode, DuplicationViolationErrorCode:
			code = http.StatusBadRequest
		}
	}
	return code
}
