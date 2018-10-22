package appsrv

import (
	"container/list"
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

const (
	WORKER_STATE_ACTIVE = 0
	WORKER_STATE_DETACH = 1
)

var isDebug = false

func enableDebug() {
	isDebug = true
}

var workerManagers []*SWorkerManager

func init() {
	workerManagers = make([]*SWorkerManager, 0)
}

type SWorker struct {
	id        uint64
	state     int
	container *list.Element
	manager   *SWorkerManager
}

func newWorker(id uint64, manager *SWorkerManager) *SWorker {
	return &SWorker{
		id:        id,
		state:     WORKER_STATE_ACTIVE,
		container: nil,
		manager:   manager,
	}
}

func (worker *SWorker) isDetached() bool {
	worker.manager.workerLock.Lock()
	defer worker.manager.workerLock.Unlock()

	return worker.state == WORKER_STATE_DETACH
}

func (worker *SWorker) run() {
	for {
		if worker.isDetached() {
			if isDebug {
				log.Debugf("deteched worker %s, no need to pick up new job", worker)
			}
			break
		}
		req := worker.manager.queue.Pop()
		if req != nil {
			task := req.(*sWorkerTask)
			if task.worker != nil {
				task.worker <- worker
			}
			if isDebug {
				log.Debugf("start exec task on worker %s", worker)
			}
			execCallback(task)
			if isDebug {
				log.Debugf("end exec task on worker %s", worker)
			}
		} else {
			if isDebug {
				log.Debugf("no more job, exit worker %s", worker)
			}
			break
		}
	}
	worker.manager.removeWorker(worker)
}

func (worker *SWorker) Detach(reason string) {
	worker.manager.workerLock.Lock()
	defer worker.manager.workerLock.Unlock()

	worker.state = WORKER_STATE_DETACH
	worker.manager.activeWorker.removeWithLock(worker)
	worker.manager.detachedWorker.addWithLock(worker)

	log.Warningf("detach worker %s due to reason %s", worker, reason)
}

func (worker *SWorker) String() string {
	return fmt.Sprintf("#%d(%d)", worker.id, worker.state)
}

type SWorkerList struct {
	list *list.List
}

func newWorkerList() SWorkerList {
	return SWorkerList{
		list: list.New(),
	}
}

func (wl *SWorkerList) addWithLock(worker *SWorker) {
	ele := wl.list.PushBack(worker)
	worker.container = ele
}

func (wl *SWorkerList) removeWithLock(worker *SWorker) {
	wl.list.Remove(worker.container)
	worker.container = nil
}

func (wl *SWorkerList) size() int {
	return wl.list.Len()
}

type SWorkerManager struct {
	name           string
	queue          *Ring
	workerCount    int
	backlog        int
	activeWorker   SWorkerList
	detachedWorker SWorkerList
	workerLock     *sync.Mutex
	workerId       uint64
}

func NewWorkerManager(name string, workerCount int, backlog int) *SWorkerManager {
	manager := SWorkerManager{name: name,
		queue:          NewRing(workerCount * backlog),
		workerCount:    workerCount,
		backlog:        backlog,
		activeWorker:   newWorkerList(),
		detachedWorker: newWorkerList(),
		workerLock:     &sync.Mutex{},
		workerId:       0}

	workerManagers = append(workerManagers, &manager)
	return &manager
}

type sWorkerTask struct {
	task   func()
	worker chan *SWorker
	err    chan interface{}
}

func (wm *SWorkerManager) Run(task func(), worker chan *SWorker, err chan interface{}) bool {
	ret := wm.queue.Push(&sWorkerTask{task: task, worker: worker, err: err})
	if ret {
		wm.schedule()
	}
	return ret
}

func (wm *SWorkerManager) removeWorker(worker *SWorker) {
	wm.workerLock.Lock()
	defer wm.workerLock.Unlock()

	if worker.state == WORKER_STATE_ACTIVE {
		wm.activeWorker.removeWithLock(worker)
	} else {
		wm.detachedWorker.removeWithLock(worker)
	}
}

func execCallback(task *sWorkerTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("WorkerManager exec callback error: %s", r)
			debug.PrintStack()
			if task.err != nil {
				task.err <- r
				close(task.err)
			}
		}
	}()
	task.task()
	if task.err != nil {
		close(task.err)
	}
}

func (wm *SWorkerManager) schedule() {
	wm.workerLock.Lock()
	defer wm.workerLock.Unlock()

	if wm.activeWorker.size() < wm.workerCount && wm.queue.Size() > 0 {
		wm.workerId += 1
		worker := newWorker(wm.workerId, wm)
		wm.activeWorker.addWithLock(worker)
		if isDebug {
			log.Debugf("no enough worker, add new worker %s", worker)
		}
		go worker.run()
	}
}

func (wm *SWorkerManager) ActiveWorkerCount() int {
	return wm.activeWorker.size()
}

func (wm *SWorkerManager) DetachedWorkerCount() int {
	return wm.detachedWorker.size()
}

type SWorkerManagerStates struct {
	Name            string
	Backlog         int
	ActiveWorkerCnt int
	DetachWorkerCnt int
}

func (wm *SWorkerManager) getState() SWorkerManagerStates {
	state := SWorkerManagerStates{}

	state.Name = wm.name
	state.Backlog = wm.queue.Size()
	state.ActiveWorkerCnt = wm.activeWorker.size()
	state.DetachWorkerCnt = wm.detachedWorker.size()

	return state
}

func WorkerStatsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	stats := make([]SWorkerManagerStates, 0)
	for i := 0; i < len(workerManagers); i += 1 {
		stats = append(stats, workerManagers[i].getState())
	}
	result := jsonutils.NewDict()
	result.Add(jsonutils.Marshal(&stats), "workers")
	fmt.Fprintf(w, result.String())
}

func WaitChannel(ch chan interface{}) interface{} {
	var ret interface{}
	stop := false
	for !stop {
		select {
		case c, more := <-ch:
			if more {
				ret = c
			} else {
				stop = true
			}
		}
	}
	return ret
}
