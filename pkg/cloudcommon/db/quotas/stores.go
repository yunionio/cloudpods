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

	"strings"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type IQuotaStore interface {
	GetQuota(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error
	SetQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error
}

type SMemoryQuotaStore struct {
	store map[string]jsonutils.JSONObject
}

func NewMemoryQuotaStore() *SMemoryQuotaStore {
	return &SMemoryQuotaStore{
		store: make(map[string]jsonutils.JSONObject),
	}
}

func getMemoryStoreKey(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, name []string) string {
	keys := make([]string, 0)
	keys = append(keys, ownerId.GetProjectDomainId())
	if scope == rbacutils.ScopeProject {
		keys = append(keys, ownerId.GetProjectId())
	}
	if len(name) > 0 {
		keys = append(keys, name...)
	}
	return strings.Join(keys, nameSeparator)
}

func (self *SMemoryQuotaStore) GetQuota(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, name []string, quota IQuota) error {
	key := getMemoryStoreKey(scope, ownerId, name)
	json, ok := self.store[key]
	if ok {
		return json.Unmarshal(quota)
	}
	return nil
}

func (self *SMemoryQuotaStore) SetQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, name []string, quota IQuota) error {
	key := getMemoryStoreKey(scope, ownerId, name)
	if quota.IsEmpty() {
		delete(self.store, key)
	} else {
		self.store[key] = jsonutils.Marshal(quota)
	}
	return nil
}

type SDBQuotaStore struct {
}

func NewDBQuotaStore() *SDBQuotaStore {
	return &SDBQuotaStore{}
}

func (store *SDBQuotaStore) GetQuota(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, quota IQuota) error {
	var tenant *db.STenant
	var err error

	switch scope {
	case rbacutils.ScopeDomain:
		tenant, err = db.TenantCacheManager.FetchDomainById(ctx, ownerId.GetProjectDomainId())
	default:
		tenant, err = db.TenantCacheManager.FetchTenantById(ctx, ownerId.GetProjectId())
	}

	if err != nil {
		return err
	}
	quotaStr := tenant.GetMetadata(METADATA_KEY, nil)
	quotaJson, _ := jsonutils.ParseString(quotaStr)
	if quotaJson != nil {
		return quotaJson.Unmarshal(quota)
	}
	return nil
}

/*func (store *SDBQuotaStore) SetQuota(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, quota IQuota) error {
	var tenant *db.STenant
	var err error

	switch scope {
	case rbacutils.ScopeDomain:
		tenant, err = db.TenantCacheManager.FetchDomainById(ctx, ownerId.GetProjectDomainId())
	default:
		tenant, err = db.TenantCacheManager.FetchTenantById(ctx, ownerId.GetProjectId())
	}

	if err != nil {
		return err
	}
	quotaJson := jsonutils.Marshal(quota)
	return tenant.SetMetadata(ctx, METADATA_KEY, quotaJson, userCred)
}*/
