package model

import "time"

type Timezone struct {
	Id   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// GetMillis is a convience method to get milliseconds since epoch.
func GetMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
