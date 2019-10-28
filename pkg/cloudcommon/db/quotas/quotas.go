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

package quotas

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	METADATA_KEY = "quota"
)

type IQuota interface {
	FetchSystemQuota(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider)
	FetchUsage(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string) error
	Update(quota IQuota)
	Add(quota IQuota)
	Sub(quota IQuota)
	Exceed(request IQuota, quota IQuota) error
	IsEmpty() bool
	ToJSON(prefix string) jsonutils.JSONObject
}

/*type SQuotaManager struct {
	keyword        string
	quotaType      reflect.Type
	persistenStore IQuotaStore
	pendingStore   IQuotaStore
}*/

func (manager *SQuotaBaseManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (manager *SQuotaBaseManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchProjectInfo(ctx, data)
}

func (manager *SQuotaBaseManager) newQuota() IQuota {
	model, _ := db.NewModelObject(manager)
	return model.(IQuota)
}

func (manager *SQuotaBaseManager) CancelPendingUsage(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, localUsage IQuota, cancelUsage IQuota) error {
	lockman.LockClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))
	defer lockman.ReleaseClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))

	return manager._cancelPendingUsage(ctx, userCred, scope, ownerId, nil, localUsage, cancelUsage)
}

func (manager *SQuotaBaseManager) _cancelPendingUsage(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, localUsage IQuota, cancelUsage IQuota) error {

	quota := manager.newQuota()
	err := manager.pendingStore.GetQuota(ctx, scope, ownerId, platform, quota)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	quota.Sub(cancelUsage)
	err = manager.pendingStore.SetQuota(ctx, userCred, scope, ownerId, platform, quota)
	if err != nil {
		log.Errorf("%s", err)
	}
	if localUsage != nil {
		localUsage.Sub(cancelUsage)
	}

	// update usage
	manager.PostUsageJob(scope, ownerId, platform, nil, false, false)
	return err
}

func (manager *SQuotaBaseManager) GetPendingUsage(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	return manager.pendingStore.GetQuota(ctx, scope, ownerId, nil, quota)
}

func (manager *SQuotaBaseManager) GetQuota(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	err := manager.getQuotaInternal(ctx, scope, ownerId, nil, quota)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	/*if quota.IsEmpty() && manager.autoCreate {
		quota.FetchSystemQuota()
	}*/
	return nil
}

func (manager *SQuotaBaseManager) SetQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	lockman.LockClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))
	defer lockman.ReleaseClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))

	return manager.setQuotaInternal(ctx, userCred, scope, ownerId, nil, quota)
}

func (manager *SQuotaBaseManager) DeleteQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string) error {
	lockman.LockClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))
	defer lockman.ReleaseClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))

	return manager.deleteQuotaInternal(ctx, userCred, scope, ownerId, nil)
}

func (manager *SQuotaBaseManager) CheckQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, request IQuota) (IQuota, error) {
	lockman.LockClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))
	defer lockman.ReleaseClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))

	return manager._checkQuota(ctx, userCred, scope, ownerId, nil, request)
}

func (manager *SQuotaBaseManager) _checkQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, request IQuota) (IQuota, error) {
	stored := manager.newQuota()
	err := manager.GetQuota(ctx, scope, ownerId, platform, stored)
	if err != nil {
		log.Errorf("fail to get quota %s", err)
		return nil, err
	}
	used := manager.newQuota()
	err = used.FetchUsage(ctx, scope, ownerId, platform)
	if err != nil {
		log.Errorf("fail to get quota usage %s", err)
		return nil, err
	}

	pending := manager.newQuota()
	err = manager.GetPendingUsage(ctx, scope, ownerId, platform, pending)
	if err != nil {
		log.Errorf("fail to get pending usage %s", err)
		return nil, err
	}

	used.Add(pending)
	used.Add(request)

	err = used.Exceed(request, stored)
	if err != nil {
		return nil, err
	}

	return used, nil
}

func (manager *SQuotaBaseManager) CheckSetPendingQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	lockman.LockClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))
	defer lockman.ReleaseClass(ctx, manager, mcclient.OwnerIdString(ownerId, scope))

	return manager._checkSetPendingQuota(ctx, userCred, scope, ownerId, nil, quota)
}

func (manager *SQuotaBaseManager) _checkSetPendingQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	_, err := manager._checkQuota(ctx, userCred, scope, ownerId, platform, quota)
	if err != nil {
		return err
	}
	pending := manager.newQuota()
	err = manager.pendingStore.GetQuota(ctx, scope, ownerId, platform, pending)
	if err != nil {
		log.Errorf("GetQuota fail %s", err)
		return err
	}
	pending.Add(quota)
	return manager.pendingStore.SetQuota(ctx, userCred, scope, ownerId, platform, pending)
}
