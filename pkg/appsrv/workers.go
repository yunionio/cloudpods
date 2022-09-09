// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package appsrv

import (
	"container/list"
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	WORKER_STATE_ACTIVE = 0
	WORKER_STATE_DETACH = 1
)

var isDebug = false

func enableDebug() {
	isDebug = true
}

var (
	workerManagers []*SWorkerManager

	workerManagerLock *sync.Mutex
)

func init() {
	workerManagers = make([]*SWorkerManager, 0)

	workerManagerLock = &sync.Mutex{}
}

type SWorker struct {
	id        uint64
	state     int
	container *list.Element
	task      *sWorkerTask
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
	defer worker.manager.removeWorker(worker)
	for {
		if worker.isDetached() {
			break
		}
		req := worker.manager.queue.Pop()
		if req != nil {
			task := req.(*sWorkerTask)
			if task.worker != nil {
				task.worker <- worker
			}
			task.start = time.Now()
			worker.task = task
			execCallback(task)
		} else {
			break
		}
	}
}

func (worker *SWorker) Detach(reason string) {
	worker.manager.workerLock.Lock()
	defer worker.manager.workerLock.Unlock()

	worker.state = WORKER_STATE_DETACH
	worker.manager.activeWorker.removeWithLock(worker)
	worker.manager.detachedWorker.addWithLock(worker)

	log.Warningf("detach worker %s(%s) due to reason %s after %s", worker, worker.task.task.Dump(), reason, time.Now().Sub(worker.task.start))

	worker.manager.scheduleWithLock()
}

func (worker *SWorker) StateStr() string {
	if worker.state == WORKER_STATE_ACTIVE {
		return "active"
	} else {
		return "detach"
	}
}

func (worker *SWorker) String() string {
	workerInfo := ""
	if worker.task != nil {
		workerInfo = worker.task.task.Dump()
	}
	return fmt.Sprintf("#%d(%p, %s) %s", worker.id, worker, worker.StateStr(), workerInfo)
}

type SWorkerList struct {
	list *list.List
}

func newWorkerList() *SWorkerList {
	return &SWorkerList{
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
	activeWorker   *SWorkerList
	detachedWorker *SWorkerList
	workerLock     *sync.Mutex
	workerId       uint64
	dbWorker       bool

	ignoreOverflow bool
}

func NewWorkerManager(name string, workerCount int, backlog int, dbWorker bool) *SWorkerManager {
	return NewWorkerManagerIgnoreOverflow(name, workerCount, backlog, dbWorker, false)
}

func NewWorkerManagerIgnoreOverflow(name string, workerCount int, backlog int, dbWorker bool, ignoreOverflow bool) *SWorkerManager {
	manager := SWorkerManager{name: name,
		queue:          NewRing(workerCount * backlog),
		workerCount:    workerCount,
		backlog:        backlog,
		activeWorker:   newWorkerList(),
		detachedWorker: newWorkerList(),
		workerLock:     &sync.Mutex{},
		workerId:       0,
		dbWorker:       dbWorker,

		ignoreOverflow: ignoreOverflow,
	}

	workerManagerLock.Lock()
	defer workerManagerLock.Unlock()

	workerManagers = append(workerManagers, &manager)
	return &manager
}

type IWorkerTask interface {
	Run()
	Dump() string
}

type sWorkerTask struct {
	task    IWorkerTask
	worker  chan *SWorker
	onError func(error)
	start   time.Time
}

func (wm *SWorkerManager) UpdateWorkerCount(workerCount int) error {
	wm.workerLock.Lock()
	defer wm.workerLock.Unlock()
	if wm.queue.Size() > 0 {
		return errors.Errorf("worker queue is not empty")
	}
	wm.queue = NewRing(workerCount * wm.backlog)
	wm.workerCount = workerCount
	return nil
}

func (wm *SWorkerManager) String() string {
	return wm.name
}

func (wm *SWorkerManager) Run(task IWorkerTask, worker chan *SWorker, onErr func(error)) bool {
	ret := wm.queue.Push(&sWorkerTask{task: task, worker: worker, onError: onErr})
	if ret {
		wm.schedule()
	} else if !wm.ignoreOverflow {
		log.Warningf("[%s] queue full, task dropped", wm)
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
			if task.onError != nil {
				task.onError(fmt.Errorf("%s", r))
			}
			debug.PrintStack()
		}
	}()
	task.task.Run()
}

func (wm *SWorkerManager) schedule() {
	wm.workerLock.Lock()
	defer wm.workerLock.Unlock()

	wm.scheduleWithLock()
}

func (wm *SWorkerManager) scheduleWithLock() {
	queueSize := wm.queue.Size()
	if wm.activeWorker.size() < wm.workerCount && queueSize > 0 {
		wm.workerId += 1
		worker := newWorker(wm.workerId, wm)
		wm.activeWorker.addWithLock(worker)
		if isDebug {
			log.Debugf("no enough worker, add new worker %s", worker)
		}
		go worker.run()
	} else if queueSize > 10 {
		log.Warningf("[%s] BUSY activeWork %d detachedWork %d max %d queue: %d", wm, wm.ActiveWorkerCount(), wm.DetachedWorkerCount(), wm.workerCount, wm.queue.Size())
	} else if queueSize > 50 {
		w := wm.activeWorker.list.Front()
		for w != nil {
			worker := w.Value.(*SWorker)
			log.Warningf("work [%s]%s stucking for a while", worker.task.start, worker.task.task.Dump())
			w = w.Next()
		}
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
	QueueCnt        int
	MaxWorkerCnt    int
	ActiveWorkerCnt int
	DetachWorkerCnt int
	DbWorker        bool
	AllowOverflow   bool
}

func (s SWorkerManagerStates) IsBusy() bool {
	if s.QueueCnt == 0 && s.ActiveWorkerCnt == 0 && s.DetachWorkerCnt == 0 {
		return false
	}
	return true
}

func (wm *SWorkerManager) getState() SWorkerManagerStates {
	state := SWorkerManagerStates{}

	state.Name = wm.name
	state.Backlog = wm.backlog
	state.QueueCnt = wm.queue.Size()
	state.MaxWorkerCnt = wm.workerCount
	state.ActiveWorkerCnt = wm.activeWorker.size()
	state.DetachWorkerCnt = wm.detachedWorker.size()
	state.DbWorker = wm.dbWorker
	state.AllowOverflow = wm.ignoreOverflow

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

func GetDBConnectionCount() int {
	conn := 0
	for i := 0; i < len(workerManagers); i += 1 {
		if workerManagers[i].dbWorker {
			conn += workerManagers[i].workerCount
		}
	}
	return conn
}
