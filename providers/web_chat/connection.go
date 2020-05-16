package web_chat

import "github.com/webitel/flow_manager/model"

type connection struct {
	Id         int64           `json:"id" db:"id"`
	Title      string          `json:"title" db:"title"`
	CreatedAt  int64           `json:"created_at" db:"created_at"`
	ActivityAt int64           `json:"activity_at" db:"activity_at"`
	ClosedAt   int64           `json:"closed_at" db:"closed_at"`
	Variables  model.Variables `json:"variables" db:"variables"`
}
