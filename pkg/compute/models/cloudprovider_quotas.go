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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SCloudproviderQuotaManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var CloudproviderQuotaManager *SCloudproviderQuotaManager

func init() {
	CloudproviderQuotaManager = &SCloudproviderQuotaManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SCloudproviderQuota{},
			"cloudprovider_quotas_tbl",
			"cloudproviderquota",
			"cloudproviderquotas",
		),
	}
	CloudproviderQuotaManager.SetVirtualObject(CloudproviderQuotaManager)
}

type SCloudproviderQuota struct {
	db.SStandaloneResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	// 配额范围
	// cloudregion: 区域级别
	// cloudprovider: 云订阅级别
	QuotaRange string `width:"32" charset:"ascii" list:"user"`

	// 已使用的配额
	// -1代表未从云平台拿到已使用配额信息
	UsedCount int `nullable:"false" default:"0" list:"user"`

	// 最大配额限制
	MaxCount int `nullable:"false" default:"0" list:"user"`

	// 配额类型
	QuotaType string `width:"32" charset:"ascii" list:"user"`
}

func (manager *SCloudproviderQuotaManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudproviderManager},
	}
}

func (man *SCloudproviderQuotaManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudproviderQuotaListInput,
) (*sqlchemy.SQuery, error) {

	var err error
	q, err = man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	if len(query.QuotaType) > 0 {
		q = q.Equals("quota_type", query.QuotaType)
	}
	if len(query.QuotaRange) > 0 {
		q = q.Equals("quota_range", query.QuotaRange)
	}

	return q, nil
}

func (man *SCloudproviderQuotaManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudproviderQuotaListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SCloudproviderQuotaManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SCloudproviderQuota) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.CloudproviderQuotaDetails, error) {
	return api.CloudproviderQuotaDetails{}, nil
}

func (manager *SCloudproviderQuotaManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudproviderQuotaDetails {
	rows := make([]api.CloudproviderQuotaDetails, len(objs))
	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.CloudproviderQuotaDetails{
			StandaloneResourceDetails: stdRows[i],
			ManagedResourceInfo:       manRows[i],
			CloudregionResourceInfo:   regRows[i],
		}
	}
	return rows
}

func (manager *SCloudproviderQuotaManager) GetQuotas(provider *SCloudprovider, region *SCloudregion, quotaRange string) ([]SCloudproviderQuota, error) {
	q := manager.Query()
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	if region != nil {
		q = q.Equals("cloudregion_id", region.Id)
	}
	if len(quotaRange) > 0 {
		q = q.Equals("quota_range", quotaRange)
	}
	quotas := []SCloudproviderQuota{}
	err := db.FetchModelObjects(manager, q, &quotas)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return quotas, nil
}

func (manager *SCloudproviderQuotaManager) SyncQuotas(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, region *SCloudregion, quotaRange string, iQuotas []cloudprovider.ICloudQuota) compare.SyncResult {
	lockman.LockRawObject(ctx, "quotas", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "quotas", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	result := compare.SyncResult{}

	dbQuotas, err := manager.GetQuotas(provider, region, quotaRange)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SCloudproviderQuota, 0)
	commondb := make([]SCloudproviderQuota, 0)
	commonext := make([]cloudprovider.ICloudQuota, 0)
	added := make([]cloudprovider.ICloudQuota, 0)
	err = compare.CompareSets(dbQuotas, iQuotas, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudQuota(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		err = manager.newFromCloudQuota(ctx, userCred, provider, region, quotaRange, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SCloudproviderQuota) SyncWithCloudQuota(ctx context.Context, userCred mcclient.TokenCredential, iQuota cloudprovider.ICloudQuota) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.UsedCount = iQuota.GetCurrentQuotaUsedCount()
		self.MaxCount = iQuota.GetMaxQuotaCount()
		return nil
	})
	return err
}

func (manager *SCloudproviderQuotaManager) newFromCloudQuota(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, quotaRange string, iQuota cloudprovider.ICloudQuota) error {
	quota := SCloudproviderQuota{}
	quota.SetModelManager(manager, &quota)

	quota.Name = iQuota.GetQuotaType()
	quota.Description = iQuota.GetDesc()
	quota.ExternalId = iQuota.GetGlobalId()
	quota.ManagerId = provider.Id
	quota.QuotaType = iQuota.GetQuotaType()
	quota.QuotaRange = quotaRange
	quota.UsedCount = iQuota.GetCurrentQuotaUsedCount()
	quota.MaxCount = iQuota.GetMaxQuotaCount()
	if region != nil {
		quota.CloudregionId = region.Id
	}

	return manager.TableSpec().Insert(ctx, &quota)
}

func (manager *SCloudproviderQuotaManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
