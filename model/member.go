package model

type Member struct {
	Id   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type SearchMember struct {
	QueueId     uint32  `json:"queue_id"`
	Destination *string `json:"destination"`
	Name        *string `json:"name"`
	Today       *bool   `json:"today"`
	Completed   *bool   `json:"completed"`
	BucketId    *uint32 `json:"bucket_id"`
}
