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

func (e *ExpireManager) Run() {
	t := time.Tick(u.ToDuration(o.GetOptions().ExpireQueueConsumptionPeriod))

	notInSession := func(ids []string, resType string) []string {
		//var newIds []string
		//for _, id := range ids {
		//if !schedManager.ReservedPoolManager.InSession(resType, id) {
		//newIds = append(newIds, id)
		//}
		//}
		//return newIds
		return ids
	}

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
		dirtyHostMap := make(map[string]int, expireRequestNumber)
		dirtyBaremetalMap := make(map[string]int, expireRequestNumber)
		dirtyHosts := make([]string, 0)
		dirtyBaremetals := make([]string, 0)
		// Merge all same host.
		for i := 0; i < expireRequestNumber; i++ {
			expireArgs := <-e.expireChannel
			log.V(4).Infof("Get expireArgs from channel: %#v", expireArgs)
			for _, host := range expireArgs.DirtyHosts {
				if _, ok := dirtyHostMap[host]; !ok {
					dirtyHostMap[host] = len(dirtyHosts)
					dirtyHosts = append(dirtyHosts, host)
				}
			}
			for _, baremetal := range expireArgs.DirtyBaremetals {
				if _, ok := dirtyBaremetalMap[baremetal]; !ok {
					dirtyBaremetalMap[baremetal] = len(dirtyBaremetals)
					dirtyBaremetals = append(dirtyBaremetals, baremetal)
				}
			}
		}
		log.V(4).Infof("batchMergeExpire dirtyHosts: %v, dirtyBaremetals: %v", dirtyHosts, dirtyBaremetals)
		wg := &sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			dirtyHosts = notInSession(dirtyHosts, "host")
			if len(dirtyHosts) > 0 {
				log.V(10).Debugf("CleanDirty Hosts: %v\n", dirtyHosts)
				_, err := schedManager.CandidateManager.Reload("host", dirtyHosts)
				schedManager.CandidateManager.CleanDirtyCandidatesOnce(dirtyHosts)
				if err != nil {
					log.Errorf("%v", err)
				}
			}
		}()

		go func() {
			defer wg.Done()
			dirtyBaremetals = notInSession(dirtyBaremetals, "baremetal")
			if len(dirtyBaremetals) > 0 {
				log.V(10).Debugf("CleanDirty Baremetals: %v\n", dirtyBaremetals)
				_, err := schedManager.CandidateManager.Reload("baremetal", dirtyBaremetals)
				schedManager.CandidateManager.CleanDirtyCandidatesOnce(dirtyBaremetals)
				if err != nil {
					log.Errorf("%v", err)
				}
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
