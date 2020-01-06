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
	"io/ioutil"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

/*
资源套餐下载连接信息
server: 虚拟机
elasticcache: 弹性缓存(redis&memcached)
*/
type SSkuResourcesMeta struct {
	region            *SCloudregion
	caches            map[string][]jsonutils.JSONObject
	regionalSkuCaches map[string]map[string][]jsonutils.JSONObject
	zoneCaches        map[string]*SZone
	regionCaches      map[string]*SCloudregion

	Server       string
	ElasticCache string
	DBInstance   string `json:"dbinstance"`
}

func (self *SSkuResourcesMeta) GetServerSkus(region *SCloudregion) ([]SServerSku, error) {
	self.SetRegionFilter(region)

	result := []SServerSku{}
	objs, err := self.get(self.Server)
	if err != nil {
		return nil, errors.Wrap(err, "self.get")
	}
	for _, obj := range objs {
		sku := SServerSku{}
		err = obj.Unmarshal(&sku)
		if err != nil {
			return nil, errors.Wrap(err, "obj.Unmarshal")
		}

		// provider must not be empty
		if len(sku.Provider) == 0 {
			log.Debugf("source sku error: provider should not be empty. %#v", sku)
			continue
		}

		// 处理数据
		sku.Id = ""

		r, err := self.fetchRegion(sku.CloudregionId)
		if err != nil {
			return nil, errors.Wrap(err, "SkuResourcesMeta.GetServerSkus.fetchRegion")
		}
		sku.CloudregionId = r.GetId()

		if len(sku.ZoneId) > 0 {
			zone, err := self.fetchZone(sku.ZoneId)
			if err != nil {
				return nil, errors.Wrap(err, "SkuResourcesMeta.GetServerSkus.fetchZone")
			}

			sku.ZoneId = zone.GetId()
		}

		result = append(result, sku)
	}
	return result, nil
}

func (self *SSkuResourcesMeta) GetDBInstanceSkusByRegion(regionId string) ([]SDBInstanceSku, error) {
	result := []SDBInstanceSku{}
	objs, err := self.getSkusByRegion(self.DBInstance, regionId)
	if err != nil {
		return nil, errors.Wrapf(err, "getSkusByRegion")
	}
	for _, obj := range objs {
		sku := SDBInstanceSku{}
		err = obj.Unmarshal(&sku)
		if err != nil {
			return nil, errors.Wrapf(err, "obj.Unmarshal")
		}
		result = append(result, sku)
	}
	return result, nil
}

func (self *SSkuResourcesMeta) GetElasticCacheSkus() ([]SElasticcacheSku, error) {
	result := []SElasticcacheSku{}
	objs, err := self.get(self.ElasticCache)
	if err != nil {
		return nil, errors.Wrap(err, "self.get(self.ElasticCache)")
	}
	for _, obj := range objs {
		sku := SElasticcacheSku{}
		err = obj.Unmarshal(&sku)
		if err != nil {
			return nil, errors.Wrap(err, "obj.Unmarshal")
		}
		// 处理数据
		sku.Id = ""

		r, err := self.fetchRegion(sku.CloudregionId)
		if err != nil {
			return nil, errors.Wrap(err, "SkuResourcesMeta.GetElasticCacheSkus.fetchRegion")
		}
		sku.CloudregionId = r.GetId()

		if len(sku.ZoneId) > 0 {
			zone, err := self.fetchZone(sku.ZoneId)
			if err != nil {
				return nil, errors.Wrap(err, "SkuResourcesMeta.GetElasticCacheSkus.MasterZone")
			}

			sku.ZoneId = zone.GetId()
		}

		if len(sku.SlaveZoneId) > 0 {
			zone, err := self.fetchZone(sku.SlaveZoneId)
			if err != nil {
				return nil, errors.Wrap(err, "SkuResourcesMeta.GetElasticCacheSkus.SlaveZone")
			}

			sku.SlaveZoneId = zone.GetId()
		}
		result = append(result, sku)
	}

	return result, nil
}

func (self *SSkuResourcesMeta) fetchZone(zoneExternalId string) (*SZone, error) {
	if self.zoneCaches == nil {
		self.zoneCaches = map[string]*SZone{}
	}

	if z, ok := self.zoneCaches[zoneExternalId]; ok {
		return z, nil
	}

	_zone, err := db.FetchByExternalId(ZoneManager, zoneExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "SkuResourcesMeta.fetchZone.FetchByExternalId")
	}

	z := _zone.(*SZone)
	self.zoneCaches[zoneExternalId] = z
	return z, nil
}

