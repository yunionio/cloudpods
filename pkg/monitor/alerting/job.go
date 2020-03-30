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
	"sync"
)

// Job holds state about when the alert rule should be evaluated.
type Job struct {
	Offset      int64
	OffsetWait  bool
	Delay       bool
	running     bool
	Rule        *Rule
	runningLock sync.Mutex
}

// GetRunning returns true if the job is running. A lock is taken and released on the Job to ensure atomicity.
func (j *Job) GetRunning() bool {
	defer j.runningLock.Unlock()
	j.runningLock.Lock()
	return j.running
}

// SetRunning sets the running property on the Job. A lock is taken and released on the Job to ensure atomicity.
func (j *Job) SetRunning(b bool) {
	j.runningLock.Lock()
	j.running = b
	j.runningLock.Unlock()
}
