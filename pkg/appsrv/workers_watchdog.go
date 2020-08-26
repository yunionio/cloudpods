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

package appsrv

import (
	"time"

	"yunion.io/x/log"
)

const (
	WATCHDOG_SLEEP_SECONDS = 30
)

var (
	busyWorkers map[*SWorkerManager]int

	exitFlag bool
)

func init() {
	busyWorkers = make(map[*SWorkerManager]int)

	watchdog()
}

func watchdog() {
	do_worker_watchdog()

	time.AfterFunc(time.Second*WATCHDOG_SLEEP_SECONDS, watchdog)
}

func do_worker_watchdog() {
	for _, w := range workerManagers {
		stats := w.getState()
		busy := stats.IsBusy()
		if busy {
			if _, ok := busyWorkers[w]; ok {
				busyWorkers[w] += 1
			} else {
				busyWorkers[w] = 1
			}
		} else {
			if _, ok := busyWorkers[w]; ok {
				delete(busyWorkers, w)
			}
		}
	}
	if len(busyWorkers) > 0 {
		for w, k := range busyWorkers {
			if k > 1 {
				log.Warningf("WorkerManager %s has been busy for %d cycles...", w.name, k)
			}
		}
	} else {
		if exitFlag {
			log.Fatalln("System is idle, no worker is busy, exitFlag is set, to exit ...")
		}
	}
}

func SetExitFlag() {
	exitFlag = true
}
