package model

type ChatAiAnswer struct {
	Model             string            `json:"model"`
	Categories        []string          `json:"categories"`
	Variables         map[string]string `json:"variables"`
	Timeout           int               `json:"timeout"`
	HistoryLength     int               `json:"historyLength"`
	Connection        string            `json:"connection"`
	Messages          []ChatMessage     `json:"-"`
	Response          string            `json:"response"`
	DefinedCategories string            `json:"definedCategories"`
}
