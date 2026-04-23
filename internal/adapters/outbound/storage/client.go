package storage

import (
	"context"
	"io"

	"github.com/webitel/flow_manager/gen/storage"
	"github.com/webitel/flow_manager/infra/grpcdial"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
)

const serviceName = "storage"

type StorageClient struct {
	file       *grpcdial.Client[storage.FileServiceClient]
	transcript *grpcdial.Client[storage.FileTranscriptServiceClient]
}

func NewStorageClient(consulTarget string) (*StorageClient, error) {
	file, err := grpcdial.NewClient(consulTarget, serviceName, storage.NewFileServiceClient)
	if err != nil {
		return nil, err
	}
	transcript, err := grpcdial.NewClient(consulTarget, serviceName, storage.NewFileTranscriptServiceClient)
	if err != nil {
		return nil, err
	}

	return &StorageClient{
		file:       file,
		transcript: transcript,
	}, nil
}

func (api *StorageClient) Upload(ctx context.Context, domainId int64, uuid string, sFile io.Reader, metadata domstorage.File) (domstorage.File, error) {
	stream, err := api.file.API.UploadFile(ctx)
	if err != nil {
		return domstorage.File{}, err
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
		return domstorage.File{}, err
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
		return domstorage.File{}, err
	}

	var res *storage.UploadFileResponse
	res, err = stream.CloseAndRecv()
	if err != nil {
		return domstorage.File{}, err
	}

	metadata.Id = int(res.FileId)
	metadata.Size = res.Size
	metadata.Url = res.FileUrl
	metadata.PublicUrl = res.Server + res.FileUrl

	return metadata, nil
}

func (api *StorageClient) Download(ctx context.Context, domainId, id int64) (io.ReadCloser, error) {
	stream, err := api.file.API.DownloadFile(ctx, &storage.DownloadFileRequest{
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
				writer.Write(r.Chunk)
			}
		}
		writer.Close()
	}(writer)

	return reader, nil
}

func (api *StorageClient) GenerateFileLink(ctx context.Context, fileId, domainId int64, source, action string, query map[string]string) (string, error) {
	uri, err := api.file.API.GenerateFileLink(ctx, &storage.GenerateFileLinkRequest{
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

func (api *StorageClient) BulkGenerateFileLink(ctx context.Context, domainId int64, files []domstorage.FileLinkRequest) ([]string, error) {
	data := make([]*storage.GenerateFileLinkRequest, len(files))
	for i, v := range files {
		data[i] = &storage.GenerateFileLinkRequest{
			DomainId: domainId,
			FileId:   v.FileId,
			Source:   v.Source,
			Action:   v.Action,
		}
	}
	res, err := api.file.API.BulkGenerateFileLink(ctx, &storage.BulkGenerateFileLinkRequest{
		Files: data,
	})
	if err != nil {
		return nil, err
	}

	links := make([]string, len(res.Links))
	for k, v := range res.Links {
		links[k] = v.Url
	}

	return links, nil
}

func (api *StorageClient) GetFileTranscription(ctx context.Context, fileId, domainId, profileId int64, language string) (string, error) {
	resp, err := api.transcript.API.FileTranscriptSafe(ctx, &storage.FileTranscriptSafeRequest{
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
	case domstorage.ChannelCall:
		return storage.UploadFileChannel_CallChannel
	case domstorage.ChannelMail:
		return storage.UploadFileChannel_MailChannel
	case domstorage.ChannelChat:
		return storage.UploadFileChannel_ChatChannel
	default:
		return storage.UploadFileChannel_UnknownChannel
	}
}
