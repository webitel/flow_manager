package storage

const (
	ChannelCall = "call"
	ChannelChat = "chat"
	ChannelMail = "mail"
)

type File struct {
	Id        int
	Url       string
	PublicUrl string
	Name      string
	Size      int64
	MimeType  string
	Channel   string
}

type FileLinkRequest struct {
	FileId int64
	Action string
	Source string
}
