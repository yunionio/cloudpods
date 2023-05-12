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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNasSkuManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudregionResourceBaseManager
}

var NasSkuManager *SNasSkuManager

func init() {
	NasSkuManager = &SNasSkuManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SNasSku{},
			"nas_skus_tbl",
			"nas_sku",
			"nas_skus",
		),
	}
	NasSkuManager.NameRequireAscii = false
	NasSkuManager.SetVirtualObject(NasSkuManager)
}

type SNasSku struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase

	PrepaidStatus  string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"` // 预付费资源状态   available|soldout
	PostpaidStatus string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"` // 按需付费资源状态  available|soldout

	StorageType   string `width:"32" index:"true" list:"user" create:"optional"`
	DiskSizeStep  int    `list:"user" default:"-1" create:"optional"` //步长
	MaxDiskSizeGb int    `list:"user" create:"optional"`
	MinDiskSizeGb int    `list:"user" create:"optional"`

	NetworkTypes   string `list:"user" create:"optional"`
	FileSystemType string `list:"user" create:"optional"`
	Protocol       string `list:"user" create:"optional"`

	Provider string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin"`
	ZoneIds  string `charset:"utf8" nullable:"true" list:"user" update:"admin" create:"admin_optional" json:"zone_ids"`
}

func (manager *SNasSkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NasSkuListInput,
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

func (manager *SNasSkuManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NasSkuDetails {
	rows := make([]api.NasSkuDetails, len(objs))

	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.NasSkuDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
			CloudregionResourceInfo:                regRows[i],
		}

		rows[i].CloudEnv = strings.Split(regRows[i].RegionExternalId, "/")[0]
	}

	return rows
}

func (manager *SNasSkuManager) ListItemExportKeys(ctx context.Context,
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

func (manager *SNasSkuManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *SNasSkuManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NasSkuListInput,
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

func (self *SCloudregion) GetNasSkus() ([]SNasSku, error) {
	skus := []SNasSku{}
	q := NasSkuManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(NasSkuManager, q, &skus)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return skus, nil
}

func (self SNasSku) GetGlobalId() string {
	return self.ExternalId
}

func (self *SCloudregion) SyncNasSkus(ctx context.Context, userCred mcclient.TokenCredential, meta *SSkuResourcesMeta, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, "nas-sku")
	defer lockman.ReleaseRawObject(ctx, self.Id, "nas-sku")

	syncResult := compare.SyncResult{}

	iskus, err := meta.GetNasSkusByRegionExternalId(self.ExternalId)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	dbSkus, err := self.GetNasSkus()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SNasSku, 0)
	commondb := make([]SNasSku, 0)
	commonext := make([]SNasSku, 0)
	added := make([]SNasSku, 0)

	err = compare.CompareSets(dbSkus, iskus, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
			continue
		}
		syncResult.Delete()
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].syncWithCloudSku(ctx, userCred, commonext[i])
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		err = self.newFromCloudNasSku(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SNasSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, sku SNasSku) error {
	_, err := db.Update(self, func() error {
		jsonutils.Update(self, sku)
		self.Status = api.NAS_SKU_AVAILABLE
		return nil
	})
	return err
}

func (self *SCloudregion) newFromCloudNasSku(ctx context.Context, userCred mcclient.TokenCredential, isku SNasSku) error {
	sku := &isku
	sku.SetModelManager(NasSkuManager, sku)
	sku.Id = "" //避免使用yunion meta的id,导致出现duplicate entry问题
	sku.Status = api.NAS_SKU_AVAILABLE
	sku.SetEnabled(true)
	sku.CloudregionId = self.Id
	return NasSkuManager.TableSpec().Insert(ctx, sku)
}

func SyncNasSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	err := SyncRegionNasSkus(ctx, userCred, "", isStart, false)
	if err != nil {
		log.Errorf("SyncRegionNasSkus error: %v", err)
	}
}

func SyncRegionNasSkus(ctx context.Context, userCred mcclient.TokenCredential, regionId string, isStart, xor bool) error {
	if isStart {
		q := NasSkuManager.Query()
		if len(regionId) > 0 {
			q = q.Equals("cloudregion_id", regionId)
		}
		cnt, err := q.Limit(1).CountWithError()
		if err != nil && err != sql.ErrNoRows {
			return errors.Wrapf(err, "SyncRegionNasSkus.QueryNasSku")
		}
		if cnt > 0 {
			log.Debugf("SyncRegionNasSkus synced skus, skip...")
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

	meta, err := FetchSkuResourcesMeta()
	if err != nil {
		return errors.Wrapf(err, "FetchSkuResourcesMeta")
	}

	index, err := meta.getSkuIndex(meta.NasBase)
	if err != nil {
		log.Errorf("get nas sku index error: %v", err)
		return err
	}

	for i := range regions {
		region := regions[i]
		if !region.GetDriver().IsSupportedNas() {
			log.Debugf("region %s(%s) not support nas, skip sync", regions[i].Name, regions[i].Id)
			continue
		}

		skuMeta := &SNasSku{}
		skuMeta.SetModelManager(NasSkuManager, skuMeta)
		skuMeta.Id = region.ExternalId

		oldMd5 := db.Metadata.GetStringValue(ctx, skuMeta, db.SKU_METADAT_KEY, userCred)
		newMd5, ok := index[region.ExternalId]
		if ok {
			db.Metadata.SetValue(ctx, skuMeta, db.SKU_METADAT_KEY, newMd5, userCred)
		}

		if newMd5 == EMPTY_MD5 {
			log.Debugf("%s nas skus is empty skip syncing", region.Name)
			continue
		}

		if len(oldMd5) > 0 && newMd5 == oldMd5 {
			log.Debugf("%s nas skus not changed skip syncing", region.Name)
			continue
		}

		result := regions[i].SyncNasSkus(ctx, userCred, meta, xor)
		msg := result.Result()
		notes := fmt.Sprintf("SyncNasSkus for region %s result: %s", regions[i].Name, msg)
		log.Debugf(notes)
	}
	return nil
}

func (manager *SNasSkuManager) PerformSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SkuSyncInput) (jsonutils.JSONObject, error) {
	return PerformActionSyncSkus(ctx, userCred, manager.Keyword(), input)
}

func (manager *SNasSkuManager) GetPropertySyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query api.SkuTaskQueryInput) (jsonutils.JSONObject, error) {
	return GetPropertySkusSyncTasks(ctx, userCred, query)
}
