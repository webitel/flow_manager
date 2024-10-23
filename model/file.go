package model

type SearchFile struct {
	Id   int
	Name string
}

const (
	FileChannelCall = "call"
	FileChannelChat = "chat"
	FileChannelMail = "mail"
)

type File struct {
	Id        int     `json:"id" db:"id"`
	Url       string  `json:"url,omitempty" db:"-"`
	PublicUrl string  `json:"public_url,omitempty" db:"-"`
	Name      string  `json:"name" db:"name"`
	Size      int64   `json:"size" db:"size"`
	MimeType  string  `json:"mime" db:"mime_type"`
	ViewName  *string `json:"view_name" db:"view_name"`
	Server    string  `json:"server" db:"-"`
	Channel   string  `json:"channel" db:"-"`
}

func (f *File) GetViewName() string {
	if f.ViewName != nil {
		return *f.ViewName
	}

	return f.Name
}
