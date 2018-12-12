package workmanager

import (
	"context"
	"sync/atomic"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

type DelayTaskFunc func(ctx context.Context, params jsonutils.JSONObject)

type SWorkManager struct {
	curCount int32
}

func (w *SWorkManager) add() {
	atomic.AddInt32(&w.curCount, 1)
}

func (w *SWorkManager) done() {
	atomic.AddInt32(&w.curCount, -1)
}

func (w *SWorkManager) DelayTask(task func()) {
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
		time.Sleep(1)
	}
}

func NewWorkManger() *SWorkManager {
	return &SWorkManager{}
}
