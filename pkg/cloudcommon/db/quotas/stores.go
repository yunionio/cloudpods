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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IQuotaStore interface {
	GetQuota(ctx context.Context, tenantId string, quota IQuota) error
	SetQuota(ctx context.Context, userCred mcclient.TokenCredential, tenantId string, quota IQuota) error
}

type SMemoryQuotaStore struct {
	store map[string]jsonutils.JSONObject
}

func NewMemoryQuotaStore() *SMemoryQuotaStore {
	return &SMemoryQuotaStore{
		store: make(map[string]jsonutils.JSONObject),
	}
}

func (self *SMemoryQuotaStore) GetQuota(ctx context.Context, tenantId string, quota IQuota) error {
	json, ok := self.store[tenantId]
	if ok {
		return json.Unmarshal(quota)
	}
	return nil
}

func (self *SMemoryQuotaStore) SetQuota(ctx context.Context, userCred mcclient.TokenCredential, tenantId string, quota IQuota) error {
	if quota.IsEmpty() {
		delete(self.store, tenantId)
	} else {
		self.store[tenantId] = jsonutils.Marshal(quota)
	}
	return nil
}

type SDBQuotaStore struct {
}

func NewDBQuotaStore() *SDBQuotaStore {
	return &SDBQuotaStore{}
}

func (store *SDBQuotaStore) GetQuota(ctx context.Context, tenantId string, quota IQuota) error {
	tenant, err := db.TenantCacheManager.FetchTenantById(ctx, tenantId)
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

func (store *SDBQuotaStore) SetQuota(ctx context.Context, userCred mcclient.TokenCredential, tenantId string, quota IQuota) error {
	tenant, err := db.TenantCacheManager.FetchTenantById(ctx, tenantId)
	if err != nil {
		return err
	}
	quotaJson := jsonutils.Marshal(quota)
	return tenant.SetMetadata(ctx, METADATA_KEY, quotaJson, userCred)
}
