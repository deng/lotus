package sectorstorage

import (
	"container/heap"
	"context"
	"math/rand"
	"sort"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-actors/actors/abi"

	"github.com/filecoin-project/sector-storage/sealtasks"
	"github.com/filecoin-project/sector-storage/storiface"
)

type schedPrioCtxKey int

var SchedPriorityKey schedPrioCtxKey
var DefaultSchedPriority = 0
var SelectorTimeout = 5 * time.Second

var (
	SchedWindows = 2
)

func getPriority(ctx context.Context) int {
	sp := ctx.Value(SchedPriorityKey)
	if p, ok := sp.(int); ok {
		return p
	}

	return DefaultSchedPriority
}

func WithPriority(ctx context.Context, priority int) context.Context {
	return context.WithValue(ctx, SchedPriorityKey, priority)
}

const mib = 1 << 20

type WorkerAction func(ctx context.Context, w Worker) error

type WorkerSelector interface {
	Ok(ctx context.Context, task sealtasks.TaskType, spt abi.RegisteredSealProof, a *workerHandle) (bool, error) // true if worker is acceptable for performing a task

	Cmp(ctx context.Context, task sealtasks.TaskType, a, b *workerHandle) (bool, error) // true if a is preferred over b
}

type scheduler struct {
	spt abi.RegisteredSealProof

	workersLk  sync.Mutex
	nextWorker WorkerID
	workers    map[WorkerID]*workerHandle

	newWorkers chan *workerHandle

	watchClosing  chan WorkerID
	workerClosing chan WorkerID

	schedule       chan *workerRequest
	windowRequests chan *schedWindowRequest

	// owned by the sh.runSched goroutine
	schedQueue  *requestQueue
	openWindows []*schedWindowRequest

	closing  chan struct{}
	testSync chan struct{} // used for testing
}

type workerHandle struct {
	w Worker

	info storiface.WorkerInfo

	preparing *activeResources
	active    *activeResources
}

type schedWindowRequest struct {
	worker WorkerID

	done chan *schedWindow
}

type schedWindow struct {
	allocated activeResources
	todo      []*workerRequest
}

type activeResources struct {
	memUsedMin uint64
	memUsedMax uint64
	gpuUsed    bool
	cpuUse     uint64

	cond *sync.Cond
}

type workerRequest struct {
	sector   abi.SectorID
	taskType sealtasks.TaskType
	priority int // larger values more important
	sel      WorkerSelector

	prepare WorkerAction
	work    WorkerAction

	index int // The index of the item in the heap.

	ret chan<- workerResponse
	ctx context.Context
}

type workerResponse struct {
	err error
}

func newScheduler(spt abi.RegisteredSealProof) *scheduler {
	return &scheduler{
		spt: spt,

		nextWorker: 0,
		workers:    map[WorkerID]*workerHandle{},

		newWorkers: make(chan *workerHandle),

		watchClosing:  make(chan WorkerID),
		workerClosing: make(chan WorkerID),

		schedule:       make(chan *workerRequest),
		windowRequests: make(chan *schedWindowRequest),

		schedQueue: &requestQueue{},

		closing: make(chan struct{}),
	}
}

