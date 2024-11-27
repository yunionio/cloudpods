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

package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	schedmodels "yunion.io/x/onecloud/pkg/scheduler/models"
)

const (
	TaskExecutorStatusWaiting string = "waiting"
	TaskExecutorStatusRunning string = "running"
	TaskExecutorStatusFailed  string = "failed"
	TaskExecutorStatusKilled  string = "killed"
	TaskExecutorStatusSuccess string = "success"
)

type TaskExecuteCallback func(task *TaskExecutor)

type TaskExecutor struct {
	Tag       string
	Status    string
	Time      time.Time
	Consuming time.Duration
	scheduler Scheduler
	callback  TaskExecuteCallback
	unit      *core.Unit

	resultItems *core.ScheduleResult
	resultError error
	logs        []string
	capacityMap interface{}
	completed   bool
}

func NewTaskExecutor(scheduler Scheduler, taskExecuteCallback TaskExecuteCallback) *TaskExecutor {
	return &TaskExecutor{
		Tag:       scheduler.SchedData().Tag,
		Status:    TaskExecutorStatusWaiting,
		Time:      time.Now(),
		scheduler: scheduler,
		callback:  taskExecuteCallback,
		completed: false,
	}
}

func (te *TaskExecutor) Execute(ctx context.Context) {
	te.Status = TaskExecutorStatusRunning
	te.resultItems, te.resultError = te.execute(ctx)
	te.completed = true

	if te.resultError != nil {
		te.Status = TaskExecutorStatusFailed
	} else {
		te.Status = TaskExecutorStatusSuccess
	}

	if te.callback != nil {
		te.callback(te)
	}
}

// do execute schedule()
func (te *TaskExecutor) execute(ctx context.Context) (*core.ScheduleResult, error) {
	scheduler := te.scheduler
	genericScheduler, err := core.NewGenericScheduler(scheduler.(core.Scheduler))
	if err != nil {
		return nil, err
	}

	// Get current resources from DB.
	candidates, err := scheduler.Candidates()
	if err != nil {
		return nil, err
	}

	te.unit = scheduler.Unit()
	schedInfo := te.unit.SchedInfo
	// generate result helper
	helper := GenerateResultHelper(schedInfo)
	result, err := genericScheduler.Schedule(ctx, te.unit, candidates, helper)
	if err != nil {
		return nil, errors.Wrap(err, "genericScheduler.Schedule")
	}
	if schedInfo.IsSuggestion {
		return result, nil
	}
	driver := te.unit.GetHypervisorDriver()

	// set sched pending usage
	if err := setSchedPendingUsage(driver, schedInfo, result.Result); err != nil {
		return nil, errors.Wrap(err, "setSchedPendingUsage")
	}
	return result, nil
}

func GenerateResultHelper(schedInfo *api.SchedInfo) core.IResultHelper {
	if !schedInfo.IsSuggestion {
		return core.SResultHelperFunc(core.ResultHelp)
	}
	if schedInfo.ShowSuggestionDetails && schedInfo.SuggestionAll {
		return core.SResultHelperFunc(core.ResultHelpForForcast)
	}
	return core.SResultHelperFunc(core.ResultHelpForTest)
}

func setSchedPendingUsage(driver computemodels.IGuestDriver, req *api.SchedInfo, resp *schedapi.ScheduleOutput) error {
	if req.IsSuggestion || IsDriverSkipScheduleDirtyMark(driver) {
		return nil
	}
	for i, item := range resp.Candidates {
		if item.Error != "" {
			// schedule failed skip add pending usage
			continue
		}
		var guestId string
		if len(req.GuestIds) > i {
			guestId = req.GuestIds[i]
		}
		schedmodels.HostPendingUsageManager.AddPendingUsage(guestId, req, item)
	}
	return nil
}

func IsDriverSkipScheduleDirtyMark(driver computemodels.IGuestDriver) bool {
	return driver == nil || !(driver.DoScheduleCPUFilter() && driver.DoScheduleMemoryFilter() && driver.DoScheduleStorageFilter())
}

