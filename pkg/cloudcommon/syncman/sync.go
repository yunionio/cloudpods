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

package syncman

import (
	"fmt"
	"sync/atomic"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
)

type ISyncClient interface {
	DoSync(first bool) (time.Duration, error)
	NeedSync(dat *jsonutils.JSONDict) bool
	Name() string
}

type SSyncManager struct {
	ISyncClient

	lastSync  time.Time
	syncTimer *time.Timer
	syncOnce  int32

	syncWorkerManager *appsrv.SWorkerManager
}

func (manager *SSyncManager) InitSync(client ISyncClient) {
	manager.ISyncClient = client
	manager.syncWorkerManager = appsrv.NewWorkerManagerIgnoreOverflow(fmt.Sprintf("(%s)sync_worker", client.Name()), 1, 1, true, true)
}

func (manager *SSyncManager) syncByInterval() error {
	if manager.syncTimer != nil {
		manager.syncTimer.Stop()
		manager.syncTimer = nil
	}
	isFirst := false
	if manager.lastSync.IsZero() {
		isFirst = true
	}
	next, err := manager.DoSync(isFirst)
	if err == nil {
		manager.lastSync = time.Now()
	}
	manager.syncTimer = time.AfterFunc(next, manager.SyncOnce)
	return err
}

func (manager *SSyncManager) SyncOnce() {
	log.Debugf("[%s] SyncOnce", manager.Name())
	if atomic.CompareAndSwapInt32(&manager.syncOnce, 0, 1) {
		manager.syncWorkerManager.Run(func() {
			atomic.StoreInt32(&manager.syncOnce, 0)
			manager.syncByInterval()
		}, nil, nil)
	}
}

func (manager *SSyncManager) FirstSync() error {
	return manager.syncByInterval()
}