func (sh *scheduler) Schedule(ctx context.Context, sector abi.SectorID, taskType sealtasks.TaskType, sel WorkerSelector, prepare WorkerAction, work WorkerAction) error {
	ret := make(chan workerResponse)

	select {
	case sh.schedule <- &workerRequest{
		sector:   sector,
		taskType: taskType,
		priority: getPriority(ctx),
		sel:      sel,

		prepare: prepare,
		work:    work,

		ret: ret,
		ctx: ctx,
	}:
	case <-sh.closing:
		return xerrors.New("closing")
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case resp := <-ret:
		return resp.err
	case <-sh.closing:
		return xerrors.New("closing")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *workerRequest) respond(err error) {
	select {
	case r.ret <- workerResponse{err: err}:
	case <-r.ctx.Done():
		log.Warnf("request got cancelled before we could respond")
	}
}

func (sh *scheduler) runSched() {
	go sh.runWorkerWatcher()

	for {
		select {
		case w := <-sh.newWorkers:
			sh.newWorker(w)

		case wid := <-sh.workerClosing:
			sh.dropWorker(wid)

		case req := <-sh.schedule:
			heap.Push(sh.schedQueue, req)
			sh.trySched()

			if sh.testSync != nil {
				sh.testSync <- struct{}{}
			}
		case req := <-sh.windowRequests:
			sh.openWindows = append(sh.openWindows, req)
			sh.trySched()

		case <-sh.closing:
			sh.schedClose()
			return
		}
	}
}

func (sh *scheduler) trySched() {
	/*
		This assigns tasks to workers based on:
		- Task priority (achieved by handling sh.schedQueue in order, since it's already sorted by priority)
		- Worker resource availability
		- Task-specified worker preference (acceptableWindows array below sorted by this preference)
		- Window request age

		1. For each task in the schedQueue find windows which can handle them
		1.1. Create list of windows capable of handling a task
		1.2. Sort windows according to task selector preferences
		2. Going through schedQueue again, assign task to first acceptable window
		   with resources available
		3. Submit windows with scheduled tasks to workers

	*/

	windows := make([]schedWindow, len(sh.openWindows))
	acceptableWindows := make([][]int, sh.schedQueue.Len())

	log.Debugf("trySched %d queued; %d open windows", sh.schedQueue.Len(), len(windows))

	// Step 1
	for sqi := 0; sqi < sh.schedQueue.Len(); sqi++ {
		task := (*sh.schedQueue)[sqi]
		needRes := ResourceTable[task.taskType][sh.spt]

		for wnd, windowRequest := range sh.openWindows {
			worker := sh.workers[windowRequest.worker]

			// TODO: allow bigger windows
			if !windows[wnd].allocated.canHandleRequest(needRes, windowRequest.worker, worker.info.Resources) {
				continue
			}

			ok, err := task.sel.Ok(task.ctx, task.taskType, sh.spt, worker)
			if err != nil {
				log.Errorf("trySched(1) req.sel.Ok error: %+v", err)
				continue
			}

			if !ok {
				continue
			}

			acceptableWindows[sqi] = append(acceptableWindows[sqi], wnd)
		}

		if len(acceptableWindows[sqi]) == 0 {
			continue
		}

		// Pick best worker (shuffle in case some workers are equally as good)
		rand.Shuffle(len(acceptableWindows[sqi]), func(i, j int) {
			acceptableWindows[sqi][i], acceptableWindows[sqi][j] = acceptableWindows[sqi][j], acceptableWindows[sqi][i]
		})
		sort.SliceStable(acceptableWindows[sqi], func(i, j int) bool {
			wii := sh.openWindows[acceptableWindows[sqi][i]].worker
			wji := sh.openWindows[acceptableWindows[sqi][j]].worker

			if wii == wji {
				// for the same worker prefer older windows
				return acceptableWindows[sqi][i] < acceptableWindows[sqi][j]
			}

			wi := sh.workers[wii]
			wj := sh.workers[wji]

			rpcCtx, cancel := context.WithTimeout(task.ctx, SelectorTimeout)
			defer cancel()

			r, err := task.sel.Cmp(rpcCtx, task.taskType, wi, wj)
			if err != nil {
				log.Error("selecting best worker: %s", err)
			}
			return r
		})
	}

	// Step 2
	scheduled := 0

	for sqi := 0; sqi < sh.schedQueue.Len(); sqi++ {
		task := (*sh.schedQueue)[sqi]
		needRes := ResourceTable[task.taskType][sh.spt]

		selectedWindow := -1
		for _, wnd := range acceptableWindows[sqi+scheduled] {
			wid := sh.openWindows[wnd].worker
			wr := sh.workers[wid].info.Resources

			log.Debugf("trySched try assign sqi:%d sector %d to window %d", sqi, task.sector.Number, wnd)

			// TODO: allow bigger windows
			if !windows[wnd].allocated.canHandleRequest(needRes, wid, wr) {
				continue
			}

			log.Debugf("trySched ASSIGNED sqi:%d sector %d to window %d", sqi, task.sector.Number, wnd)

			windows[wnd].allocated.add(wr, needRes)

			selectedWindow = wnd
			break
		}

		if selectedWindow < 0 {
			// all windows full
			continue
		}

		windows[selectedWindow].todo = append(windows[selectedWindow].todo, task)

		heap.Remove(sh.schedQueue, sqi)
		sqi--
		scheduled++
	}

	// Step 3

	if scheduled == 0 {
		return
	}

	scheduledWindows := map[int]struct{}{}
	for wnd, window := range windows {
		if len(window.todo) == 0 {
			// Nothing scheduled here, keep the window open
			continue
		}

		scheduledWindows[wnd] = struct{}{}

		window := window // copy
		select {
		case sh.openWindows[wnd].done <- &window:
		default:
			log.Error("expected sh.openWindows[wnd].done to be buffered")
		}
	}

	// Rewrite sh.openWindows array, removing scheduled windows
	newOpenWindows := make([]*schedWindowRequest, 0, len(sh.openWindows)-len(scheduledWindows))
	for wnd, window := range sh.openWindows {
		if _, scheduled := scheduledWindows[wnd]; scheduled {
			// keep unscheduled windows open
			continue
		}

		newOpenWindows = append(newOpenWindows, window)
	}

	sh.openWindows = newOpenWindows
}

func (sh *scheduler) runWorker(wid WorkerID) {
	go func() {
		worker := sh.workers[wid]
		scheduledWindows := make(chan *schedWindow, SchedWindows)
		taskDone := make(chan struct{}, 1)
		windowsRequested := 0

		var activeWindows []*schedWindow

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		workerClosing, err := worker.w.Closing(ctx)
		if err != nil {
			return
		}

		defer func() {
			log.Warnw("Worker closing", "workerid", wid)

			// TODO: close / return all queued tasks
		}()

		for {
			// ask for more windows if we need them
			for ; windowsRequested < SchedWindows; windowsRequested++ {
				select {
				case sh.windowRequests <- &schedWindowRequest{
					worker: wid,
					done:   scheduledWindows,
				}:
				case <-sh.closing:
					return
				case <-workerClosing:
					return
				}
			}

			select {
			case w := <-scheduledWindows:
				activeWindows = append(activeWindows, w)
			case <-taskDone:
				log.Debugw("task done", "workerid", wid)
			case <-sh.closing:
				return
			case <-workerClosing:
				return
			}

		assignLoop:
			// process windows in order
			for len(activeWindows) > 0 {
				// process tasks within a window in order
				for len(activeWindows[0].todo) > 0 {
					todo := activeWindows[0].todo[0]
					needRes := ResourceTable[todo.taskType][sh.spt]

					sh.workersLk.Lock()
					ok := worker.preparing.canHandleRequest(needRes, wid, worker.info.Resources)
					if !ok {
						sh.workersLk.Unlock()
						break assignLoop
					}

					log.Debugf("assign worker sector %d", todo.sector.Number)
					err := sh.assignWorker(taskDone, wid, worker, todo)
					sh.workersLk.Unlock()

					if err != nil {
						log.Error("assignWorker error: %+v", err)
						go todo.respond(xerrors.Errorf("assignWorker error: %w", err))
					}

					activeWindows[0].todo = activeWindows[0].todo[1:]
				}

				copy(activeWindows, activeWindows[1:])
				activeWindows[len(activeWindows)-1] = nil
				activeWindows = activeWindows[:len(activeWindows)-1]

				windowsRequested--
			}
		}
	}()
}

func (sh *scheduler) assignWorker(taskDone chan struct{}, wid WorkerID, w *workerHandle, req *workerRequest) error {
	needRes := ResourceTable[req.taskType][sh.spt]

	w.preparing.add(w.info.Resources, needRes)

	go func() {
		err := req.prepare(req.ctx, w.w)
		sh.workersLk.Lock()

		if err != nil {
			w.preparing.free(w.info.Resources, needRes)
			sh.workersLk.Unlock()

			select {
			case taskDone <- struct{}{}:
			case <-sh.closing:
				log.Warnf("scheduler closed while sending response (prepare error: %+v)", err)
			}

			select {
			case req.ret <- workerResponse{err: err}:
			case <-req.ctx.Done():
				log.Warnf("request got cancelled before we could respond (prepare error: %+v)", err)
			case <-sh.closing:
				log.Warnf("scheduler closed while sending response (prepare error: %+v)", err)
			}
			return
		}

		err = w.active.withResources(wid, w.info.Resources, needRes, &sh.workersLk, func() error {
			w.preparing.free(w.info.Resources, needRes)
			sh.workersLk.Unlock()
			defer sh.workersLk.Lock() // we MUST return locked from this function

			select {
			case taskDone <- struct{}{}:
			case <-sh.closing:
			}

			err = req.work(req.ctx, w.w)

			select {
			case req.ret <- workerResponse{err: err}:
			case <-req.ctx.Done():
				log.Warnf("request got cancelled before we could respond")
			case <-sh.closing:
				log.Warnf("scheduler closed while sending response")
			}

			return nil
		})

		sh.workersLk.Unlock()

		// This error should always be nil, since nothing is setting it, but just to be safe:
		if err != nil {
			log.Errorf("error executing worker (withResources): %+v", err)
		}
	}()

	return nil
}

func (sh *scheduler) newWorker(w *workerHandle) {
	sh.workersLk.Lock()

	id := sh.nextWorker
	sh.workers[id] = w
	sh.nextWorker++

	sh.workersLk.Unlock()

	select {
	case sh.watchClosing <- id:
	case <-sh.closing:
		return
	}

	sh.runWorker(id)
}

func (sh *scheduler) dropWorker(wid WorkerID) {
	sh.workersLk.Lock()
	defer sh.workersLk.Unlock()

	w := sh.workers[wid]
	delete(sh.workers, wid)

	newWindows := make([]*schedWindowRequest, 0, len(sh.openWindows))
	for _, window := range sh.openWindows {
		if window.worker != wid {
			newWindows = append(newWindows, window)
		}
	}
	sh.openWindows = newWindows

	// TODO: sync close worker goroutine

	go func() {
		if err := w.w.Close(); err != nil {
			log.Warnf("closing worker %d: %+v", err)
		}
	}()
}

func (sh *scheduler) schedClose() {
	sh.workersLk.Lock()
	defer sh.workersLk.Unlock()

	for i, w := range sh.workers {
		if err := w.w.Close(); err != nil {
			log.Errorf("closing worker %d: %+v", i, err)
		}
	}
}

func (sh *scheduler) Close() error {
	close(sh.closing)
	return nil
}
