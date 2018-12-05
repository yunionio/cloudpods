package workmanager

import (
	"sync/atomic"

	"yunion.io/x/log"
)

type SWorkManager struct {
	curCount int32
}

func (w *SWorkManager) add() {
	atomic.AddInt32(&w.curCount, 1)
}

func (w *SWorkManager) done() {
	atomic.AddInt32(&w.curCount, -1)
}

func (w *SWorkManager) RunTask(task func()) {
	w.add()
	go func() {
		defer w.done()
		task()
	}()
}

func (w *SWorkManager) Stop() {
	log.Infof("WorkManager To stop, wait for workers ...")
	for w.curCount > 0 {
		log.Warningf("Busy workers count %d, waiting stopped", w.curCount)
	}
}

func NewWorkManger() *SWorkManager {
	return &SWorkManager{}
}
