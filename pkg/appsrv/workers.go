package appsrv

import (
	"runtime/debug"
	"sync"

	"github.com/yunionio/log"
)

type WorkerManager struct {
	name         string
	queue        *Ring
	workerCount  int
	backlog      int
	activeWorker int
	workerLock   *sync.Mutex
	workerId     uint64
}

func NewWorkerManager(name string, workerCount int, backlog int) *WorkerManager {
	manager := WorkerManager{name: name,
		queue:        NewRing(workerCount * backlog),
		workerCount:  workerCount,
		backlog:      backlog,
		activeWorker: 0,
		workerLock:   &sync.Mutex{},
		workerId:     0}
	return &manager
}

type workerTask struct {
	task func()
	err  chan interface{}
}

func (wm *WorkerManager) Run(task func(), err chan interface{}) bool {
	ret := wm.queue.Push(&workerTask{task: task, err: err})
	if ret {
		wm.schedule()
	}
	return ret
}

func (wm *WorkerManager) workerRun(id uint64) {
	//log.Println("Start worker", id)
	defer wm.decActiveWorker()
	req := wm.queue.Pop()
	for req != nil {
		wm.execCallback(req.(*workerTask))
		req = wm.queue.Pop()
	}
	//log.Println("End worker", id)
}

func (wm *WorkerManager) decActiveWorker() {
	wm.workerLock.Lock()
	defer wm.workerLock.Unlock()
	wm.activeWorker -= 1
}

func (wm *WorkerManager) execCallback(task *workerTask) {
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

func (wm *WorkerManager) schedule() {
	wm.workerLock.Lock()
	defer wm.workerLock.Unlock()
	if wm.activeWorker < wm.workerCount && wm.queue.Size() > 0 {
		wm.activeWorker += 1
		wm.workerId += 1
		go wm.workerRun(wm.workerId)
	}
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