func (te *TaskExecutor) cleanup() {
	te.unit = nil
	te.scheduler = nil
	te.callback = nil
}

func (te *TaskExecutor) Kill() {
	if te.Status == TaskExecutorStatusWaiting {
		te.Status = TaskExecutorStatusKilled
	}
}

func (te *TaskExecutor) GetResult() (*core.ScheduleResult, error) {
	return te.resultItems, te.resultError
}

func (te *TaskExecutor) GetLogs() []string {
	return te.logs
}

func (te *TaskExecutor) GetCapacityMap() interface{} {
	return te.capacityMap
}

type TaskExecutorQueue struct {
	schedType string
	queue     chan *TaskExecutor
	running   bool
}

func (teq *TaskExecutorQueue) AddTaskExecutor(scheduler Scheduler,
	callback TaskExecuteCallback) *TaskExecutor {
	taskExecutor := NewTaskExecutor(scheduler, callback)
	teq.queue <- taskExecutor
	return taskExecutor
}

func NewTaskExecutorQueue(schedType string, stopCh <-chan struct{}) *TaskExecutorQueue {
	teq := &TaskExecutorQueue{
		schedType: schedType,
		running:   false,
	}
	teq.Start(stopCh)
	return teq
}

func (teq *TaskExecutorQueue) Start(stopCh <-chan struct{}) {
	if teq.running {
		return
	}
	teq.running = true
	teq.queue = make(chan *TaskExecutor, 5000)

	ctx, ctxCancel := context.WithCancel(context.Background())

	go func() {
		defer close(teq.queue)

		var taskExecutor *TaskExecutor
		for taskExecutor = <-teq.queue; teq.running; taskExecutor = <-teq.queue {
			if taskExecutor.Status == TaskExecutorStatusWaiting {
				taskExecutor.Execute(ctx)
			}
		}
	}()

	go func() {
		<-stopCh
		teq.running = false
		teq.queue <- nil
		ctxCancel()
	}()
}

type TaskExecutorQueueManager struct {
	taskExecutorMap map[string]*TaskExecutorQueue
	lock            sync.Mutex
	stopCh          <-chan struct{}
}

func NewTaskExecutorQueueManager(stopCh <-chan struct{}) *TaskExecutorQueueManager {
	return &TaskExecutorQueueManager{
		taskExecutorMap: make(map[string]*TaskExecutorQueue),
		lock:            sync.Mutex{},
		stopCh:          stopCh,
	}
}

func (teqm *TaskExecutorQueueManager) GetQueue(schedType string) *TaskExecutorQueue {
	teqm.lock.Lock()
	defer teqm.lock.Unlock()

	var (
		taskExecutorQueue *TaskExecutorQueue
		ok                bool
	)

	if taskExecutorQueue, ok = teqm.taskExecutorMap[schedType]; !ok {
		taskExecutorQueue = NewTaskExecutorQueue(schedType, teqm.stopCh)
		teqm.taskExecutorMap[schedType] = taskExecutorQueue
	}

	return taskExecutorQueue
}

func (teqm *TaskExecutorQueueManager) AddTaskExecutor(
	scheduler Scheduler, callback TaskExecuteCallback) *TaskExecutor {
	schedData := scheduler.SchedData()
	taskQueue := teqm.GetQueue(schedData.Hypervisor)
	return taskQueue.AddTaskExecutor(scheduler, callback)
}

type TaskManager struct {
	taskExecutorQueueManager *TaskExecutorQueueManager
	stopCh                   <-chan struct{}
	lock                     sync.Mutex
}

func NewTaskManager(stopCh <-chan struct{}) *TaskManager {
	return &TaskManager{
		taskExecutorQueueManager: NewTaskExecutorQueueManager(stopCh),
		stopCh:                   stopCh,
		lock:                     sync.Mutex{},
	}
}

func (tm *TaskManager) Run() {
	// Do nothing
}

