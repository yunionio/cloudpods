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
}

func NewExpireManager(stopCh <-chan struct{}) *ExpireManager {
	return &ExpireManager{
		expireChannel: make(chan *api.ExpireArgs, o.GetOptions().ExpireQueueMaxLength),
		stopCh:        stopCh,
	}
}

func (e *ExpireManager) Add(expireArgs *api.ExpireArgs) {
	e.expireChannel <- expireArgs
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
	t := time.Tick(u.ToDuration(o.GetOptions().ExpireQueueConsumptionPeriod))

	waitTimeOut := func(wg *sync.WaitGroup, timeout time.Duration) bool {
		ch := make(chan struct{})
		go func() {
			wg.Wait()
			close(ch)
		}()
		select {
		case <-ch:
			return true
		case <-time.After(timeout):
			return false
		}
	}

	batchMergeExpire := func() {
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
		wg := &sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			//dirtyHosts = notInSession(dirtyHosts, "host")
			if len(dirtyHosts) > 0 {
				log.V(10).Debugf("CleanDirty Hosts: %v\n", dirtyHosts)
				if _, err := schedManager.CandidateManager.Reload("host", dirtyHostSets.List()); err != nil {
					log.Errorf("Clean dirty hosts %v: %v", dirtyHosts, err)
				}
				schedManager.HistoryManager.CancelCandidatesPendingUsage(dirtyHosts)
			}
		}()

		go func() {
			defer wg.Done()
			//dirtyBaremetals = notInSession(dirtyBaremetals, "baremetal")
			if len(dirtyBaremetals) > 0 {
				log.V(10).Debugf("CleanDirty Baremetals: %v\n", dirtyBaremetals)
				if _, err := schedManager.CandidateManager.Reload("baremetal", dirtyBaremetalSets.List()); err != nil {
					log.Errorf("Clean dirty baremetals %v: %v", dirtyBaremetals, err)
				}
				schedManager.HistoryManager.CancelCandidatesPendingUsage(dirtyBaremetals)
			}
		}()
		if ok := waitTimeOut(wg, u.ToDuration(o.GetOptions().ExpireQueueConsumptionTimeout)); !ok {
			log.Errorln("time out reload data.")
		}
	}

	// Watching the expires.
	for {
		select {
		case <-t:
			batchMergeExpire()
		case <-e.stopCh:
			// update all the expire before return
			batchMergeExpire()
			close(e.expireChannel)
			e.expireChannel = nil
			log.Errorln("expire manager EXIT!")
			return
		default:
			// if expire number is bigger then 80 then update
			if len(e.expireChannel) >= o.GetOptions().ExpireQueueDealLength {
				batchMergeExpire()
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}
}
