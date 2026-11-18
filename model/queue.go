package model

type SearchEntity struct {
	Id   *int    `json:"id"`
	Name *string `json:"name"`
}

// IsEmpty reports whether neither a positive id nor a non-empty name is set.
// Decoded schema arguments like {"id": ""} produce Id = &0, which is treated as empty.
func (s *SearchEntity) IsEmpty() bool {
	return s == nil || ((s.Id == nil || *s.Id <= 0) && (s.Name == nil || *s.Name == ""))
}

type InQueueKey struct {
	AttemptId int64
	AppId     string
}

type QueueData struct {
	Type     uint `json:"type" db:"type"`
	Enabled  bool `json:"enabled" db:"enabled"`
	Priority int  `json:"priority" db:"priority"`
}

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

type SearchQueueActiveStatistics struct {
	QueueId    *int
	QueueName  *string
	BucketId   *int
	BucketName *string
	Metric     string
	Field      string
	State      string
}

func (s *SearchEntity) GetId() *int {
	if s == nil {
		return nil
	}

	return s.Id
}
