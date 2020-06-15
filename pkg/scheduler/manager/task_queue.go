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
	"fmt"
	"strings"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
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

	resultItems *ScheduleResult
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

func (te *TaskExecutor) Execute() {
	te.Status = TaskExecutorStatusRunning
	te.resultItems, te.resultError = te.execute()
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

type ScheduleResult struct {
	// Result is sync schedule result
	Result *schedapi.ScheduleOutput
	// ForecastResult is forecast schedule result
	ForecastResult interface{}
	// TestResult is test schedule result
	TestResult interface{}
}

func (te *TaskExecutor) execute() (*ScheduleResult, error) {
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
	result, err := genericScheduler.Schedule(te.unit, candidates)
	if err != nil {
		return nil, errors.Wrap(err, "genericScheduler.Schedule")
	}
	out := new(ScheduleResult)
	if schedInfo.IsSuggestion {
		if schedInfo.ShowSuggestionDetails && schedInfo.SuggestionAll {
			out.ForecastResult = transToSchedForecastResult(result)
		} else {
			out.TestResult = transToSchedTestResult(result, schedInfo.SuggestionLimit)
		}
	} else {
		out.Result = transToSchedResult(result, schedInfo)
		driver := te.unit.GetHypervisorDriver()
		if err := setSchedPendingUsage(driver, schedInfo, out.Result); err != nil {
			return nil, errors.Wrap(err, "setSchedPendingUsage")
		}
	}
	return out, nil
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

func (te *TaskExecutor) GetResult() (*ScheduleResult, error) {
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

	go func() {
		defer close(teq.queue)

		var taskExecutor *TaskExecutor
		for taskExecutor = <-teq.queue; teq.running; taskExecutor = <-teq.queue {
			if taskExecutor.Status == TaskExecutorStatusWaiting {
				taskExecutor.Execute()
			}
		}
	}()

	go func() {
		<-stopCh
		teq.running = false
		teq.queue <- nil
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
		key               string
		taskExecutorQueue *TaskExecutorQueue
		ok                bool
	)

	if taskExecutorQueue, ok = teqm.taskExecutorMap[key]; !ok {
		taskExecutorQueue = NewTaskExecutorQueue(schedType, teqm.stopCh)
		teqm.taskExecutorMap[key] = taskExecutorQueue
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
	taskExecutors []*TaskExecutor   `json:"-"`
	manager       *SchedulerManager `json:"-"`
	lock          sync.Mutex
	waitCh        chan struct{}

	completedCount int
	resultItems    *ScheduleResult
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

func (t *Task) Wait() (*ScheduleResult, error) {
	log.V(10).Infof("Task wait...")
	<-t.waitCh
	return t.GetResult()
}

func (t *Task) GetResult() (*ScheduleResult, error) {
	return t.resultItems, t.resultError
}
