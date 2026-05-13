package files

// moved from model/file.go — see model/file.go for re-export aliases

const (
	FileChannelCall = "call"
	FileChannelChat = "chat"
	FileChannelMail = "mail"
)

// SearchFile is a minimal file reference used in search operations.
type SearchFile struct {
	Id   int
	Name string
}

// File describes a stored file attachment.
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
