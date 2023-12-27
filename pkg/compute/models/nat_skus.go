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
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/yunionmeta"
)

type SNatSkuManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudregionResourceBaseManager
}

var NatSkuManager *SNatSkuManager

func init() {
	NatSkuManager = &SNatSkuManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SNatSku{},
			"nat_skus_tbl",
			"nat_sku",
			"nat_skus",
		),
	}
	NatSkuManager.NameRequireAscii = false
	NatSkuManager.SetVirtualObject(NatSkuManager)
}

type SNatSku struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase

	PrepaidStatus  string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"` // 预付费资源状态   available|soldout
	PostpaidStatus string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"` // 按需付费资源状态  available|soldout

	Provider string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin"`
	ZoneIds  string `charset:"utf8" nullable:"true" list:"user" update:"admin" create:"admin_optional" json:"zone_ids"`
}

func (manager *SNatSkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatSkuListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	if len(query.PostpaidStatus) > 0 {
		q = q.Equals("postpaid_status", query.PostpaidStatus)
	}
	if len(query.PrepaidStatus) > 0 {
		q = q.Equals("prepaid_status", query.PrepaidStatus)
	}
	if len(query.Providers) > 0 {
		q = q.In("provider", query.Providers)
	}
	return q, nil
}

func (manager *SNatSkuManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NatSkuDetails {
	rows := make([]api.NatSkuDetails, len(objs))

	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.NatSkuDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
			CloudregionResourceInfo:                regRows[i],
		}

		rows[i].CloudEnv = strings.Split(regRows[i].RegionExternalId, "/")[0]
	}

	return rows
}

func (manager *SNatSkuManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SNatSkuManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SNatSkuManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatSkuListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (self *SCloudregion) GetNatSkus() ([]SNatSku, error) {
	skus := []SNatSku{}
	q := NatSkuManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(NatSkuManager, q, &skus)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return skus, nil
}

func (self *SCloudregion) GetElasticcacheSkus() ([]SElasticcacheSku, error) {
	skus := []SElasticcacheSku{}
	q := ElasticcacheSkuManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(ElasticcacheSkuManager, q, &skus)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return skus, nil
}

func (self SNatSku) GetGlobalId() string {
	return self.ExternalId
}

func (self *SCloudregion) SyncNatSkus(ctx context.Context, userCred mcclient.TokenCredential, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, NatSkuManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, self.Id, NatSkuManager.Keyword())

	result := compare.SyncResult{}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		result.Error(errors.Wrapf(err, "FetchYunionmeta"))
		return result
	}

	iskus := []SNatSku{}
	err = meta.List(NatSkuManager.Keyword(), self.ExternalId, &iskus)
	if err != nil {
		result.Error(err)
		return result
	}

	dbSkus, err := self.GetNatSkus()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SNatSku, 0)
	commondb := make([]SNatSku, 0)
	commonext := make([]SNatSku, 0)
	added := make([]SNatSku, 0)

	err = compare.CompareSets(dbSkus, iskus, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].syncWithCloudSku(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		err = self.newFromCloudNatSku(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (self *SNatSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, sku SNatSku) error {
	_, err := db.Update(self, func() error {
		self.PrepaidStatus = sku.PrepaidStatus
		self.PostpaidStatus = sku.PostpaidStatus
		return nil
	})
	return err
}

func (self *SCloudregion) newFromCloudNatSku(ctx context.Context, userCred mcclient.TokenCredential, isku SNatSku) error {
	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		return err
	}
	zones, err := self.GetZones()
	if err != nil {
		return errors.Wrap(err, "GetZones")
	}
	zoneMaps := map[string]string{}
	for _, zone := range zones {
		zoneMaps[zone.ExternalId] = zone.Id
	}

	sku := &SNatSku{}
	sku.SetModelManager(NatSkuManager, sku)

	skuUrl := self.getMetaUrl(meta.NatBase, isku.GetGlobalId())
	err = meta.Get(skuUrl, sku)
	if err != nil {
		return errors.Wrapf(err, "Get")
	}

	if len(sku.ZoneIds) > 0 {
		zoneIds := []string{}
		for _, zoneExtId := range strings.Split(sku.ZoneIds, ",") {
			zoneId := yunionmeta.GetZoneIdBySuffix(zoneMaps, zoneExtId) // Huawei rds sku zone1 maybe is cn-north-4f
			if len(zoneId) > 0 {
				zoneIds = append(zoneIds, zoneId)
			}
		}
		sku.ZoneIds = strings.Join(zoneIds, ",")
	}

	sku.Status = api.NAS_SKU_AVAILABLE
	sku.SetEnabled(true)
	sku.CloudregionId = self.Id
	sku.Provider = self.Provider
	return NatSkuManager.TableSpec().Insert(ctx, sku)
}

func SyncNatSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	err := SyncRegionNatSkus(ctx, userCred, "", isStart, false)
	if err != nil {
		log.Errorf("SyncRegionNatSkus error: %v", err)
	}
}

