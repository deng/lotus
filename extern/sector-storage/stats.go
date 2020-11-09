package sectorstorage

import (
	"context"
	"github.com/filecoin-project/lotus/extern/sector-storage/sealtasks"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	"time"

	"github.com/google/uuid"

	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
)

func (m *Manager) WorkerStats() map[uuid.UUID]storiface.WorkerStats {
	m.sched.workersLk.RLock()
	defer m.sched.workersLk.RUnlock()

	out := map[uuid.UUID]storiface.WorkerStats{}

	for id, handle := range m.sched.workers {
		out[uuid.UUID(id)] = storiface.WorkerStats{
			Info:    handle.info,
			Enabled: handle.enabled,

			MemUsedMin: handle.active.memUsedMin,
			MemUsedMax: handle.active.memUsedMax,
			GpuUsed:    handle.active.gpuUsed,
			CpuUse:     handle.active.cpuUse,
		}
	}

	return out
}

func (m *Manager) WorkerJobs() map[uuid.UUID][]storiface.WorkerJob {
	out := map[uuid.UUID][]storiface.WorkerJob{}
	calls := map[storiface.CallID]struct{}{}

	for _, t := range m.sched.workTracker.Running() {
		out[uuid.UUID(t.worker)] = append(out[uuid.UUID(t.worker)], t.job)
		calls[t.job.ID] = struct{}{}
	}

	m.sched.workersLk.RLock()

	for id, handle := range m.sched.workers {
		handle.wndLk.Lock()
		for wi, window := range handle.activeWindows {
			for _, request := range window.todo {
				out[uuid.UUID(id)] = append(out[uuid.UUID(id)], storiface.WorkerJob{
					ID:      storiface.UndefCall,
					Sector:  request.sector,
					Task:    request.taskType,
					RunWait: wi + 1,
					Start:   request.start,
				})
			}
		}
		handle.wndLk.Unlock()
	}

	m.sched.workersLk.RUnlock()

	m.workLk.Lock()
	defer m.workLk.Unlock()

	for id, work := range m.callToWork {
		_, found := calls[id]
		if found {
			continue
		}

		out[uuid.UUID{}] = append(out[uuid.UUID{}], storiface.WorkerJob{
			ID:      id,
			Sector:  id.Sector,
			Task:    work.Method,
			RunWait: -1,
			Start:   time.Time{},
		})
	}

	return out
}

func (whnd *workerHandle) getTaskTypes(ctx context.Context) (map[sealtasks.TaskType]struct{}, error) {
	if whnd.supported == nil || len(whnd.supported) == 0 {
		log.Infof("=====TaskTypes===> %s", whnd.info.Url)
		tasks, err := whnd.workerRpc.TaskTypes(ctx)
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
		p, err := whnd.workerRpc.Paths(ctx)
		if err != nil {
			return nil, err
		}
		whnd.paths = p
	}
	return whnd.paths, nil
}
