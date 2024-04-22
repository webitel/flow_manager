package model

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

type AmdMLParameters struct {
	Tags []string
}