func (self *SSkuResourcesMeta) fetchRegion(regionExternalId string) (*SCloudregion, error) {
	if self.regionCaches == nil {
		self.regionCaches = map[string]*SCloudregion{}
	}

	if r, ok := self.regionCaches[regionExternalId]; ok {
		return r, nil
	}

	_region, err := db.FetchByExternalId(CloudregionManager, regionExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "SkuResourcesMeta.fetchRegion.FetchByExternalId")
	}

	r := _region.(*SCloudregion)
	self.regionCaches[regionExternalId] = r
	return r, nil
}

func (self *SSkuResourcesMeta) SetRegionFilter(region *SCloudregion) {
	self.region = region
}

func (self *SSkuResourcesMeta) filterByRegion(items []jsonutils.JSONObject) []jsonutils.JSONObject {
	if self.region == nil {
		return items
	}

	ret := []jsonutils.JSONObject{}
	for i := range items {
		item := items[i]
		regionId, _ := item.GetString("cloudregion_id")
		if self.region.GetExternalId() != strings.TrimSpace(regionId) {
			continue
		}

		ret = append(ret, item)
	}

	return ret
}

func (self *SSkuResourcesMeta) get(url string) ([]jsonutils.JSONObject, error) {
	if self.caches == nil {
		self.caches = map[string][]jsonutils.JSONObject{}
	}

	if items, ok := self.caches[url]; !ok || len(items) == 0 {
		items, err := self._get(url)
		if err != nil {
			return nil, errors.Wrap(err, "SkuResourcesMeta.get")
		}

		self.caches[url] = items
	}

	items := self.caches[url]
	return self.filterByRegion(items), nil
}

func (self *SSkuResourcesMeta) getSkusByRegion(url string, region string) ([]jsonutils.JSONObject, error) {
	items, err := self.get(url)
	if err != nil {
		return nil, err
	}

	if self.regionalSkuCaches == nil {
		self.regionalSkuCaches = map[string]map[string][]jsonutils.JSONObject{}
		self.regionalSkuCaches[url] = map[string][]jsonutils.JSONObject{}
		for i := range items {
			cloudregion, _ := items[i].GetString("cloudregion_id")
			if len(cloudregion) > 0 {
				if _, ok := self.regionalSkuCaches[url][cloudregion]; !ok {
					self.regionalSkuCaches[url][cloudregion] = []jsonutils.JSONObject{}
				}
				self.regionalSkuCaches[url][cloudregion] = append(self.regionalSkuCaches[url][cloudregion], items[i])
			}
		}
	}

	if skus, ok := self.regionalSkuCaches[url][region]; ok {
		return skus, nil
	}
	return []jsonutils.JSONObject{}, nil
}

func (self *SSkuResourcesMeta) _get(url string) ([]jsonutils.JSONObject, error) {
	if !strings.HasPrefix(url, "http") {
		return nil, fmt.Errorf("SkuResourcesMeta.get invalid url %s.expected has prefix 'http'", url)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.NewRequest %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.Get %s", err)
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.ReadAll %s", err)
	}

	jsonContent, err := jsonutils.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.Parse %s", err)
	}

	var ret []jsonutils.JSONObject
	err = jsonContent.Unmarshal(&ret)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.Unmarshal %s", err)
	}

	return ret, nil
}

