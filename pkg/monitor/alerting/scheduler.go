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

package alerting

import (
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

type schedulerImpl struct {
	jobs map[string]*Job
}

func newScheduler() scheduler {
	return &schedulerImpl{
		jobs: make(map[string]*Job),
	}
}

func (s *schedulerImpl) Update(rules []*Rule) {
	log.Debugf("Scheduling update, rule count %d", len(rules))

	jobs := make(map[string]*Job)

	for _, rule := range rules {
		var job *Job
		if s.jobs[rule.Id] != nil {
			job = s.jobs[rule.Id]
		} else {
			job = &Job{}
			job.SetRunning(false)
		}

		job.Rule = rule

		//offset := ((rule.Frequency * 1000) / int64(len(rules))) * int64(i)
		//job.Offset = int64(math.Floor(float64(offset) / 1000))

		if job.Offset == 0 {
			// zero offset causes division with 0 panics
			job.Offset = 1
		}
		jobs[rule.Id] = job
	}

	s.jobs = jobs
}

func (s *schedulerImpl) Tick(tickTime time.Time, execQueue chan *Job) {
	now := tickTime.Unix()

	for _, job := range s.jobs {
		if job.GetRunning() || job.Rule.State == monitor.AlertStatePaused {
			continue
		}

		if job.OffsetWait && now%job.Offset == 0 {
			job.OffsetWait = false
			s.enqueue(job, execQueue)
			continue
		}

		// Check the job frequency against the minium interval required
		interval := job.Rule.Frequency
		if interval < options.Options.AlertingMinIntervalSeconds {
			interval = options.Options.AlertingMinIntervalSeconds
		}

		if now%interval == 0 {
			if job.Offset > 0 {
				job.OffsetWait = true
			} else {
				s.enqueue(job, execQueue)
			}
		}
	}
}

func (s *schedulerImpl) enqueue(job *Job, execQueue chan *Job) {
	log.Debugf("Scheduler: putting job into exec queue, name %s:%s", job.Rule.Name, job.Rule.Id)
	execQueue <- job
}
