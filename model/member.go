package model

type Member struct {
	Id   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type SearchMember struct {
	QueueIds    []int   `json:"queue_ids"`
	Destination *string `json:"destination"`
	Name        *string `json:"name"`
	Today       *bool   `json:"today"`
	Completed   *bool   `json:"completed"`
	BucketId    *int    `json:"bucket_id"`
}
