package guestfs

import (
	"sync"

	"yunion.io/x/log"
)

func LockXfsPartition(uuid string) {
	log.Infof("xfs lock %s", uuid)

	var (
		xfsLock *sync.Mutex
		ok      bool
	)

	mapLock.Lock()
	xfsLock, ok = xfsMountUniqueTool[uuid]
	if !ok {
		xfsLock = new(sync.Mutex)
		xfsMountUniqueTool[uuid] = xfsLock
	}
	mapLock.Unlock()

	xfsLock.Lock()
}

func UnlockXfsPartition(uuid string) {
	log.Infof("xfs unlock %s", uuid)
	mapLock.Lock()
	xfsLock := xfsMountUniqueTool[uuid]
	mapLock.Unlock()

	xfsLock.Unlock()
}

var (
	mapLock            = sync.Mutex{}
	xfsMountUniqueTool = map[string]*sync.Mutex{}
)
