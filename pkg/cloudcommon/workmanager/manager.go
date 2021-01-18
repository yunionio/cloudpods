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

package workmanager

import (
	"context"
	"runtime/debug"
	"sync/atomic"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
)

type DelayTaskFunc func(context.Context, interface{}) (jsonutils.JSONObject, error)
type OnTaskFailed func(context.Context, string)
type OnTaskCompleted func(context.Context, jsonutils.JSONObject)

type SWorkManager struct {
	curCount int32

	onFailed    OnTaskFailed
	onCompleted OnTaskCompleted
	worker      *appsrv.SWorkerManager
}

func (w *SWorkManager) add() {
	atomic.AddInt32(&w.curCount, 1)
}

func (w *SWorkManager) done() {
	atomic.AddInt32(&w.curCount, -1)
}

func (w *SWorkManager) DelayTask(ctx context.Context, task DelayTaskFunc, params interface{}) {
	w.delayTask(ctx, task, params, w.worker)
}

func (w *SWorkManager) DelayTaskWithWorker(
	ctx context.Context, task DelayTaskFunc, params interface{}, worker *appsrv.SWorkerManager,
) {
	w.delayTask(ctx, task, params, worker)
}

type workerTask struct {
	ctx    context.Context
	w      *SWorkManager
	task   DelayTaskFunc
	params interface{}
}

func (t *workerTask) Run() {
	defer t.w.done()
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("DelayTask panic: %s", r)
			debug.PrintStack()
			switch val := r.(type) {
			case string:
				t.w.onFailed(t.ctx, val)
			case error:
				t.w.onFailed(t.ctx, val.Error())
			default:
				t.w.onFailed(t.ctx, "Unknown panic")
			}
		}
	}()

	// HACK: callback only
	if t.task == nil {
		t.w.onCompleted(t.ctx, nil)
		return
	}

	res, err := t.task(t.ctx, t.params)
	if err != nil {
		log.Infof("DelayTask failed: %s", err)
		t.w.onFailed(t.ctx, err.Error())
	} else {
		log.Infof("DelayTask complete: %v", res)
		t.w.onCompleted(t.ctx, res)
	}
}

func (t *workerTask) Dump() string {
	return ""
}

// If delay task is not panic and task func return err is nil
// task complete will be called, otherwise called task failed
// Params is interface for receive any type, task func should do type assertion
func (w *SWorkManager) delayTask(ctx context.Context, task DelayTaskFunc, params interface{}, worker *appsrv.SWorkerManager) {
	if ctx == nil || ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID) == nil {
		w.delayTaskWithoutReqctx(ctx, task, params, worker)
		return
	} else {
		// delayTask should have a new context.Context with value 'taskid'
		ctx = context.WithValue(context.Background(), appctx.APP_CONTEXT_KEY_TASK_ID, ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID))
		w.add()
		t := workerTask{
			ctx:    ctx,
			w:      w,
			task:   task,
			params: params,
		}
		worker.Run(&t, nil, nil)
	}
}

func (w *SWorkManager) DelayTaskWithoutReqctx(ctx context.Context, task DelayTaskFunc, params interface{}) {
	w.delayTaskWithoutReqctx(ctx, task, params, w.worker)
}

type delayWorkerTask struct {
	w      *SWorkManager
	task   DelayTaskFunc
	ctx    context.Context
	params interface{}
}

func (t *delayWorkerTask) Run() {
	defer t.w.done()
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("DelayTaskWithoutReqctx panic: ", r)
			debug.PrintStack()
		}
	}()

	if t.task == nil {
		return
	}

	if _, err := t.task(t.ctx, t.params); err != nil {
		log.Errorln("DelayTaskWithoutReqctx error: ", err)
		t.w.onFailed(t.ctx, err.Error())
	}
}

func (t *delayWorkerTask) Dump() string {
	return ""
}

// response task by self, did not callback
func (w *SWorkManager) delayTaskWithoutReqctx(
	ctx context.Context, task DelayTaskFunc, params interface{}, worker *appsrv.SWorkerManager,
) {
	// delayTaskWithoutReqctx should have a new context.Context
	newCtx := context.Background()
	if ctx != nil && ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID) != nil {
		newCtx = context.WithValue(newCtx, appctx.APP_CONTEXT_KEY_TASK_ID, ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID))
	}
	ctx = newCtx
	w.add()
	t := delayWorkerTask{
		w:      w,
		task:   task,
		ctx:    ctx,
		params: params,
	}
	w.worker.Run(&t, nil, nil)
}

func (w *SWorkManager) Stop() {
	log.Infof("WorkManager stop, waitting for workers ...")
	for w.curCount > 0 {
		log.Warningf("Busy workers count %d, waiting stopped", w.curCount)
		time.Sleep(1 * time.Second)
	}
}

func NewWorkManger(onFailed OnTaskFailed, onCompleted OnTaskCompleted, workerCount int) *SWorkManager {
	if workerCount <= 0 {
		workerCount = 1
	}
	return &SWorkManager{
		onFailed:    onFailed,
		onCompleted: onCompleted,
		worker: appsrv.NewWorkerManager(
			"RequestWorker", workerCount, appsrv.DEFAULT_BACKLOG, false),
	}
}
