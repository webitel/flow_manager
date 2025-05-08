package app

import (
	"context"
	"github.com/webitel/engine/pkg/wbt"
	"github.com/webitel/flow_manager/gen/storage"
	"github.com/webitel/flow_manager/model"
	"io"
)

type storageClient struct {
	file       *wbt.Client[storage.FileServiceClient]
	transcript *wbt.Client[storage.FileTranscriptServiceClient]
}

func NewStorageClient(consulTarget string) (*storageClient, error) {
	file, err := wbt.NewClient(consulTarget, wbt.StorageServiceName, storage.NewFileServiceClient)
	if err != nil {
		return nil, err
	}
	transcript, err := wbt.NewClient(consulTarget, wbt.StorageServiceName, storage.NewFileTranscriptServiceClient)
	if err != nil {
		return nil, err
	}

	return &storageClient{
		file:       file,
		transcript: transcript,
	}, nil
}

func (api *storageClient) Upload(ctx context.Context, domainId int64, uuid string, sFile io.Reader, metadata model.File) (model.File, error) {
	stream, err := api.file.Api.UploadFile(ctx)
	if err != nil {
		return model.File{}, err
	}

	err = stream.Send(&storage.UploadFileRequest{
		Data: &storage.UploadFileRequest_Metadata_{
			Metadata: &storage.UploadFileRequest_Metadata{
				DomainId: domainId,
				Name:     metadata.Name,
				MimeType: metadata.MimeType,
				Uuid:     uuid,
				Channel:  fileChannel(metadata.Channel),
			},
		},
	})

	if err != nil {
		return model.File{}, err
	}

	defer stream.CloseSend()

	buf := make([]byte, 4*1024)
	var n int
	for {
		n, err = sFile.Read(buf)
		buf = buf[:n]
		if err != nil {
			break
		}
		err = stream.Send(&storage.UploadFileRequest{
			Data: &storage.UploadFileRequest_Chunk{
				Chunk: buf,
			},
		})
		if err != nil {
			break
		}
	}

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		return model.File{}, err
	}

	var res *storage.UploadFileResponse
	res, err = stream.CloseAndRecv()
	if err != nil {
		return model.File{}, err
	}

	metadata.Id = int(res.FileId)
	metadata.Size = res.Size
	metadata.Url = res.FileUrl
	metadata.PublicUrl = res.Server + res.FileUrl

	return metadata, nil
}

func (api *storageClient) Download(ctx context.Context, domainId int64, id int64) (io.ReadCloser, error) {
	stream, err := api.file.Api.DownloadFile(ctx, &storage.DownloadFileRequest{
		Id:       id,
		DomainId: domainId,
		Metadata: false,
	})

	if err != nil {
		return nil, err
	}

	reader, writer := io.Pipe()

	go func(w io.WriteCloser) {
		var sf *storage.StreamFile
		var err error
		for {
			sf, err = stream.Recv()
			if err != nil {
				break
			}

			if r, ok := sf.Data.(*storage.StreamFile_Chunk); ok {
				// todo
				writer.Write(r.Chunk)
			}
		}
		writer.Close()
	}(writer)

	return reader, nil
}

func (api *storageClient) GenerateFileLink(ctx context.Context, fileId, domainId int64, source, action string, query map[string]string) (string, error) {
	uri, err := api.file.Api.GenerateFileLink(ctx, &storage.GenerateFileLinkRequest{
		DomainId: domainId,
		FileId:   fileId,
		Source:   source,
		Action:   action,
		Query:    query,
	})
	if err != nil {
		return "", err
	}
	return uri.Url, nil

}

func (api *storageClient) BulkGenerateFileLink(ctx context.Context, domainId int64, files []model.FileLinkRequest) ([]string, error) {
	var data []*storage.GenerateFileLinkRequest
	for _, v := range files {
		data = append(data, &storage.GenerateFileLinkRequest{
			DomainId: domainId,
			FileId:   v.FileId,
			Source:   v.Source,
			Action:   v.Action,
		})
	}
	res, err := api.file.Api.BulkGenerateFileLink(ctx, &storage.BulkGenerateFileLinkRequest{
		Files: data,
	})
	if err != nil {
		return nil, err
	}

	l := len(res.Links)
	links := make([]string, l, l)
	for k, v := range res.Links {
		links[k] = v.Url
	}

	return links, nil
}

func (api *storageClient) GetFileTranscription(ctx context.Context, fileId, domainId int64, profileId int64, language string) (string, error) {

	resp, err := api.transcript.Api.FileTranscriptSafe(ctx, &storage.FileTranscriptSafeRequest{
		FileId:    fileId,
		Locale:    language,
		ProfileId: profileId,
		DomainId:  domainId,
	})
	if err != nil {
		return "", err
	}
	return resp.Transcript, nil

}

func fileChannel(s string) storage.UploadFileChannel {
	switch s {
	case model.FileChannelCall:
		return storage.UploadFileChannel_CallChannel
	case model.FileChannelMail:
		return storage.UploadFileChannel_MailChannel
	case model.FileChannelChat:
		return storage.UploadFileChannel_ChatChannel
	default:
		return storage.UploadFileChannel_UnknownChannel
	}
}
