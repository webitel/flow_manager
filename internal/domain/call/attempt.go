package call

// moved from model/attempt.go — see model/attempt.go for re-export alias
// NOTE: AttemptResult.AddCommunications references CallbackCommunication which
//       lives in internal/domain/queue — see that package for the type definition.

import "github.com/webitel/flow_manager/internal/domain/queue"

// AttemptResult describes the outcome of a call-centre attempt.
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
	AddCommunications           []queue.CallbackCommunication `json:"addCommunications"`
	WaitBetweenRetries          *int32                        `json:"waitBetweenRetries"`
}
