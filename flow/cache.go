package flow

import (
	"context"
	"net/http"
	"strconv"

	"github.com/webitel/flow_manager/model"
)

type CacheArgs struct {
	Type   string            `json:"type,omitempty"`
	Action string            `json:"action,omitempty"`
	Get    map[string]string `json:"get,omitempty"`
	Set    *Set              `json:"set,omitempty"`
	Delete *Delete           `json:"delete,omitempty"`
}

type Set struct {
	Data map[string]string `json:"data,omitempty"`
	Ttl  string            `json:"ttl,omitempty"`
}

type Delete struct {
	Keys []string `json:"keys,omitempty"`
}

func (r *router) Cache(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var (
		cacheArgs *CacheArgs
	)

	if err := scope.Decode(args, &cacheArgs); err != nil {
		return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.not_found", nil, "bad arguments", http.StatusBadRequest)
	}
	switch cacheArgs.Action {
	case "set":
		if cacheArgs.Set == nil {
			return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.set", nil, "no 'set' object when type is 'set'", http.StatusInternalServerError)
		}
		ttl, err := strconv.ParseInt(cacheArgs.Set.Ttl, 10, 0)
		if err != nil {
			return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.set", nil, err.Error(), http.StatusBadRequest)
		}
		for key, value := range cacheArgs.Set.Data {
			err := r.fm.CacheSetValue(ctx, cacheArgs.Type, conn.DomainId(), key, value, ttl)
			if err != nil {
				return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.set", nil, err.Error(), http.StatusBadRequest)
			}
		}
	case "get":
		variables := make(model.Variables)
		if cacheArgs.Get == nil {
			return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.get", nil, "no 'get' object when type is 'get'", http.StatusInternalServerError)
		}
		for variable, key := range cacheArgs.Get {
			value, err := r.fm.CacheGetValue(ctx, cacheArgs.Type, conn.DomainId(), key)
			if err != nil {
				return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.get", nil, err.Error(), http.StatusBadRequest)
			}
			val, err := value.String()
			if err != nil {
				return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.get", nil, err.Error(), http.StatusBadRequest)
			}
			variables[variable] = val

		}
		_, err := conn.Set(ctx, variables)
		if err != nil {
			return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.get", nil, err.Error(), http.StatusInternalServerError)
		}

	case "delete":
		if cacheArgs.Delete == nil {
			return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.delete", nil, "no 'delete' object when type is 'delete'", http.StatusInternalServerError)
		}
		for _, key := range cacheArgs.Delete.Keys {
			err := r.fm.CacheDeleteValue(ctx, cacheArgs.Type, conn.DomainId(), key)
			if err != nil {
				return ResponseErr, model.NewAppError("Flow.Cache", "flow.cache.delete", nil, err.Error(), http.StatusInternalServerError)
			}
		}

	}

	return ResponseOK, nil

}
