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
	"sync"
	"time"

	"yunion.io/x/log"

	schedmodels "yunion.io/x/onecloud/pkg/scheduler/models"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

// ReloadCancelTask represents a task to reload candidates and then cancel pending usage
type ReloadCancelTask struct {
	ResType     string        // "host" or "baremetal"
	HostIds     []string      // host/baremetal IDs to reload
	ExpireHosts []*expireHost // hosts to cancel pending usage
}

// ReloadCancelQueue manages a queue of reload+cancel tasks
// It ensures that cancel happens only after reload completes
type ReloadCancelQueue struct {
	queue  chan *ReloadCancelTask
	stopCh <-chan struct{}
	wg     sync.WaitGroup
}

// NewReloadCancelQueue creates a new ReloadCancelQueue
func NewReloadCancelQueue(stopCh <-chan struct{}) *ReloadCancelQueue {
	queueSize := o.Options.ExpireQueueMaxLength
	return &ReloadCancelQueue{
		queue:  make(chan *ReloadCancelTask, queueSize),
		stopCh: stopCh,
	}
}

// Add adds a reload+cancel task to the queue
func (q *ReloadCancelQueue) Add(task *ReloadCancelTask) {
	select {
	case q.queue <- task:
		log.Debugf("Added reload+cancel task: resType=%s, hostIds=%v", task.ResType, task.HostIds)
	default:
		log.Warningf("ReloadCancelQueue is full, dropping task: resType=%s", task.ResType)
	}
}

// Run starts the queue worker
func (q *ReloadCancelQueue) Run() {
	defer close(q.queue)

	// Start multiple workers for better throughput
	workerCount := 2
	for i := 0; i < workerCount; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	<-q.stopCh
	log.Infof("ReloadCancelQueue stopping...")

	// Wait for all workers to finish current tasks
	close(q.queue)
	q.wg.Wait()
	log.Infof("ReloadCancelQueue stopped")
}

// worker processes tasks from the queue
func (q *ReloadCancelQueue) worker(id int) {
	defer q.wg.Done()

	log.Infof("ReloadCancelQueue worker %d started", id)
	defer log.Infof("ReloadCancelQueue worker %d stopped", id)

	for task := range q.queue {
		if task == nil {
			continue
		}

		q.processTask(task)
	}
}

// processTask executes reload first, then cancel pending usage
func (q *ReloadCancelQueue) processTask(task *ReloadCancelTask) {
	startTime := time.Now()
	log.Infof("[ReloadCancelQueue] Processing task: resType=%s, hostIds=%v, expireHosts=%d",
		task.ResType, task.HostIds, len(task.ExpireHosts))

	// Step 1: Reload candidates
	if len(task.HostIds) > 0 {
		log.Infof("[ReloadCancelQueue] Step 1: Starting reload for %s, hostIds=%v", task.ResType, task.HostIds)
		// Mark the start of Reload to protect pending usage added during reload
		schedmodels.HostPendingUsageManager.SetReloadStartTime()

		reloadStart := time.Now()
		if _, err := schedManager.CandidateManager.Reload(task.ResType, task.HostIds); err != nil {
			log.Errorf("[ReloadCancelQueue] Failed to reload %s candidates %v: %v", task.ResType, task.HostIds, err)
			// Continue to cancel even if reload fails, as cancel is independent
		} else {
			reloadDuration := time.Since(reloadStart)
			log.Infof("[ReloadCancelQueue] Successfully reloaded %s candidates: %v (duration=%v)",
				task.ResType, task.HostIds, reloadDuration)
			/*
				log.Infof("[ReloadCancelQueue] Step 2: Clearing pending usage for reloaded hosts")
				// Clear pending usage created before Reload started (partial reload)
				// This protects pending usage added during reload
				cutoffTime := schedmodels.HostPendingUsageManager.GetReloadStartTime()
				if cutoffTime.IsZero() {
					log.Warningf("[ReloadCancelQueue] No cutoff time set, skipping pending usage cleanup for hosts %v", task.HostIds)
				} else {
					log.Infof("[ReloadCancelQueue] Clearing pending usage for hosts %v created before %v", task.HostIds, cutoffTime)
					schedmodels.HostPendingUsageManager.GetStore().ClearHostPendingUsageBefore(task.HostIds, cutoffTime)
					log.Infof("[ReloadCancelQueue] Cleared pending usage for hosts %v created before %v", task.HostIds, cutoffTime)
				}
			*/
		}
	}

	// Step 2: Cancel pending usage (always execute, even if reload failed)
	if len(task.ExpireHosts) > 0 {
		log.Infof("[ReloadCancelQueue] Step 3: Canceling pending usage for %d expire hosts", len(task.ExpireHosts))
		schedManager.HistoryManager.CancelCandidatesPendingUsage(task.ExpireHosts)
		log.Infof("[ReloadCancelQueue] Canceled pending usage for %s: %v", task.ResType, task.ExpireHosts)
	}

	duration := time.Since(startTime)
	log.Infof("[ReloadCancelQueue] Completed task: resType=%s, duration=%v", task.ResType, duration)
}

// AddBatch adds multiple tasks in batch
func (q *ReloadCancelQueue) AddBatch(tasks []*ReloadCancelTask, _ []*ReloadCancelTask) {
	// Simply add all tasks to queue
	// Deduplication is not needed here as ExpireManager already merges by host ID
	for _, task := range tasks {
		if task != nil {
			q.Add(task)
		}
	}
}
