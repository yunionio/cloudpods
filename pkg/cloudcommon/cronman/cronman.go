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

package cronman

import (
	"container/heap"
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/elect"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var (
	DefaultAdminSessionGenerator = auth.AdminCredential
	ErrCronJobNameConflict       = errors.New("Cron job Name Conflict")
)

type TCronJobFunction func(ctx context.Context, userCred mcclient.TokenCredential, isStart bool)

var manager *SCronJobManager

type ICronTimer interface {
	Next(time.Time) time.Time
}

type Timer1 struct {
	dur time.Duration
}

func (t *Timer1) Next(now time.Time) time.Time {
	return now.Add(t.dur)
}

type Timer2 struct {
	day, hour, min, sec int
}

func (t *Timer2) Next(now time.Time) time.Time {
	next := now.Add(time.Hour * time.Duration(t.day) * 24)
	return time.Date(next.Year(), next.Month(), next.Day(), t.hour, t.min, t.sec, 0, next.Location())
}

type TimerHour struct {
	hour, min, sec int
}

func (t *TimerHour) Next(now time.Time) time.Time {
	next := now.Add(time.Hour * time.Duration(t.hour))
	return time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), t.min, t.sec, 0, next.Location())
}

type SCronJob struct {
	Name     string
	job      TCronJobFunction
	Timer    ICronTimer
	Next     time.Time
	StartRun bool
}

type CronJobTimerHeap []*SCronJob

func (c CronJobTimerHeap) String() string {
	var s string
	for i := 0; i < len(c); i++ {
		s += c[i].Name + " : " + c[i].Next.String() + "\n"
	}
	return s
}

func (cjth CronJobTimerHeap) Len() int {
	return len(cjth)
}

func (cjth CronJobTimerHeap) Swap(i, j int) {
	cjth[i], cjth[j] = cjth[j], cjth[i]
}

func (cjth CronJobTimerHeap) Less(i, j int) bool {
	if cjth[i].Next.IsZero() {
		return false
	}
	if cjth[j].Next.IsZero() {
		return true
	}
	return cjth[i].Next.Before(cjth[j].Next)
}

func (cjth *CronJobTimerHeap) Push(x interface{}) {
	*cjth = append(*cjth, x.(*SCronJob))
}

func (cjth *CronJobTimerHeap) Pop() interface{} {
	old := *cjth
	n := old.Len()
	x := old[n-1]
	*cjth = old[0 : n-1]
	return x
}

type SCronJobManager struct {
	jobs     CronJobTimerHeap
	stopFunc context.CancelFunc
	add      chan struct{}
	running  bool
	workers  *appsrv.SWorkerManager
	dataLock *sync.Mutex
}

func InitCronJobManager(isDbWorker bool, workerCount int) *SCronJobManager {
	if manager == nil {
		manager = &SCronJobManager{
			jobs:     make([]*SCronJob, 0),
			workers:  appsrv.NewWorkerManager("CronJobWorkers", workerCount, 1024, isDbWorker),
			dataLock: new(sync.Mutex),
			add:      make(chan struct{}),
		}
	}
	return manager
}

func GetCronJobManager() *SCronJobManager {
	return manager
}

func (self *SCronJobManager) IsNameUnique(name string) bool {
	for i := 0; i < len(self.jobs); i++ {
		if self.jobs[i].Name == name {
			return false
		}
	}
	return true
}

func (self *SCronJobManager) String() string {
	return self.jobs.String()
}

func (self *SCronJobManager) AddJobAtIntervals(name string, interval time.Duration, jobFunc TCronJobFunction) error {
	return self.AddJobAtIntervalsWithStartRun(name, interval, jobFunc, false)
}

func (self *SCronJobManager) AddJobAtIntervalsWithStartRun(name string, interval time.Duration, jobFunc TCronJobFunction, startRun bool) error {
	if interval <= 0 {
		return errors.New("AddJobAtIntervals: interval must > 0")
	}
	self.dataLock.Lock()
	defer self.dataLock.Unlock()

	if !self.IsNameUnique(name) {
		return ErrCronJobNameConflict
	}

	t := Timer1{
		dur: interval,
	}
	job := SCronJob{
		Name:     name,
		job:      jobFunc,
		Timer:    &t,
		StartRun: startRun,
	}
	if !self.running {
		self.jobs = append(self.jobs, &job)
	} else {
		self.addJob(&job)
	}
	return nil
}

func (self *SCronJobManager) AddJobEveryFewDays(name string, day, hour, min, sec int, jobFunc TCronJobFunction, startRun bool) error {
	switch {
	case day <= 0:
		return errors.New("AddJobEveryFewDays: day must > 0")
	case hour < 0:
		return errors.New("AddJobEveryFewDays: hour must > 0")
	case min < 0:
		return errors.New("AddJobEveryFewDays: min must > 0")
	case sec < 0:
		return errors.New("AddJobEveryFewDays: sec must > 0")
	}

	self.dataLock.Lock()
	defer self.dataLock.Unlock()

	if !self.IsNameUnique(name) {
		return ErrCronJobNameConflict
	}

	t := Timer2{
		day:  day,
		hour: hour,
		min:  min,
		sec:  sec,
	}
	job := SCronJob{
		Name:     name,
		job:      jobFunc,
		Timer:    &t,
		StartRun: startRun,
	}
	if !self.running {
		self.jobs = append(self.jobs, &job)
	} else {
		self.addJob(&job)
	}
	return nil
}

