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

// If delay task is not panic and task func return err is nil
// task complete will be called, otherwise called task failed
// Params is interface for receive any type, task func should do type assert
func (w *SWorkManager) DelayTask(ctx context.Context, task DelayTaskFunc, params interface{}) {
	if ctx == nil || ctx.Value(APP_CONTEXT_KEY_TASK_ID) == nil {
		w.DelayTaskWithoutTaskid(task, params)
		return
	}

	w.add()
	go func() {
		defer w.done()
		defer func() {
			if r := recover(); r != nil {
				log.Errorln("DelayTask panic: ", r)
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
			log.Debugf("DelayTask failed: %s", err)
			httpclients.TaskFailed(ctx, err.Error())
		} else {
			log.Debugf("DelayTask complete: %v", res)
			httpclients.TaskComplete(ctx, res)
		}
	}()
}

func StartWorker()

func (w *SWorkManager) DelayTaskWithoutTaskid(task DelayTaskFunc, params interface{}) {
	w.add()
	go func() {
		defer w.done()
		defer func() {
			if r := recover(); r != nil {
				log.Errorln("DelayTaskWithoutTaskid panic: ", r)
			}
		}()
		if _, err := task(ctx, params); err != nil {
			log.Errorln("DelayTaskWithoutTaskid", err)
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
