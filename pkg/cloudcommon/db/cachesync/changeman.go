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

package cachesync

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/syncman/watcher"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

type SResourceChangeManager struct {
	watcher.SInformerSyncManager

	resMan          informer.IResourceManager
	intervalSeconds int

	ids     []string
	idsLock *sync.Mutex
}

func newResourceChangeManager(resMan informer.IResourceManager, intvalSecs int) *SResourceChangeManager {
	man := &SResourceChangeManager{
		resMan:          resMan,
		intervalSeconds: intvalSecs,

		ids:     make([]string, 0),
		idsLock: &sync.Mutex{},
	}
	man.InitSync(man)
	man.FirstSync()
	man.StartWatching(resMan)
	return man
}

func (man *SResourceChangeManager) DoSync(first bool, timeout bool) (time.Duration, error) {
	if first || timeout {
		// reset id list
		man.resetId()
	} else {
		log.Debugf("to do incremental sync ids %s", jsonutils.Marshal(man.ids))
	}

	switch man.resMan.KeyString() {
	case identity_modules.Projects.KeywordPlural:
		tenantCacheSyncWorkerMan.Run(&tenantCacheSyncWorker{
			ids: man.ids,
		}, nil, nil)
	case identity_modules.Domains.KeywordPlural:
		tenantCacheSyncWorkerMan.Run(&domainCacheSyncWorker{
			ids: man.ids,
		}, nil, nil)
	case identity_modules.UsersV3.KeywordPlural:
		tenantCacheSyncWorkerMan.Run(&userCacheSyncWorker{
			ids: man.ids,
		}, nil, nil)
	}
	man.resetId()
	log.Debugf("sync DONE, next sync %d seconds later...", man.intervalSeconds*8)
	return time.Second * time.Duration(man.intervalSeconds) * 8, nil
}

func (man *SResourceChangeManager) NeedSync(dat *jsonutils.JSONDict) bool {
	if dat != nil && dat.Contains("id") {
		idstr, _ := dat.GetString("id")
		idstr = strings.TrimSpace(idstr)
		if len(idstr) > 0 {
			man.addId(idstr)
		}
	}
	return true
}

func (man *SResourceChangeManager) addId(idstr string) {
	man.idsLock.Lock()
	defer man.idsLock.Unlock()

	man.ids = append(man.ids, idstr)
}

func (man *SResourceChangeManager) resetId() {
	man.idsLock.Lock()
	defer man.idsLock.Unlock()

	man.ids = man.ids[0:0]
}

func (man *SResourceChangeManager) Name() string {
	return fmt.Sprintf("ResourceChangeManager:%s", man.resMan.GetKeyword())
}
