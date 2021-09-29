package model

type StepLog struct {
	Name  string `json:"name"`
	Start int64  `json:"start"`
	Stop  int64  `json:"stop"`
}