// AddTask provides an interface to increase the scheduling task,
// it will be a scheduling request by the host specification type
// split into multiple scheduling tasks, added to the scheduling
// task manager.
func (tm *TaskManager) AddTask(schedulerManager *SchedulerManager, schedInfo *api.SchedInfo) (*Task, error) {
	var (
		scheduler Scheduler
		err       error
	)

	task := NewTask(schedulerManager, schedInfo)
	// Split into multiple scheduling tasks by host specification type.
	if schedInfo.Hypervisor == api.SchedTypeBaremetal {
		scheduler, err = newBaremetalScheduler(schedulerManager, schedInfo)
	} else {
		scheduler, err = newGuestScheduler(schedulerManager, schedInfo)
	}
	if err != nil {
		return nil, err
	}

	taskExecutorCallback := func(taskExecutor *TaskExecutor) {
		taskExecutor.Consuming = time.Since(taskExecutor.Time)
		task.onTaskCompleted(taskExecutor)
	}

	tm.lock.Lock()
	defer tm.lock.Unlock()

	taskExecutor := tm.taskExecutorQueueManager.AddTaskExecutor(scheduler, taskExecutorCallback)
	task.taskExecutors = append(task.taskExecutors, taskExecutor)

	return task, nil
}

type Task struct {
	Time          time.Time
	SchedInfo     *api.SchedInfo
	Consuming     time.Duration
	taskExecutors []*TaskExecutor
	manager       *SchedulerManager
	lock          sync.Mutex
	waitCh        chan struct{}

	completedCount int
	resultItems    *core.ScheduleResult
	resultError    error
}

func NewTask(manager *SchedulerManager, schedInfo *api.SchedInfo) *Task {
	return &Task{
		Time:          time.Now(),
		SchedInfo:     schedInfo,
		manager:       manager,
		taskExecutors: []*TaskExecutor{},
		lock:          sync.Mutex{},
		waitCh:        make(chan struct{}),
		resultError:   nil,
	}
}

func (t *Task) GetTaskExecutor(tag string) *TaskExecutor {
	for _, executor := range t.taskExecutors {
		if executor.Tag == tag {
			return executor
		}
	}

	return nil
}

func (t *Task) GetSessionID() string {
	return t.SchedInfo.SessionId
}

func (t *Task) GetStatus() string {
	statusMap := make(map[string]int)
	for _, executor := range t.taskExecutors {
		if count, ok := statusMap[executor.Status]; ok {
			statusMap[executor.Status] = count + 1
		} else {
			statusMap[executor.Status] = 1
		}
	}

	ss := []string{}
	for status, count := range statusMap {
		ss = append(ss, fmt.Sprintf("%v %v", count, status))
	}

	return strings.Join(ss, ", ")
}

func (t *Task) onTaskCompleted(taskExecutor *TaskExecutor) {
	t.lock.Lock()
	defer t.lock.Unlock()

	log.V(10).Infof("onTaskCompleted executor: %#v", taskExecutor)
	if taskExecutor.resultError != nil {
		t.resultError = taskExecutor.resultError
		t.onError()
	} else {
		t.resultItems = taskExecutor.resultItems
		t.completedCount += 1
		if t.completedCount >= len(t.taskExecutors) {
			t.onCompleted()
		}
	}

	go func() {
		t.readLog(taskExecutor)
		taskExecutor.cleanup()
	}()
}

func (t *Task) readLog(taskExecutor *TaskExecutor) {
	u := taskExecutor.unit
	if u != nil {
		logs := u.LogManager.Read()
		taskExecutor.logs = logs
		taskExecutor.capacityMap = u.CapacityMap
	}
}

func (t *Task) onError() {
	for _, taskExecutor := range t.taskExecutors {
		taskExecutor.Kill()
	}

	log.Errorf("Remove Session on error: %v", t.SchedInfo.SessionId)

	close(t.waitCh)
}

func (t *Task) onCompleted() {
	t.Consuming = time.Since(t.Time)
	close(t.waitCh)
}

func (t *Task) Wait() (*core.ScheduleResult, error) {
	log.V(10).Infof("Task wait...")
	<-t.waitCh
	return t.GetResult()
}

func (t *Task) GetResult() (*core.ScheduleResult, error) {
	return t.resultItems, t.resultError
}
