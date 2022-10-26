package model

type SearchFile struct {
	Id   int
	Name string
}

type File struct {
	Id        int    `json:"id" db:"id"`
	Url       string `json:"url,omitempty" db:"-"`
	PublicUrl string `json:"public_url,omitempty" db:"-"`
	Name      string `json:"name" db:"name"`
	Size      int64  `json:"size" db:"size"`
	MimeType  string `json:"mime" db:"mime_type"`
}
