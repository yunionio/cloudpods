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
	"fmt"

	"github.com/serialx/hashring"

	"yunion.io/x/pkg/util/stringutils"
)

type SHashedWorkerManager struct {
	workers    []*SWorkerManager
	workerRing *hashring.HashRing
	indexMap   map[string]int
}

func NewHashWorkerManager(name string, workerCount int, subWorkerCnt int, backlog int, dbWorker bool) *SHashedWorkerManager {
	workers := make([]*SWorkerManager, workerCount)
	syncWorkerIndexes := make([]string, workerCount)
	indexMap := map[string]int{}
	for i := range workers {
		workers[i] = NewWorkerManager(
			fmt.Sprintf("%s-%d", name, i+1),
			subWorkerCnt,
			backlog,
			dbWorker,
		)
		syncWorkerIndexes[i] = stringutils.UUID4()
		indexMap[syncWorkerIndexes[i]] = i
	}
	workerRing := hashring.New(syncWorkerIndexes)
	return &SHashedWorkerManager{
		workers:    workers,
		workerRing: workerRing,
		indexMap:   indexMap,
	}
}

func (man *SHashedWorkerManager) GetWorkerManager(key string) *SWorkerManager {
	nodeIdxStr, _ := man.workerRing.GetNode(key)
	return man.workers[man.indexMap[nodeIdxStr]]
}
