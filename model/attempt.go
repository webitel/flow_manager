package model

type AttemptResult struct {
	Id                          int64
	Status                      string
	Description                 string
	ReadyAt                     *int64 `json:"readyAt"`
	ExpiredAt                   *int64
	Variables                   map[string]string
	StickyDisplay               bool
	AgentId                     int32
	Redial                      bool
	ExcludeCurrentCommunication bool
	AddCommunications           []CallbackCommunication `json:"addCommunications"`
	WaitBetweenRetries          *int32                  `json:"waitBetweenRetries"`
}
