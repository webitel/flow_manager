package flow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/gen/kb"
	"github.com/webitel/flow_manager/model"
)

// kbSearchTimeout bounds one knowledge base search; the knowledge base skips
// the rerank rather than miss it.
const kbSearchTimeout = 5 * time.Second

// kbSearch is the "KB Search" node: it finds the fragments of the knowledge
// base spaces that answer the query and writes them, with the execution state,
// to the schema variables. A failed search is a state, not an error, so the
// schema can branch on it.
func (r *router) kbSearch(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv model.KBSearchArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	req, appErr := kbSearchRequest(argv)
	if appErr != nil {
		if argv.SetExecVar != "" {
			if _, err := conn.Set(ctx, model.Variables{argv.SetExecVar: model.KBSearchProviderError}); err != nil {
				conn.Log().Error("kb search: set the execution state", wlog.Err(err))
			}
		}

		return nil, appErr
	}

	if strings.TrimSpace(req.GetQuery()) == "" {
		return conn.Set(ctx, kbSearchVariables(argv, nil, model.KBSearchNoMatch))
	}

	searchCtx, cancel := context.WithTimeout(ctx, kbSearchTimeout)
	defer cancel()

	res, err := r.fm.KBSemanticSearch(searchCtx, argv.Token, req)
	if err != nil {
		conn.Log().Error("kb search failed", wlog.Err(err))
	}

	fragments := kbFragments(res)

	return conn.Set(ctx, kbSearchVariables(argv, fragments, kbExecState(err, len(fragments))))
}

// kbSearchRequest builds the search from the node arguments.
func kbSearchRequest(argv model.KBSearchArgs) (*kb.SemanticSearchRequest, *model.AppError) {
	if argv.Token == "" {
		return nil, model.ErrorRequiredParameter(model.KBSearchApp, "token")
	}

	if argv.SetVar == "" {
		return nil, model.ErrorRequiredParameter(model.KBSearchApp, "setVar")
	}

	spaceIDs := make([]int64, 0, len(argv.Spaces))
	for _, space := range argv.Spaces {
		if space.Id != nil && *space.Id > 0 {
			spaceIDs = append(spaceIDs, int64(*space.Id))
		}
	}

	if len(spaceIDs) == 0 {
		return nil, model.ErrorRequiredParameter(model.KBSearchApp, "spaces")
	}

	filters, err := kbFilters(argv.Filters)
	if err != nil {
		return nil, model.NewAppError("Valid", "valid.app."+model.KBSearchApp, nil, fmt.Sprintf("App=%s %s", model.KBSearchApp, err), http.StatusBadRequest)
	}

	topK := argv.TopK
	if topK <= 0 {
		topK = model.KBSearchDefaultTopK
	}

	return &kb.SemanticSearchRequest{
		Query:    argv.Query,
		SpaceIds: spaceIDs,
		TopK:     int32(topK),
		Tags:     filters.Tags,
		TagMatch: kbTagMatch[filters.TagMatch],
		Rerank:   argv.Rerank,
	}, nil
}

// kbTagMatch maps the node tag mode to the search one; an empty mode is any.
var kbTagMatch = map[string]kb.TagMatch{
	"":    kb.TagMatch_TAG_MATCH_UNSPECIFIED,
	"any": kb.TagMatch_TAG_MATCH_ANY,
	"all": kb.TagMatch_TAG_MATCH_ALL,
}

// kbFilters reads the node filters; none when they are empty.
func kbFilters(raw *model.JsonValue) (model.KBSearchFilters, error) {
	var filters model.KBSearchFilters

	if raw == nil || strings.TrimSpace(string(*raw)) == "" {
		return filters, nil
	}

	if err := json.Unmarshal(*raw, &filters); err != nil {
		return filters, fmt.Errorf("filters must be a JSON object: %w", err)
	}

	filters.TagMatch = strings.ToLower(strings.TrimSpace(filters.TagMatch))
	if _, ok := kbTagMatch[filters.TagMatch]; !ok {
		return filters, errors.New("filters.tagMatch must be any or all")
	}

	return filters, nil
}

// kbFragments turns the found chunks into the node result.
func kbFragments(res *kb.SemanticSearchResponse) []model.KBFragment {
	chunks := res.GetChunks()
	fragments := make([]model.KBFragment, 0, len(chunks))

	for _, chunk := range chunks {
		fragment := model.KBFragment{
			ArticleID: chunk.GetArticleId(),
			VersionID: chunk.GetVersionId(),
			ChunkID:   chunk.GetId(),
			Title:     chunk.GetSubject(),
			Snippet:   chunk.GetContent(),
		}

		if chunk.Score != nil {
			score := chunk.GetScore()
			fragment.Score = &score
		}

		fragments = append(fragments, fragment)
	}

	return fragments
}

// kbExecState is the execution state of a search that found n fragments or
// failed with err.
func kbExecState(err error, n int) string {
	if err == nil {
		if n == 0 {
			return model.KBSearchNoMatch
		}

		return model.KBSearchSuccess
	}

	if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
		return model.KBSearchTimeout
	}

	return model.KBSearchProviderError
}

// kbSearchVariables are the schema variables the node writes: the fragments,
// an empty array unless the search succeeded, and the state when asked.
func kbSearchVariables(argv model.KBSearchArgs, fragments []model.KBFragment, state string) model.Variables {
	if state != model.KBSearchSuccess || fragments == nil {
		fragments = make([]model.KBFragment, 0)
	}

	vars := model.Variables{argv.SetVar: model.ToJson(fragments)}
	if argv.SetExecVar != "" {
		vars[argv.SetExecVar] = state
	}

	return vars
}
