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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	syncWorker *appsrv.SWorkerManager
)

func InitSyncWorkers() {
	syncWorker = appsrv.NewWorkerManager(
		"identityProviderSyncWorkerManager",
		1,
		2048,
		true,
	)
}

type syncTask struct {
	ctx      context.Context
	userCred mcclient.TokenCredential
	idp      *SIdentityProvider
}

func (t *syncTask) Run() {
	t.idp.SetSyncStatus(t.ctx, t.userCred, api.IdentitySyncStatusSyncing)
	defer t.idp.SetSyncStatus(t.ctx, t.userCred, api.IdentitySyncStatusIdle)

	conf, err := GetConfigs(t.idp, true, nil, nil)
	if err != nil {
		log.Errorf("GetConfig for idp %s fail %s", t.idp.Name, err)
		t.idp.MarkDisconnected(t.ctx, t.userCred, err)
		return
	}
	driver, err := driver.GetDriver(t.idp.Driver, t.idp.Id, t.idp.Name, t.idp.Template, t.idp.TargetDomainId, conf)
	if err != nil {
		log.Errorf("GetDriver for idp %s fail %s", t.idp.Name, err)
		t.idp.MarkDisconnected(t.ctx, t.userCred, err)
		return
	}
	err = driver.Probe(t.ctx)
	if err != nil {
		log.Errorf("Probe for idp %s fail %s", t.idp.Name, err)
		t.idp.MarkDisconnected(t.ctx, t.userCred, err)
		return
	}

	t.idp.MarkConnected(t.ctx, t.userCred)

	err = driver.Sync(t.ctx)
	if err != nil {
		log.Errorf("Sync for idp %s fail %s", t.idp.Name, err)
		return
	}
}

func (t *syncTask) Dump() string {
	return fmt.Sprintf("idp %s", jsonutils.Marshal(t.idp).String())
}

func submitIdpSyncTask(ctx context.Context, userCred mcclient.TokenCredential, idp *SIdentityProvider) {
	idp.SetSyncStatus(ctx, userCred, api.IdentitySyncStatusQueued)
	task := &syncTask{
		ctx:      ctx,
		userCred: userCred,
		idp:      idp,
	}
	syncWorker.Run(task, nil, nil)
}
