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

func submitIdpSyncTask(ctx context.Context, userCred mcclient.TokenCredential, idp *SIdentityProvider) {
	idp.SetSyncStatus(ctx, userCred, api.IdentitySyncStatusQueued)
	syncWorker.Run(func() {
		idp.SetSyncStatus(ctx, userCred, api.IdentitySyncStatusSyncing)
		defer idp.SetSyncStatus(ctx, userCred, api.IdentitySyncStatusIdle)

		conf, err := GetConfigs(idp, true)
		if err != nil {
			log.Errorf("GetConfig for idp %s fail %s", idp.Name, err)
			idp.MarkDisconnected(ctx, userCred)
			return
		}
		driver, err := driver.GetDriver(idp.Driver, idp.Id, idp.Name, idp.Template, idp.TargetDomainId, idp.AutoCreateProject.Bool(), conf)
		if err != nil {
			log.Errorf("GetDriver for idp %s fail %s", idp.Name, err)
			idp.MarkDisconnected(ctx, userCred)
			return
		}
		err = driver.Probe(ctx)
		if err != nil {
			log.Errorf("Probe for idp %s fail %s", idp.Name, err)
			idp.MarkDisconnected(ctx, userCred)
			return
		}

		idp.MarkConnected(ctx, userCred)

		err = driver.Sync(ctx)
		if err != nil {
			log.Errorf("Sync for idp %s fail %s", idp.Name, err)
			return
		}

	}, nil, nil)
}
