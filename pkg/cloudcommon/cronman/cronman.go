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
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var (
	DefaultAdminSessionGenerator = auth.AdminCredential
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
	jobs    CronJobTimerHeap
	add     chan *SCronJob
	stop    chan struct{}
	running bool
	workers *appsrv.SWorkerManager
}

func GetCronJobManager(idDbWorker bool) *SCronJobManager {
	if manager == nil {
		manager = &SCronJobManager{
			jobs:    make([]*SCronJob, 0),
			add:     make(chan *SCronJob),
			workers: appsrv.NewWorkerManager("CronJobWorkers", 1, 1024, idDbWorker),
		}
	}

	return manager
}

func (self *SCronJobManager) AddJobAtIntervals(name string, interval time.Duration, jobFunc TCronJobFunction) {
	self.AddJobAtIntervalsWithStartRun(name, interval, jobFunc, false)
}

func (self *SCronJobManager) AddJobAtIntervalsWithStartRun(name string, interval time.Duration, jobFunc TCronJobFunction, startRun bool) {
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
		self.add <- &job
	}
}

func (self *SCronJobManager) AddJobEveryFewDays(name string, day, hour, min, sec int, jobFunc TCronJobFunction, startRun bool) {
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
		self.add <- &job
	}
}

func (self *SCronJobManager) AddJobEveryFewHour(name string, hour, min, sec int, jobFunc TCronJobFunction, startRun bool) {
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
		self.add <- &job
	}
}

func (self *SCronJobManager) Next(now time.Time) {
	for _, job := range self.jobs {
		job.Next = job.Timer.Next(now)
	}
}

func (self *SCronJobManager) Start() {
	if self.running {
		return
	}
	self.running = true
	self.init()
	go self.run()
}

func (self *SCronJobManager) Stop() {
	if self.stop != nil {
		close(self.stop)
	}
}

func (self *SCronJobManager) init() {
	now := time.Now()
	self.Next(now)
	heap.Init(&self.jobs)
	for i := 0; i < len(self.jobs); i += 1 {
		if self.jobs[i].StartRun {
			self.jobs[i].StartRun = false
			self.jobs[i].runJob(true)
		}
	}
}

func (self *SCronJobManager) run() {
	for {
		now := time.Now()
		var timer *time.Timer
		if len(self.jobs) == 0 || self.jobs[0].Next.IsZero() {
			timer = time.NewTimer(100000 * time.Hour)
		} else {
			timer = time.NewTimer(self.jobs[0].Next.Sub(now))
		}
		select {
		case now = <-timer.C:
			self.runJob(now)
		case newJob := <-self.add:
			now = time.Now()
			newJob.Next = newJob.Timer.Next(now)
			if newJob.StartRun {
				newJob.runJob(true)
			}
			heap.Push(&self.jobs, newJob)
		case <-self.stop:
			timer.Stop()
			return
		}
	}
}

func (self *SCronJobManager) runJob(now time.Time) {
	if len(self.jobs) > 0 && !(self.jobs[0].Next.After(now) || self.jobs[0].Next.IsZero()) {
		self.jobs[0].runJob(false)
		self.jobs[0].Next = self.jobs[0].Timer.Next(now)
		heap.Fix(&self.jobs, 0)
		self.runJob(now)
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

	// log.Debugf("Cron job: %s started", job.Name)
	ctx := context.Background()
	ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_APPNAME, "Cron-Service")
	userCred := DefaultAdminSessionGenerator()
	job.job(ctx, userCred, isStart)
}
