package chat

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

const contactSearchVariable = "wbt_contact_search"

type ContactQuery struct {
	Qin []string `json:"qin"`
	Q   string   `json:"q"`
}

type SearchContactArg struct {
	Query []ContactQuery `json:"query"`
}

func (r *Router) searchContact(ctx context.Context, scope *flow.Flow, conv Conversation, args any) (model.Response, *model.AppError) {
	var argv SearchContactArg
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if len(argv.Query) == 0 {
		return nil, flow.ErrorRequiredParameter("searchContact", "query")
	}

	query := make([]ContactQuery, 0, len(argv.Query))

	for _, q := range argv.Query {
		if q.Q == "" || len(q.Qin) == 0 {
			continue
		}

		query = append(query, q)
	}

	spec, err := json.Marshal(query)
	if err != nil {
		return nil, model.NewAppError("searchContact", "chat.search_contact.marshal", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, aErr := conv.Set(ctx, model.Variables{contactSearchVariable: string(spec)}); aErr != nil {
		return nil, aErr
	}

	return conv.Export(ctx, []string{contactSearchVariable})
}
