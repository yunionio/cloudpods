package cronman

import (
	"time"
	"runtime/debug"
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"reflect"
	"runtime"
)

const (
	DEFAULT_CRON_INTERVAL = 60*time.Second // default resolution is 1 monutes
)

type SCronJobManager struct {
	checkInterval time.Duration
	timer *time.Timer
	jobs []SCronJob
}

type SCronJob struct {
	name string
	runInterval time.Duration
	job func(ctx context.Context, userCred mcclient.TokenCredential)
	lastRun time.Time
}

func NewCronJobManager(interval time.Duration) *SCronJobManager {
	if interval == 0 {
		interval = DEFAULT_CRON_INTERVAL
	}
	cron := SCronJobManager{checkInterval: interval, jobs: make([]SCronJob, 0)}
	return &cron
}

func (self *SCronJobManager) Start() {
	if self.timer != nil {
		return
	}
	self.timer = time.AfterFunc(self.checkInterval, self.runCronJobs)
}

func (self *SCronJobManager) Stop() {
	if self.timer != nil {
		self.timer.Stop()
		self.timer = nil
	}
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (self *SCronJobManager) AddJob(name string, interval time.Duration, jobFunc func(ctx context.Context, userCred mcclient.TokenCredential)) {
	// name := getFunctionName(jobFunc)
	log.Debugf("Add cronjob %s", name)
	job := SCronJob{name: name, job: jobFunc, runInterval: interval}
	self.jobs = append(self.jobs, job)
}

func (self *SCronJobManager) runCronJobs() {
	now := time.Now()
	for i := 0; i < len(self.jobs); i += 1 {
		self.jobs[i].run(now)
	}
	self.timer = nil
	self.Start() // schedule next run
}

func (self *SCronJob) run(now time.Time) {
	if self.lastRun.IsZero() || now.Sub(self.lastRun) >= self.runInterval {
		log.Debugf("Run cronjob %s", self.name)
		go runJob(self.name, self.job)
	}
}

func runJob(name string, job func(ctx context.Context, userCred mcclient.TokenCredential)) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("CronJob task %s run error: %s", name, r)
			debug.PrintStack()
		}
	}()

	ctx := context.Background()
	userCred := auth.AdminCredential()
	job(ctx, userCred)
}

