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
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func AutoSyncIdentityProviderTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	idps, err := IdentityProviderManager.FetchEnabledProviders("")
	if err != nil {
		log.Errorf("FetchEnabledProviders fail %s", err)
		return
	}
	if isStart {
		for i := range idps {
			idps[i].SetSyncStatus(ctx, userCred, api.IdentitySyncStatusIdle)
		}
	}
	for i := range idps {
		err = syncIdentityProvider(ctx, userCred, &idps[i])
		if err != nil {
			log.Errorf("Fail to sync identityprovider %s: %s", idps[i].Name, err)
		}
	}
}

func syncIdentityProvider(ctx context.Context, userCred mcclient.TokenCredential, idp *SIdentityProvider) error {
	if idp.SyncStatus != api.IdentitySyncStatusIdle {
		log.Debugf("IDP %s cannot sync in non-idle status", idp.Name)
		return nil
	}

	if !idp.CanSync() {
		log.Debugf("IDP %s cannot sync", idp.Name)
		return nil
	}

	if !idp.NeedSync() {
		log.Debugf("IDP %s no need to sync", idp.Name)
		return nil
	}

	drvCls := driver.GetDriverClass(idp.Driver)
	if drvCls.SyncMethod() == api.IdentityProviderSyncLocal {
		// skip, no need to sync
		log.Debugf("IDP %s is local, no need to sync", idp.Name)
		return nil
	}
	if drvCls.SyncMethod() == api.IdentityProviderSyncOnAuth {
		log.Debugf("IDP %s sync on auth, no need to sync", idp.Name)
		return nil
	}
	submitIdpSyncTask(ctx, userCred, idp)
	return nil
}
