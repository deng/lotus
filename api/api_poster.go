package api

import "context"

type PosterAPI interface {
	StorageAddLocal(ctx context.Context, path string) error
}
