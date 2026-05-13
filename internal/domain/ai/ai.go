package ai

// moved from model/ai.go — see model/ai.go for re-export alias

import "github.com/webitel/flow_manager/internal/domain/chat"

// ChatAiAnswer holds the configuration and result of an AI chat processing step.
type ChatAiAnswer struct {
	Model             string            `json:"model"`
	Categories        string            `json:"categories"`
	Variables         map[string]string `json:"variables"`
	Timeout           int               `json:"timeout"`
	HistoryLength     int               `json:"historyLength"`
	Connection        string            `json:"connection"`
	Messages          []chat.ChatMessage `json:"-"`
	Response          string            `json:"response"`
	DefinedCategories string            `json:"definedCategories"`
}
