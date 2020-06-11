package model

type SearchEntity struct {
	Id   *int
	Name *string
}

type SearchQueueCompleteStatistics struct {
	QueueId     *int
	QueueName   *string
	BucketId    *int
	BucketName  *string
	LastMinutes int
	Metric      string
	Field       string
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
