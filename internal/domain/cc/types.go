package cc

type QueueEvent struct {
	AttemptId int64
	Event     string
	Result    string
}
