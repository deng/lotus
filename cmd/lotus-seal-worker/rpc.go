package main

import (
	"context"

	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/lotus/storage/sector"

	"github.com/filecoin-project/lotus/build"
)

type worker struct {
	*sector.LocalWorker
}

func (w *worker) Version(context.Context) (build.Version, error) {
	return build.APIVersion, nil
}

var _ storage.Sealer = &worker{}
