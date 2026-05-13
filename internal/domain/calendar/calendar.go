package calendar

// moved from model/calendar.go — see model/calendar.go for re-export alias

// Calendar describes the operating-hours schedule and its current status.
type Calendar struct {
	Name     string  `json:"name" db:"name"`
	Excepted *string `json:"excepted" db:"excepted"`
	Accept   bool    `json:"accept" db:"accept"`
	Expire   bool    `json:"expire"`
}
