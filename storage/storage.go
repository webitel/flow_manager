package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/webitel/flow_manager/model"

	"github.com/webitel/protos/storage"
	"google.golang.org/grpc"

	_ "github.com/mbobakov/grpc-consul-resolver"
)

type Api struct {
	service storage.FileServiceClient
}

func NewClient(consulTarget string) (*Api, *model.AppError) {
	conn, err := grpc.Dial(fmt.Sprintf("consul://%s/storage?wait=14s", consulTarget),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, model.NewAppError("Storage", "storage.create_client.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	service := storage.NewFileServiceClient(conn)
	return &Api{
		service: service,
	}, nil
}

func (api *Api) Upload(ctx context.Context, domainId int64, uuid string, sFile io.Reader, metadata model.File) (model.File, error) {
	stream, err := api.service.UploadFile(ctx)
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
