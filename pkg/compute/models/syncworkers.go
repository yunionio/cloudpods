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

package models

import (
	"context"
	"fmt"
	"strconv"

	"github.com/serialx/hashring"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/util/panicutils"
)

var (
	syncAccountWorker *appsrv.SWorkerManager
	syncWorkers       []*appsrv.SWorkerManager
	syncWorkerRing    *hashring.HashRing
)

func InitSyncWorkers(count int) {
	syncWorkers = make([]*appsrv.SWorkerManager, count)
	syncWorkerIndexes := make([]string, count)
	for i := range syncWorkers {
		syncWorkers[i] = appsrv.NewWorkerManager(
			fmt.Sprintf("syncWorkerManager-%d", i+1),
			1,
			2048,
			true,
		)
		syncWorkerIndexes[i] = strconv.Itoa(i)
	}
	syncWorkerRing = hashring.New(syncWorkerIndexes)
	syncAccountWorker = appsrv.NewWorkerManager(
		"cloudAccountProbeWorkerManager",
		1,
		2048,
		true,
	)
}

type resSyncTask struct {
	syncFunc func()
	key      string
}

func (t *resSyncTask) Run() {
	t.syncFunc()
}

func (t *resSyncTask) Dump() string {
	return fmt.Sprintf("key: %s", t.key)
}

func RunSyncCloudproviderRegionTask(ctx context.Context, key string, syncFunc func()) {
	nodeIdxStr, _ := syncWorkerRing.GetNode(key)
	nodeIdx, _ := strconv.Atoi(nodeIdxStr)
	task := resSyncTask{
		syncFunc: syncFunc,
		key:      key,
	}
	log.Debugf("run sync task at %d len %d", nodeIdx, len(syncWorkers))
	syncWorkers[nodeIdx].Run(&task, nil, func(err error) {
		panicutils.SendPanicMessage(ctx, err)
	})
}

func RunSyncCloudAccountTask(ctx context.Context, probeFunc func()) {
	task := resSyncTask{
		syncFunc: probeFunc,
		key:      "AccountProb",
	}
	syncAccountWorker.Run(&task, nil, func(err error) {
		panicutils.SendPanicMessage(ctx, err)
	})
}
