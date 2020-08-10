package guestfs

import (
	"sync"

	"yunion.io/x/log"
)

func LockXfsPartition(uuid string) {
	log.Infof("xfs lock %s", uuid)
	if iLock, ok := xfsMountUniqueTool.Load(uuid); ok {
		lock := iLock.(*sync.Mutex)
		lock.Lock()
	} else {
		lock := new(sync.Mutex)
		xfsMountUniqueTool.Store(uuid, lock)
		lock.Lock()
	}
}

func UnlockXfsPartition(uuid string) {
	log.Infof("xfs unlock %s", uuid)
	iLock, _ := xfsMountUniqueTool.Load(uuid)
	lock := iLock.(*sync.Mutex)
	lock.Unlock()
}

var xfsMountUniqueTool = sync.Map{}
