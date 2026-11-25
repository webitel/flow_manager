package model

// KBSearchApp is the schema application id of the "KB Search" node.
const KBSearchApp = "kbSearch"

// Execution states the node writes to setExecVar.
const (
	KBSearchSuccess       = "success"
	KBSearchNoMatch       = "no_match"
	KBSearchProviderError = "provider_error"
	KBSearchTimeout       = "timeout"
)

// KBSearchDefaultTopK is how many fragments the node asks for by default.
const KBSearchDefaultTopK = 5

// KBSearchArgs are the node's arguments as stored in the schema.
//
//nolint:tagliatelle // schema argument names, shared with the flow editor
type KBSearchArgs struct {
	// Token is the access token the search runs under.
	Token string `json:"token"`
	// Spaces are the knowledge base spaces to search.
	Spaces []SearchEntity `json:"spaces"`
	// Query is the search text, usually the client's message.
	Query string `json:"query"`
	// TopK is the maximum number of fragments.
	TopK int `json:"topK"`
	// SetVar receives the fragments as a JSON array.
	SetVar string `json:"setVar"`
	// SetExecVar receives the execution state; optional.
	SetExecVar string `json:"setExecVar"`
	// Rerank reorders the fragments with the reranker of the spaces.
	Rerank bool `json:"rerank"`
	// Filters is a JSON object of KBSearchFilters; optional.
	Filters *JsonValue `json:"filters"`
}

// KBSearchFilters narrow the search by article tags.
//
//nolint:tagliatelle // schema argument names, shared with the flow editor
type KBSearchFilters struct {
	Tags []string `json:"tags"`
	// TagMatch is "any" (default) or "all".
	TagMatch string `json:"tagMatch"`
}

// KBFragment is one fragment of the node result.
type KBFragment struct {
	ArticleID int64  `json:"article_id,string"`
	VersionID int64  `json:"version_id,string"`
	ChunkID   int64  `json:"chunk_id,string"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	// Score is the relevance in [0, 1]; absent in a space without vector search.
	Score *float64 `json:"score,omitempty"`
}
