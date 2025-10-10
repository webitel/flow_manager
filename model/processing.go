package model

// ProcessingWithoutAnswer extends the base Processing configuration with answer-specific rules.
type ProcessingWithoutAnswer struct {
	Enabled       bool                    `json:"enabled"`
	RenewalSec    uint32                  `json:"renewalSec"`
	WithoutAnswer bool                    `json:"withoutAnswer"`
	Sec           uint32                  `json:"sec"`
	Form          ProcessingForm          `json:"form"`
	Prolongation  *ProcessingProlongation `json:"processing_prolongation"`
}

// Processing describes the post-processing configuration for flow elements.
// Used to control how processing occurs after the main action is performed.
type Processing struct {
	Enabled      bool                    `json:"enabled"`
	RenewalSec   uint32                  `json:"renewal_sec"`
	Sec          uint32                  `json:"sec"`
	Form         ProcessingForm          `json:"form"`
	Prolongation *ProcessingProlongation `json:"processing_prolongation"`
}

// ProcessingForm describes the type or form of post-processing.
// Used to identify specific logic or patterns.
type ProcessingForm struct {
	Id   int32  `json:"id"`
	Name string `json:"name"`
}

// ProcessingProlongation sets the rules for prolongation (extension)
// or repeated execution of post-processing.
type ProcessingProlongation struct {
	Enabled             bool   `json:"enabled"`
	RepeatsNumber       uint32 `json:"repeats_number"`
	ProlongationTimeSec uint32 `json:"prolongation_time_sec"`
	IsTimeoutRetry      bool   `json:"is_timeout_retry"`
}
