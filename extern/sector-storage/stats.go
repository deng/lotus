package sectorstorage

import (
	"context"

	"github.com/filecoin-project/lotus/extern/sector-storage/sealtasks"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
)

func (m *Manager) WorkerStats() map[uint64]storiface.WorkerStats {
	m.sched.workersLk.RLock()
	defer m.sched.workersLk.RUnlock()

	out := map[uint64]storiface.WorkerStats{}

	for id, handle := range m.sched.workers {
		out[uint64(id)] = storiface.WorkerStats{
			Info:       handle.info,
			MemUsedMin: handle.active.memUsedMin,
			MemUsedMax: handle.active.memUsedMax,
			GpuUsed:    handle.active.gpuUsed,
			CpuUse:     handle.active.cpuUse,
		}
	}

	return out
}

func (m *Manager) WorkerJobs() map[uint64][]storiface.WorkerJob {
	m.sched.workersLk.RLock()
	defer m.sched.workersLk.RUnlock()

	out := map[uint64][]storiface.WorkerJob{}

	for id, handle := range m.sched.workers {
		out[uint64(id)] = handle.wt.Running()
	}

	return out
}

func (whnd *workerHandle) getTaskTypes(ctx context.Context) (map[sealtasks.TaskType]struct{}, error) {
	if whnd.supported == nil || len(whnd.supported) == 0 {
		log.Infof("=====TaskTypes===> %s", whnd.info.Url)
		tasks, err := whnd.w.TaskTypes(ctx)
		if err != nil {
			return nil, err
		}
		whnd.supported = tasks
	}
	return whnd.supported, nil
}

func (whnd *workerHandle) getPaths(ctx context.Context) ([]stores.StoragePath, error) {
	if whnd.paths == nil || len(whnd.paths) == 0 {
		log.Infof("=====Paths===> %s", whnd.info.Url)
		p, err := whnd.w.Paths(ctx)
		if err != nil {
			return nil, err
		}
		whnd.paths = p
	}
	return whnd.paths, nil
}
