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
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

type lbbJanitor struct {
	once   *sync.Once
	notify chan struct{}
	ticker *time.Ticker
}

func newLbbJanitor() *lbbJanitor {
	j := &lbbJanitor{
		once:   &sync.Once{},
		notify: make(chan struct{}, 1),
	}
	return j
}

func (j *lbbJanitor) Start(ctx context.Context) {
	j.once.Do(func() {
		j.start(ctx)
	})
}

func (j *lbbJanitor) start(ctx context.Context) {
	j.ticker = time.NewTicker(24 * time.Hour)
	for {
		select {
		case <-j.ticker.C:
			j.Signal()
			continue
		case _, ok := <-j.notify:
			if !ok {
				log.Errorf("lbb janitor bye: stopped")
				return
			}
		case <-ctx.Done():
			log.Errorf("lbb janitor bye: %v", ctx.Err())
			return
		}
		guestTbl := GuestManager.TableSpec().Instance()
		q := LoadbalancerBackendManager.Query().
			IsFalse("pending_deleted").
			Equals("backend_type", api.LB_BACKEND_GUEST)
		q = q.Join(guestTbl, sqlchemy.AND(
			sqlchemy.IsTrue(guestTbl.Field("deleted")),
			sqlchemy.Equals(guestTbl.Field("id"), q.Field("backend_id")),
		))
		lbbs := []SLoadbalancerBackend{}
		if err := db.FetchModelObjects(LoadbalancerBackendManager, q, &lbbs); err != nil {
			log.Errorf("lbb janitor db fetch: %v", err)
			continue
		}
		adminCred := auth.AdminCredential()
		if adminCred == nil {
			log.Errorf("lbb janitor: get admin credential failed")
			continue
		}
		for i := range lbbs {
			lbb := &lbbs[i]
			log.Warningf("delete lbb %s(%s): backend guest %s was deleted", lbb.Name, lbb.Id, lbb.BackendId)
			lbb.DoPendingDelete(ctx, adminCred)
		}
	}
}

func (j *lbbJanitor) Signal() {
	select {
	case j.notify <- struct{}{}:
	default:
	}
}

func (j *lbbJanitor) Stop() {
	close(j.notify)
	if j.ticker != nil {
		j.ticker.Stop()
	}
}

var theLbbJanitor = newLbbJanitor()

func (man *SLoadbalancerBackendManager) initializeJanitor() {
	go theLbbJanitor.Start(context.Background())
	eh := atexit.ExitHandler{
		Prio: atexit.PRIO_LOG_OTHER,
		Func: func(atexit.ExitHandler) {
			theLbbJanitor.Stop()
		},
		Reason: "loadbalancer backend stales janitor",
	}
	atexit.Register(eh)
}
