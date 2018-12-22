package workmanager

import (
	"context"
	"sync/atomic"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/httpclients"
)

type DelayTaskFunc func(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

type SWorkManager struct {
	curCount int32
}

func (w *SWorkManager) add() {
	atomic.AddInt32(&w.curCount, 1)
}

func (w *SWorkManager) done() {
	atomic.AddInt32(&w.curCount, -1)
}

func (w *SWorkManager) DelayTask(ctx context.Context, task DelayTaskFunc, params interface{}) {
	w.add()
	go func() {
		defer w.done()
		defer func() {
			if r := recover(); r != nil {
				log.Errorln("Delay task recover: ", r)
				switch val := r.(type) {
				case string:
					httpclients.TaskFailed(ctx, val)
				case error:
					httpclients.TaskFailed(ctx, val.Error())
				default:
					httpclients.TaskFailed(ctx, "Unknown panic")
				}
			}
		}()
		res, err := task(ctx, params)
		if err != nil {
			httpclients.TaskFailed(ctx, err.Error())
		} else {
			httpclients.TaskComplete(ctx, res)
		}
	}()
}

func (w *SWorkManager) Stop() {
	log.Infof("WorkManager To stop, wait for workers ...")
	for w.curCount > 0 {
		log.Warningf("Busy workers count %d, waiting stopped", w.curCount)
		time.Sleep(1 * time.Second)
	}
}

func NewWorkManger() *SWorkManager {
	return &SWorkManager{}
}