func (self *SCronJobManager) AddJobEveryFewHour(name string, hour, min, sec int, jobFunc TCronJobFunction, startRun bool) error {
	switch {
	case hour <= 0:
		return errors.New("AddJobEveryFewHour: hour must > 0")
	case min < 0:
		return errors.New("AddJobEveryFewHour: min must > 0")
	case sec < 0:
		return errors.New("AddJobEveryFewHour: sec must > 0")
	}

	self.dataLock.Lock()
	defer self.dataLock.Unlock()

	if !self.IsNameUnique(name) {
		return ErrCronJobNameConflict
	}

	t := TimerHour{
		hour: hour,
		min:  min,
		sec:  sec,
	}
	job := SCronJob{
		Name:     name,
		job:      jobFunc,
		Timer:    &t,
		StartRun: startRun,
	}
	if !self.running {
		self.jobs = append(self.jobs, &job)
	} else {
		self.addJob(&job)
	}
	return nil
}

func (self *SCronJobManager) addJob(newJob *SCronJob) {
	now := time.Now()
	newJob.Next = newJob.Timer.Next(now)
	if newJob.StartRun {
		newJob.runJob(true)
	}
	heap.Push(&self.jobs, newJob)
	go func() { self.add <- struct{}{} }()
}

func (self *SCronJobManager) Remove(name string) error {
	self.dataLock.Lock()
	defer self.dataLock.Unlock()

	var jobIndex = -1
	for i := 0; i < len(self.jobs); i++ {
		if self.jobs[i].Name == name {
			jobIndex = i
			break
		}
	}
	if jobIndex == -1 {
		return errors.Errorf("job %s not found", name)
	}
	heap.Remove(&self.jobs, jobIndex)
	return nil
}

func (self *SCronJobManager) next(now time.Time) {
	for _, job := range self.jobs {
		job.Next = job.Timer.Next(now)
	}
}

func (self *SCronJobManager) Start2(ctx context.Context, electObj *elect.Elect) {
	ctx, self.stopFunc = context.WithCancel(ctx)
	if electObj == nil {
		self.start(ctx)
		return
	}
	electObj.SubscribeWithAction(ctx, func() { self.start(ctx) }, self.Stop)
}

func (self *SCronJobManager) Start() {
	ctx := context.Background()
	ctx, self.stopFunc = context.WithCancel(ctx)
	self.start(ctx)
}

func (self *SCronJobManager) start(ctx context.Context) {
	if self.running {
		return
	}
	self.dataLock.Lock()
	defer self.dataLock.Unlock()
	self.running = true
	self.init()
	go self.run(ctx)
}

func (self *SCronJobManager) Stop() {
	self.stopFunc()
}

func (self *SCronJobManager) init() {
	now := time.Now()
	self.next(now)
	heap.Init(&self.jobs)
	for i := 0; i < len(self.jobs); i += 1 {
		if self.jobs[i].StartRun {
			self.jobs[i].StartRun = false
			self.jobs[i].runJob(true)
		}
	}
}

func (self *SCronJobManager) run(ctx context.Context) {
	var timer *time.Timer
	var now = time.Now()
	for {
		self.dataLock.Lock()
		if len(self.jobs) == 0 || self.jobs[0].Next.IsZero() {
			timer = time.NewTimer(100000 * time.Hour)
		} else {
			timer = time.NewTimer(self.jobs[0].Next.Sub(now))
		}
		self.dataLock.Unlock()
		select {
		case now = <-timer.C:
			self.runJobs(now)
		case <-self.add:
			continue
		case <-ctx.Done():
			timer.Stop()
			self.running = false
			return
		}
	}
}

func (self *SCronJobManager) runJobs(now time.Time) {
	self.dataLock.Lock()
	defer self.dataLock.Unlock()
	for i := 0; i < len(self.jobs); i++ {
		if !(self.jobs[i].Next.After(now) || self.jobs[i].Next.IsZero()) {
			self.jobs[i].runJob(false)
			self.jobs[i].Next = self.jobs[i].Timer.Next(now)
			heap.Fix(&self.jobs, i)
		}
	}
}

func (job *SCronJob) runJob(isStart bool) {
	manager.workers.Run(func() {
		job.runJobInWorker(isStart)
	}, nil, nil)
}

func (job *SCronJob) runJobInWorker(isStart bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("CronJob task %s run error: %s", job.Name, r)
			debug.PrintStack()
		}
	}()

	log.Debugf("Cron job: %s started", job.Name)
	ctx := context.Background()
	ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_APPNAME, "Cron-Service")
	userCred := DefaultAdminSessionGenerator()
	job.job(ctx, userCred, isStart)
}
