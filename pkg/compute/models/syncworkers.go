package models

import (
	"fmt"
	"strconv"

	"github.com/serialx/hashring"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
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

func RunSyncCloudproviderRegionTask(key string, syncFunc func()) {
	nodeIdxStr, _ := syncWorkerRing.GetNode(key)
	nodeIdx, _ := strconv.Atoi(nodeIdxStr)
	log.Debugf("run sync task at %d len %d", nodeIdx, len(syncWorkers))
	syncWorkers[nodeIdx].Run(syncFunc, nil, nil)
}

func RunSyncCloudAccountTask(probeFunc func()) {
	syncAccountWorker.Run(probeFunc, nil, nil)
}
