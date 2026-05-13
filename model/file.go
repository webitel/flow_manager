package model

import "github.com/webitel/flow_manager/internal/domain/files"

// Re-exports for backward compatibility.
type SearchFile = files.SearchFile
type File = files.File

const (
	FileChannelCall = files.FileChannelCall
	FileChannelChat = files.FileChannelChat
	FileChannelMail = files.FileChannelMail
)