// 全量同步elasticcache sku列表.
func SyncElasticCacheSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		cnt, err := CloudaccountManager.Query().IsTrue("is_public_cloud").CountWithError()
		if err != nil && err != sql.ErrNoRows {
			log.Debugf("SyncElasticCacheSkus %s.sync skipped...", err)
			return
		} else if cnt == 0 {
			log.Debugf("SyncElasticCacheSkus no public cloud.sync skipped...")
			return
		}

		cnt, err = ElasticcacheSkuManager.Query().Limit(1).CountWithError()
		if err != nil && err != sql.ErrNoRows {
			log.Errorf("SyncElasticCacheSkus.QueryElasticcacheSku %s", err)
			return
		} else if cnt > 0 {
			log.Debugf("SyncElasticCacheSkus synced skus, skip...")
			return
		}
	}

	meta, err := FetchSkuResourcesMeta()
	if err != nil {
		log.Errorf("SyncElasticCacheSkus.FetchSkuResourcesMeta %s", err)
		return
	}

	cloudregions := fetchSkuSyncCloudregions()
	for i := range cloudregions {
		region := &cloudregions[i]

		if region.GetDriver().IsSupportedElasticcache() {
			meta.SetRegionFilter(region)
			result := ElasticcacheSkuManager.SyncElasticcacheSkus(ctx, userCred, region, meta)
			notes := fmt.Sprintf("SyncElasticCacheSkusByRegion %s result: %s", region.Name, result.Result())
			log.Infof(notes)
		} else {
			notes := fmt.Sprintf("SyncElasticCacheSkusByRegion %s not support elasticcache", region.Name)
			log.Infof(notes)
		}
	}
}

// 同步Region elasticcache sku列表.
func SyncElasticCacheSkusByRegion(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion) error {
	if region.GetDriver().IsSupportedElasticcache() {
		notes := fmt.Sprintf("SyncElasticCacheSkusByRegion %s not support elasticcache", region.Name)
		log.Infof(notes)
		return nil
	}

	meta, err := FetchSkuResourcesMeta()
	if err != nil {
		return errors.Wrap(err, "SyncElasticCacheSkusByRegion.FetchSkuResourcesMeta")
	}

	meta.SetRegionFilter(region)
	result := ElasticcacheSkuManager.SyncElasticcacheSkus(ctx, userCred, region, meta)
	notes := fmt.Sprintf("SyncElasticCacheSkusByRegion %s result: %s", region.Name, result.Result())
	log.Infof(notes)
	return nil
}

// 全量同步sku列表.
func SyncServerSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		cnt, err := ServerSkuManager.GetPublicCloudSkuCount()
		if err != nil {
			log.Errorf("GetPublicCloudSkuCount fail %s", err)
			return
		}
		if cnt > 0 {
			log.Debugf("GetPublicCloudSkuCount synced skus, skip...")
			return
		}
	}

	meta, err := FetchSkuResourcesMeta()
	if err != nil {
		log.Errorf("SyncServerSkus.FetchSkuResourcesMeta %s", err)
		return
	}

	cloudregions := fetchSkuSyncCloudregions()
	for i := range cloudregions {
		region := &cloudregions[i]
		meta.SetRegionFilter(region)
		result := ServerSkuManager.SyncServerSkus(ctx, userCred, region, meta)
		notes := fmt.Sprintf("SyncServerSkusByRegion %s result: %s", region.Name, result.Result())
		log.Infof(notes)
	}

	// 清理无效的sku
	log.Debugf("DeleteInvalidSkus in processing...")
	ServerSkuManager.PendingDeleteInvalidSku()
}

// 同步指定region sku列表
func SyncServerSkusByRegion(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion) error {
	meta, err := FetchSkuResourcesMeta()
	if err != nil {
		return errors.Wrap(err, "SyncServerSkusByRegion.FetchSkuResourcesMeta")
	}

	result := ServerSkuManager.SyncServerSkus(ctx, userCred, region, meta)
	notes := fmt.Sprintf("SyncServerSkusByRegion %s result: %s", region.Name, result.Result())
	log.Infof(notes)
	return nil
}

func FetchSkuResourcesMeta() (*SSkuResourcesMeta, error) {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	meta, err := modules.OfflineCloudmeta.GetSkuSourcesMeta(s)
	if err != nil {
		return nil, errors.Wrap(err, "fetchSkuSourceUrls.GetSkuSourcesMeta")
	}

	ret := &SSkuResourcesMeta{}
	err = meta.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrap(err, "fetchSkuSourceUrls.Unmarshal")
	}

	return ret, nil
}

func fetchSkuSyncCloudregions() []SCloudregion {
	cloudregions := []SCloudregion{}
	q := CloudregionManager.Query()
	q = q.In("provider", CloudproviderManager.GetPublicProviderProvidersQuery())
	err := db.FetchModelObjects(CloudregionManager, q, &cloudregions)
	if err != nil {
		log.Errorf("fetchSkuSyncCloudregions.FetchCloudregions failed: %v", err)
		return nil
	}

	return cloudregions
}
