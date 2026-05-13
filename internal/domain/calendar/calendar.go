package calendar

// moved from model/calendar.go — see model/calendar.go for re-export alias

// Timezone maps an integer timezone ID to an IANA location name.
type Timezone struct {
	Id      int    `json:"id" db:"id"`
	SysName string `json:"sys_name" db:"sys_name"`
}

// Calendar describes the operating-hours schedule and its current status.
type Calendar struct {
	Name     string  `json:"name" db:"name"`
	Excepted *string `json:"excepted" db:"excepted"`
	Accept   bool    `json:"accept" db:"accept"`
	Expire   bool    `json:"expire"`
}
