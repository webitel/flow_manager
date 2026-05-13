package files

// moved from model/storage.go — see model/storage.go for re-export alias

// FileLinkRequest is used to request a signed link to a stored file.
type FileLinkRequest struct {
	FileId int64
	Action string
	Source string
}
