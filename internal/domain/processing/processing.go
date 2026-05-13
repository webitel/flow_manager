package processing

// moved from model/processing.go — see model/processing.go for re-export aliases

// ProcessingForm describes the type or form of post-processing.
type ProcessingForm struct {
	Id   int32  `json:"id"`
	Name string `json:"name"`
}

// ProcessingProlongation sets the rules for prolongation of post-processing.
type ProcessingProlongation struct {
	Enabled             bool   `json:"enabled"`
	RepeatsNumber       uint32 `json:"repeats_number"`
	ProlongationTimeSec uint32 `json:"prolongation_time_sec"`
	IsTimeoutRetry      bool   `json:"is_timeout_retry"`
}

// Processing describes the post-processing configuration for flow elements.
type Processing struct {
	Enabled      bool                    `json:"enabled"`
	RenewalSec   uint32                  `json:"renewal_sec"`
	Sec          uint32                  `json:"sec"`
	Form         ProcessingForm          `json:"form"`
	Prolongation *ProcessingProlongation `json:"processing_prolongation"`
}

// ProcessingWithoutAnswer extends Processing with answer-specific rules.
type ProcessingWithoutAnswer struct {
	Enabled       bool                    `json:"enabled"`
	RenewalSec    uint32                  `json:"renewalSec"`
	WithoutAnswer bool                    `json:"withoutAnswer"`
	Sec           uint32                  `json:"sec"`
	Form          ProcessingForm          `json:"form"`
	Prolongation  *ProcessingProlongation `json:"processing_prolongation"`
}
