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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/scheduler/api"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

type CompletedManager struct {
	completedChannel chan *api.CompletedNotifyArgs
	stopCh           <-chan struct{}
}

func NewCompletedManager(stopCh <-chan struct{}) *CompletedManager {
	return &CompletedManager{
		completedChannel: make(chan *api.CompletedNotifyArgs, o.Options.CompletedQueueMaxLength),
		stopCh:           stopCh,
	}
}

func (c *CompletedManager) Add(completedNotifyArgs *api.CompletedNotifyArgs) {
	c.completedChannel <- completedNotifyArgs
}

func (c *CompletedManager) Run() {
	t := time.Tick(utils.ToDuration(o.Options.CompletedQueueConsumptionPeriod))

	removeSession := func() {
		//completedNotifyArgs := <-c.completedChannel
		//pool, err := schedManager.ReservedPoolManager.SearchReservedPoolBySessionID(completedNotifyArgs.SessionID)
		//if err != nil {
		//log.Errorln(err)
		//return
		//}

		//sessionItem := pool.GetSessionItem(completedNotifyArgs.SessionID)
		//if sessionItem == nil {
		//log.Errorln(fmt.Errorf("session %v not found\n", completedNotifyArgs.SessionID))
		//return
		//}
		//candidateIds := sessionItem.AllCandidateIDs()

		// load candidates
		//if len(candidateIds) > 0 {
		//schedManager.CandidateManager.Reload(pool.Name, candidateIds)
		//}

		// remove session
		//pool.RemoveSession(completedNotifyArgs.SessionID)
	}

	reloadAndRemoveSessions := func() {
		completedRequestNumber := len(c.completedChannel)
		// If the completedRequestNumber then return right now.
		if completedRequestNumber <= 0 {
			return
		}

		wg := &utils.WaitGroupWrapper{}
		for i := 0; i < completedRequestNumber; i++ {
			wg.Wrap(removeSession)
		}

		if ok := utils.WaitTimeOut(wg, time.Duration(completedRequestNumber)*utils.ToDuration(o.Options.CompletedQueueConsumptionTimeout)); !ok {
			log.Errorln("time out reload data in completed when remove sessions.")
		}
	}

	// Watching the completed sessions.
	for {
		select {
		case <-t:
			reloadAndRemoveSessions()
		case <-c.stopCh:
			// update all the sessions before return.
			reloadAndRemoveSessions()
			close(c.completedChannel)
			c.completedChannel = nil
			log.Errorln("completed manager EXIT!")
			return
		default:
			// if sessions' number is bigger then 10 then reload and remove.
			if len(c.completedChannel) >= o.Options.CompletedQueueDealLength {
				reloadAndRemoveSessions()
			} else {
				time.Sleep(10 * time.Second)
			}
		}
	}
}
