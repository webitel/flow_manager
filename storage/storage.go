package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/webitel/flow_manager/model"

	"github.com/webitel/protos/storage"
	"google.golang.org/grpc"

	_ "github.com/mbobakov/grpc-consul-resolver"
)

type Api struct {
	file storage.FileServiceClient
}

func NewClient(consulTarget string) (*Api, error) {
	conn, err := grpc.Dial(fmt.Sprintf("consul://%s/storage?wait=14s", consulTarget),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	fileService := storage.NewFileServiceClient(conn)
	return &Api{
		file: fileService,
	}, nil
}

func (api *Api) Upload(ctx context.Context, domainId int64, uuid string, sFile io.Reader, metadata model.File) (model.File, error) {
	stream, err := api.file.UploadFile(ctx)
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

func (api *Api) Download(ctx context.Context, domainId int64, id int64) (io.ReadCloser, error) {
	stream, err := api.file.DownloadFile(ctx, &storage.DownloadFileRequest{
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
