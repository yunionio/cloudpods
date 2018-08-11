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
