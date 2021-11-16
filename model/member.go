package model

type Member struct {
	Id   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type CallbackCommunication struct {
	Destination string  `json:"destination"`
	Description *string `json:"description"`
	TypeId      int     `json:"type_id"`
}

type CallbackMember struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
	Timezone  struct {
		Id *int `json:"id"`
	} `json:"timezone"`
	Bucket struct {
		Id *int `json:"id"`
	}
	Priority      int                   `json:"priority"`
	Communication CallbackCommunication `json:"communication"`
}

type SearchMember struct {
	QueueIds    []int   `json:"queue_ids"`
	Destination *string `json:"destination"`
	Name        *string `json:"name"`
	Today       *bool   `json:"today"`
	Completed   *bool   `json:"completed"`
	BucketId    *int    `json:"bucket_id"`
}

type PatchMember struct {
	Name      *string    `json:"name"`
	Priority  *int       `json:"priority"`
	BucketId  *int       `json:"bucket_id"`
	ReadyAt   *int64     `json:"ready_at"`
	StopCause *string    `json:"stop_cause"`
	Variables *Variables `json:"variables"`
}
