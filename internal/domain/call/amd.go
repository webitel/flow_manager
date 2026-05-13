package call

// moved from model/amd.go — see model/amd.go for re-export aliases

// AmdParameters configures voice-activity / answering-machine detection.
type AmdParameters struct {
	SilenceThreshold     *int `json:"silenceThreshold"`
	MaximumWordLength    *int `json:"maximumWordLength"`
	MaximumNumberOfWords *int `json:"maximumNumberOfWords"`
	BetweenWordsSilence  *int `json:"betweenWordsSilence"`
	MinWordLength        *int `json:"minWordLength"`
	TotalAnalysisTime    *int `json:"totalAnalysisTime"`
	AfterGreetingSilence *int `json:"afterGreetingSilence"`
	Greeting             *int `json:"greeting"`
	InitialSilence       *int `json:"initialSilence"`
}

// AmdMLParameters configures machine-learning-based AMD.
type AmdMLParameters struct {
	Tags []string
}
