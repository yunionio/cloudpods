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
	"yunion.io/x/pkg/util/sets"
	u "yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/scheduler/api"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

type ExpireManager struct {
	expireChannel chan *api.ExpireArgs
	stopCh        <-chan struct{}

	mergeLock         *sync.Mutex
	reloadCancelQueue *ReloadCancelQueue
}

func NewExpireManager(stopCh <-chan struct{}) *ExpireManager {
	return &ExpireManager{
		expireChannel:     make(chan *api.ExpireArgs, o.Options.ExpireQueueMaxLength),
		stopCh:            stopCh,
		mergeLock:         new(sync.Mutex),
		reloadCancelQueue: NewReloadCancelQueue(stopCh),
	}
}

func (e *ExpireManager) Add(expireArgs *api.ExpireArgs) {
	e.expireChannel <- expireArgs
}

func (e *ExpireManager) Trigger() {
	e.batchMergeExpire()
}

type expireHost struct {
	Id        string
	SessionId string
}

func newExpireHost(id string, sid string) *expireHost {
	return &expireHost{
		Id:        id,
		SessionId: sid,
	}
}

func (e *ExpireManager) Run() {
	t := time.Tick(u.ToDuration(o.Options.ExpireQueueConsumptionPeriod))
	// Watching the expires.
	for {
		select {
		case <-t:
			e.batchMergeExpire()
		case <-e.stopCh:
			// update all the expire before return
			e.batchMergeExpire()
			close(e.expireChannel)
			e.expireChannel = nil
			log.Errorln("expire manager EXIT!")
			return
		default:
			// if expire number is bigger then 80 then update
			if len(e.expireChannel) >= o.Options.ExpireQueueDealLength {
				e.batchMergeExpire()
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (e *ExpireManager) batchMergeExpire() {
	e.mergeLock.Lock()
	defer e.mergeLock.Unlock()
	expireRequestNumber := len(e.expireChannel)
	// If the expireRequestNumber then return right now.
	if expireRequestNumber <= 0 {
		return
	}
	dirtyHostSets := sets.NewString()
	dirtyBaremetalSets := sets.NewString()
	dirtyHosts := make([]*expireHost, 0)
	dirtyBaremetals := make([]*expireHost, 0)
	// Merge all same host.
	for i := 0; i < expireRequestNumber; i++ {
		expireArgs := <-e.expireChannel
		log.V(4).Infof("Get expireArgs from channel: %#v", expireArgs)
		dirtyHostSets.Insert(expireArgs.DirtyHosts...)
		for _, host := range expireArgs.DirtyHosts {
			dirtyHosts = append(dirtyHosts, newExpireHost(host, expireArgs.SessionId))
		}
		dirtyBaremetalSets.Insert(expireArgs.DirtyBaremetals...)
		for _, baremetal := range expireArgs.DirtyBaremetals {
			dirtyBaremetals = append(dirtyBaremetals, newExpireHost(baremetal, expireArgs.SessionId))
		}
	}
	log.V(4).Infof("batchMergeExpire dirtyHosts: %v, dirtyBaremetals: %v", dirtyHosts, dirtyBaremetals)

	// Use queue to ensure reload completes before cancel
	var hostTask, baremetalTask *ReloadCancelTask

	if len(dirtyHosts) > 0 {
		hostTask = &ReloadCancelTask{
			ResType:     "host",
			HostIds:     dirtyHostSets.List(),
			ExpireHosts: dirtyHosts,
		}
	}

	if len(dirtyBaremetals) > 0 {
		baremetalTask = &ReloadCancelTask{
			ResType:     "baremetal",
			HostIds:     dirtyBaremetalSets.List(),
			ExpireHosts: dirtyBaremetals,
		}
	}

	// Add tasks to queue (will be processed asynchronously)
	if hostTask != nil || baremetalTask != nil {
		tasks := make([]*ReloadCancelTask, 0, 2)
		if hostTask != nil {
			tasks = append(tasks, hostTask)
		}
		if baremetalTask != nil {
			tasks = append(tasks, baremetalTask)
		}
		e.reloadCancelQueue.AddBatch(tasks, nil)
		log.Infof("Added reload+cancel tasks to queue: hosts=%d, baremetals=%d",
			len(dirtyHosts), len(dirtyBaremetals))
	}
}
