package model

type Member struct {
	Id   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type CallbackCommunication struct {
	Destination string       `json:"destination"`
	Description *string      `json:"description"`
	TypeId      *int         `json:"type_id"`
	Type        SearchEntity `json:"type"`
	ResourceId  *int         `json:"resource_id"`
	Display     *string      `json:"display"`
}

type CallbackMember struct {
	Name      string     `json:"name"`
	Variables *Variables `json:"variables"`
	Timezone  struct {
		Id *int `json:"id"`
	} `json:"timezone"`
	Bucket struct {
		Id *int `json:"id"`
	}
	Priority      int                   `json:"priority"`
	Communication CallbackCommunication `json:"communication"`
	Queue         SearchEntity          `json:"queue"`
	Agent         struct {
		Id *int `json:"id"`
	} `json:"agent"`
	ExpireAt *int64 `json:"expire_at"`
}

type SearchMember struct {
	QueueIds    []int          `json:"queue_ids"` // todo deprecated
	Queues      []SearchEntity `json:"queues"`
	Destination *string        `json:"destination"`
	Name        *string        `json:"name"`
	Today       *bool          `json:"today"`
	Completed   *bool          `json:"completed"`
	BucketId    *int           `json:"bucket_id"` // todo deprecated
	Bucket      *SearchEntity  `json:"bucket"`
	Id          *int64         `json:"id"`
}

type PatchMember struct {
	Name      *string       `json:"name"`
	Priority  *int          `json:"priority"`
	BucketId  *int          `json:"bucket_id"` // todo deprecated
	Bucket    *SearchEntity `json:"bucket"`
	ReadyAt   *int64        `json:"ready_at"`
	StopCause *string       `json:"stop_cause"`
	Variables *Variables    `json:"variables"`
}

func (m *SearchMember) GetQueueIds() []int {
	if m.QueueIds != nil && m.Queues == nil {
		return m.QueueIds
	}

	return getIds(m.Queues)
}

func getIds(src []SearchEntity) []int {
	l := len(src)

	res := make([]int, 0, l)
	for _, q := range src {
		if q.Id != nil {
			res = append(res, *q.Id)
		}
	}

	return res
}
