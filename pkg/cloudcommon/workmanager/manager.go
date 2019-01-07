package workmanager

import (
	"context"
	"sync/atomic"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
)

type DelayTaskFunc func(context.Context, interface{}) (jsonutils.JSONObject, error)
type OnTaskFailed func(context.Context, string)
type OnTaskCompleted func(context.Context, jsonutils.JSONObject)

type SWorkManager struct {
	curCount int32

	onFailed    OnTaskFailed
	onCompleted OnTaskCompleted
}

func (w *SWorkManager) add() {
	atomic.AddInt32(&w.curCount, 1)
}

func (w *SWorkManager) done() {
	atomic.AddInt32(&w.curCount, -1)
}

// If delay task is not panic and task func return err is nil
// task complete will be called, otherwise called task failed
// Params is interface for receive any type, task func should do type assertion
func (w *SWorkManager) DelayTask(ctx context.Context, task DelayTaskFunc, params interface{}) {
	if ctx == nil || ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID) == nil {
		w.DelayTaskWithoutReqctx(ctx, task, params)
		return
	} else {
		w.add()
		go func() {
			defer w.done()
			defer func() {
				if r := recover(); r != nil {
					log.Errorln("DelayTask panic: ", r)
					switch val := r.(type) {
					case string:
						w.onFailed(ctx, val)
					case error:
						w.onFailed(ctx, val.Error())
					default:
						w.onFailed(ctx, "Unknown panic")
					}
				}
			}()

			res, err := task(ctx, params)
			if err != nil {
				log.Debugf("DelayTask failed: %s", err)
				w.onFailed(ctx, err.Error())
			} else {
				log.Debugf("DelayTask complete: %v", res)
				w.onCompleted(ctx, res)
			}
		}()
	}
}

// response task by self, did not depend on work manager
func (w *SWorkManager) DelayTaskWithoutReqctx(ctx context.Context, task DelayTaskFunc, params interface{}) {
	w.add()
	go func() {
		defer w.done()
		defer func() {
			if r := recover(); r != nil {
				log.Errorln("DelayTaskWithoutReqctx panic: ", r)
			}
		}()
		if _, err := task(ctx, params); err != nil {
			log.Errorln("DelayTaskWithoutReqctx error: ", err)
		}
	}()
}

func (w *SWorkManager) Stop() {
	log.Infof("WorkManager stop, waitting for workers ...")
	for w.curCount > 0 {
		log.Warningf("Busy workers count %d, waiting stopped", w.curCount)
		time.Sleep(1 * time.Second)
	}
}

func NewWorkManger(onFailed OnTaskFailed, onCompleted OnTaskCompleted) *SWorkManager {
	return &SWorkManager{
		onFailed:    onFailed,
		onCompleted: onCompleted,
	}
}
