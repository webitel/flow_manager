package meeting

import "context"

type Client interface {
	Create(ctx context.Context, domainId int64, title string, expireSec int, basePath string, vars map[string]string) (string, error)
	Get(ctx context.Context, id string) (map[string]string, error)
}