func SyncRegionNatSkus(ctx context.Context, userCred mcclient.TokenCredential, regionId string, isStart, xor bool) error {
	if isStart {
		q := NatSkuManager.Query()
		if len(regionId) > 0 {
			q = q.Equals("cloudregion_id", regionId)
		}
		cnt, err := q.Limit(1).CountWithError()
		if err != nil && err != sql.ErrNoRows {
			return errors.Wrapf(err, "SyncRegionNatSkus.QueryNatSku")
		}
		if cnt > 0 {
			log.Debugf("SyncRegionNatSkus synced skus, skip...")
			return nil
		}
	}

	q := CloudregionManager.Query()
	q = q.In("provider", CloudproviderManager.GetPublicProviderProvidersQuery())
	if len(regionId) > 0 {
		q = q.Equals("id", regionId)
	}
	regions := []SCloudregion{}
	err := db.FetchModelObjects(CloudregionManager, q, &regions)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(regions) == 0 {
		return nil
	}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		return errors.Wrapf(err, "FetchYunionmeta")
	}

	index, err := meta.Index(NatSkuManager.Keyword())
	if err != nil {
		log.Errorf("get nat sku index error: %v", err)
		return err
	}

	for i := range regions {
		region := regions[i]
		if !region.GetDriver().IsSupportedNatGateway() {
			log.Infof("region %s(%s) not support nat, skip sync", regions[i].Name, regions[i].Id)
			continue
		}

		skuMeta := &SNatSku{}
		skuMeta.SetModelManager(NatSkuManager, skuMeta)
		skuMeta.Id = region.ExternalId

		oldMd5 := db.Metadata.GetStringValue(ctx, skuMeta, db.SKU_METADAT_KEY, userCred)
		newMd5, ok := index[region.ExternalId]
		if !ok || newMd5 == yunionmeta.EMPTY_MD5 || len(oldMd5) > 0 && newMd5 == oldMd5 {
			continue
		}

		db.Metadata.SetValue(ctx, skuMeta, db.SKU_METADAT_KEY, newMd5, userCred)

		result := regions[i].SyncNatSkus(ctx, userCred, xor)
		msg := result.Result()
		notes := fmt.Sprintf("SyncNatSkus for region %s result: %s", regions[i].Name, msg)
		log.Infof(notes)
	}
	return nil
}

func (manager *SNatSkuManager) PerformSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SkuSyncInput) (jsonutils.JSONObject, error) {
	return PerformActionSyncSkus(ctx, userCred, manager.Keyword(), input)
}

func (manager *SNatSkuManager) GetPropertySyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query api.SkuTaskQueryInput) (jsonutils.JSONObject, error) {
	return GetPropertySkusSyncTasks(ctx, userCred, query)
}

func (self *SCloudregion) SyncPrivateCloudNatSkus(ctx context.Context, userCred mcclient.TokenCredential, iskus []cloudprovider.ICloudNatSku) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, NatSkuManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, self.Id, NatSkuManager.Keyword())

	result := compare.SyncResult{}

	dbSkus, err := self.GetNatSkus()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SNatSku, 0)
	commondb := make([]SNatSku, 0)
	commonext := make([]cloudprovider.ICloudNatSku, 0)
	added := make([]cloudprovider.ICloudNatSku, 0)

	err = compare.CompareSets(dbSkus, iskus, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(added); i += 1 {
		err = self.newFromPrivateCloudNatSku(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (self *SCloudregion) newFromPrivateCloudNatSku(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudNatSku) error {
	sku := &SNasSku{}
	sku.SetModelManager(NatSkuManager, sku)
	sku.Name = ext.GetName()
	sku.Description = ext.GetDesc()
	sku.ExternalId = ext.GetGlobalId()
	sku.CloudregionId = self.Id
	sku.Provider = self.Provider
	sku.PrepaidStatus = ext.GetPrepaidStatus()
	sku.PostpaidStatus = ext.GetPostpaidStatus()
	sku.Enabled = tristate.True
	sku.Status = api.NAT_SKU_AVAILABLE
	return NatSkuManager.TableSpec().Insert(ctx, sku)
}
