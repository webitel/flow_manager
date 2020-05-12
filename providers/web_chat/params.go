package web_chat

import (
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

const (
	PAGE_DEFAULT     = 0
	PER_PAGE_DEFAULT = 20
	PER_PAGE_MAXIMUM = 100
)

type Params struct {
	Id      string
	Page    int
	PerPage int
}

func ParamsFromRequest(r *http.Request) Params {
	params := Params{}

	props := mux.Vars(r)
	query := r.URL.Query()

	if val, ok := props["id"]; ok {
		params.Id = val
	}

	if val, err := strconv.Atoi(query.Get("page")); err != nil || val < 0 {
		params.Page = PAGE_DEFAULT
	} else {
		params.Page = val
	}

	if val, err := strconv.Atoi(query.Get("per_page")); err != nil || val < 0 {
		params.PerPage = PER_PAGE_DEFAULT
	} else if val > PER_PAGE_MAXIMUM {
		params.PerPage = PER_PAGE_MAXIMUM
	} else {
		params.PerPage = val
	}

	return params
}
