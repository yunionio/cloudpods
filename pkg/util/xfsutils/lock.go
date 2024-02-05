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

package xfsutils

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
	defer mapLock.Unlock()
	xfsLock, ok := xfsMountUniqueTool[uuid]
	if !ok {
		return
	}

	xfsLock.Unlock()
	delete(xfsMountUniqueTool, uuid)
}

var (
	mapLock            = sync.Mutex{}
	xfsMountUniqueTool = map[string]*sync.Mutex{}
)
