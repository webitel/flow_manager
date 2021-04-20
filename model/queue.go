package model

type SearchEntity struct {
	Id   *int
	Name *string
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
