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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

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

func (manager *SQuotaBaseManager) CancelPendingUsage(ctx context.Context, userCred mcclient.TokenCredential, localUsage IQuota, cancelUsage IQuota) error {
	LockQuota(ctx, localUsage)
	defer ReleaseQuota(ctx, localUsage)

	return manager._cancelPendingUsage(ctx, userCred, localUsage, cancelUsage)
}

func (manager *SQuotaBaseManager) _cancelPendingUsage(ctx context.Context, userCred mcclient.TokenCredential, localUsage IQuota, cancelUsage IQuota) error {
	pendingUsage := manager.newQuota()
	err := manager.pendingStore.SubQuota(ctx, userCred, cancelUsage, pendingUsage)
	if err != nil {
		return errors.Wrap(err, "manager.pendingStore.SubQuota")
	}
	//
	if localUsage != nil {
		localUsage.Sub(cancelUsage)
	}
	// update usages
	quotas, err := manager.usageStore.GetParentQuotas(ctx, cancelUsage.GetKeys())
	if err != nil {
		return errors.Wrap(err, "manager.usageStore.GetParentQuotas")
	}
	for i := range quotas {
		cancelUsage.SetKeys(quotas[i].GetKeys())
		err := manager.usageStore.AddQuota(ctx, userCred, cancelUsage, quotas[i])
		if err != nil {
			return errors.Wrap(err, "manager.usageStore.AddQuota")
		}
	}
	return nil
}

/*func (manager *SQuotaBaseManager) GetPendingUsage(ctx context.Context, keys SQuotaKeys, quota IQuota) error {
	return manager.pendingStore.GetQuota(ctx, keys, quota)
}*/

func (manager *SQuotaBaseManager) GetPendingUsages(ctx context.Context, keys IQuotaKeys) ([]IQuota, error) {
	quotas, err := manager.pendingStore.GetChildrenQuotas(ctx, keys)
	if err != nil {
		return nil, errors.Wrap(err, "manager.pendingStore.GetChildrenQuotas")
	}
	ret := make([]IQuota, 0)
	for _, q := range quotas {
		if !q.IsEmpty() {
			ret = append(ret, q)
		}
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return ret, nil
}

func (manager *SQuotaBaseManager) GetQuota(ctx context.Context, keys IQuotaKeys, quota IQuota) error {
	return manager.getQuotaByKeys(ctx, keys, quota)
}

func (manager *SQuotaBaseManager) GetChildrenQuotas(ctx context.Context, keys IQuotaKeys) ([]IQuota, error) {
	return manager.getQuotasInternal(ctx, keys, false)
}

func (manager *SQuotaBaseManager) GetParentQuotas(ctx context.Context, keys IQuotaKeys) ([]IQuota, error) {
	return manager.getQuotasInternal(ctx, keys, true)
}

func (manager *SQuotaBaseManager) SetQuota(ctx context.Context, userCred mcclient.TokenCredential, quota IQuota) error {
	LockQuota(ctx, quota)
	defer ReleaseQuota(ctx, quota)

	return manager.setQuotaInternal(ctx, userCred, quota)
}

func (manager *SQuotaBaseManager) AddQuota(ctx context.Context, userCred mcclient.TokenCredential, diff IQuota, target IQuota) error {
	LockQuota(ctx, diff)
	defer ReleaseQuota(ctx, diff)

	return manager.addQuotaInternal(ctx, userCred, diff, target)
}
func (manager *SQuotaBaseManager) SubQuota(ctx context.Context, userCred mcclient.TokenCredential, diff IQuota, target IQuota) error {
	LockQuota(ctx, diff)
	defer ReleaseQuota(ctx, diff)

	return manager.subQuotaInternal(ctx, userCred, diff, target)
}

func (manager *SQuotaBaseManager) DeleteQuota(ctx context.Context, userCred mcclient.TokenCredential, keys IQuotaKeys) error {
	LockQuotaKeys(ctx, keys)
	defer ReleaseQuotaKeys(ctx, keys)

	return manager.deleteQuotaByKeys(ctx, userCred, keys)
}

func (manager *SQuotaBaseManager) DeleteAllQuotas(ctx context.Context, userCred mcclient.TokenCredential, keys IQuotaKeys) error {
	LockQuotaKeys(ctx, keys)
	defer ReleaseQuotaKeys(ctx, keys)

	return manager.deleteAllQuotas(ctx, userCred, keys)
}

func (manager *SQuotaBaseManager) CheckQuota(ctx context.Context, request IQuota) error {
	LockQuota(ctx, request)
	defer ReleaseQuota(ctx, request)

	return manager._checkQuota(ctx, request)
}

func (manager *SQuotaBaseManager) __checkQuota(ctx context.Context, quota IQuota, request IQuota) error {
	used := manager.newQuota()
	err := manager.usageStore.GetQuota(ctx, quota.GetKeys(), used)
	if err != nil {
		return errors.Wrap(err, "manager.usageStore.GetQuotaByKeys")
	}
	pendings, err := manager.pendingStore.GetChildrenQuotas(ctx, quota.GetKeys())
	if err != nil {
		return errors.Wrap(err, "manager.pendingStore.GetChildrenQuotas")
	}
	for i := range pendings {
		used.Add(pendings[i])
	}
	return used.Exceed(quota, request)
}

func (manager *SQuotaBaseManager) _checkQuota(ctx context.Context, request IQuota) error {
	quotas, err := manager.GetParentQuotas(ctx, request.GetKeys())
	if err != nil {
		return errors.Wrap(err, "manager.getMatchedQuotas")
	}
	for i := range quotas {
		err := manager.__checkQuota(ctx, quotas[i], request)
		if err != nil {
			return errors.Wrap(err, "manager.__checkQuota")
		}
	}
	return nil
}

func (manager *SQuotaBaseManager) CheckSetPendingQuota(ctx context.Context, userCred mcclient.TokenCredential, quota IQuota) error {
	LockQuota(ctx, quota)
	defer ReleaseQuota(ctx, quota)

	return manager._checkSetPendingQuota(ctx, userCred, quota, true)
}

func (manager *SQuotaBaseManager) _checkSetPendingQuota(ctx context.Context, userCred mcclient.TokenCredential, quota IQuota, setPending bool) error {
	err := manager._checkQuota(ctx, quota)
	if err != nil {
		return err
	}
	if setPending {
		target := manager.newQuota()
		err = manager.pendingStore.AddQuota(ctx, userCred, quota, target)
		if err != nil {
			return errors.Wrap(err, "manager.pendingStore.AddQuota")
		}
	}
	return nil
}
