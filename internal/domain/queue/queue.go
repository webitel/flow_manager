package queue

// moved from model/queue.go — see model/queue.go for re-export aliases

// SearchEntity is a generic reference to a named entity (by id or name).
type SearchEntity struct {
	Id   *int    `json:"id"`
	Name *string `json:"name"`
}

func (s *SearchEntity) GetId() *int {
	if s == nil {
		return nil
	}
	return s.Id
}

// InQueueKey uniquely identifies a pending queue slot.
type InQueueKey struct {
	AttemptId int64
	AppId     string
}

// QueueData holds basic queue configuration used in routing decisions.
type QueueData struct {
	Type     uint `json:"type" db:"type"`
	Enabled  bool `json:"enabled" db:"enabled"`
	Priority int  `json:"priority" db:"priority"`
}

// SearchQueueCompleteStatistics is a filter for queue completion statistics.
type SearchQueueCompleteStatistics struct {
	QueueId     *int
	QueueName   *string
	BucketId    *int
	BucketName  *string
	LastMinutes int
	Metric      string
	Field       string
	SlSec       int
}

// SearchQueueActiveStatistics is a filter for live queue statistics.
type SearchQueueActiveStatistics struct {
	QueueId    *int
	QueueName  *string
	BucketId   *int
	BucketName *string
	Metric     string
	Field      string
	State      string
}
