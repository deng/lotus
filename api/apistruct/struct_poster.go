package apistruct

import "context"

type PosterStruct struct {
	Internal struct {
		StorageAddLocal func(ctx context.Context, path string) error `perm:"admin"`
	}
}

func (c *PosterStruct) StorageAddLocal(ctx context.Context, path string) error {
	return c.Internal.StorageAddLocal(ctx, path)
}
