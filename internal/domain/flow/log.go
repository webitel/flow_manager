package flow

// moved from model/log.go — see model/log.go for re-export alias

// StepLog records timing for a single flow step.
type StepLog struct {
	Name  string `json:"name"`
	Start int64  `json:"start"`
	Stop  int64  `json:"stop"`
}
