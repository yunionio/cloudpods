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

// todo: 待测试
func (self *SSkuResourcesMeta) GetServerSkus() ([]SServerSku, error) {
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
		provider, _ := item.GetString("provider")
		regionId, _ := item.GetString("cloudregion_id")
		if self.region.GetExternalId() != fmt.Sprintf("%s/%s", provider, regionId) {
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
			if _, ok := self.regionalSkuCaches[url][cloudregion]; !ok && len(cloudregion) > 0 {
				self.regionalSkuCaches[url][cloudregion] = []jsonutils.JSONObject{}
			}
			self.regionalSkuCaches[url][cloudregion] = append(self.regionalSkuCaches[url][cloudregion], items[i])
		}
	}

	for regionId, skus := range self.regionalSkuCaches[url] {
		if strings.HasSuffix(region, regionId) {
			return skus, nil
		}
	}
	return []jsonutils.JSONObject{}, nil
}

func (self *SSkuResourcesMeta) _get(url string) ([]jsonutils.JSONObject, error) {
	if !strings.HasPrefix(url, "http") {
		return nil, fmt.Errorf("SkuResourcesMeta.get invalid url %s.expected has prefix 'http'", url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.Get %s", err)
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.Read %s", err)
	}

	contentJson, err := jsonutils.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.Parse %s", err)
	}

	ret := []jsonutils.JSONObject{}
	err = contentJson.Unmarshal(&ret)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.Unmarshal(%s) %s", contentJson.String(), err)
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

	cloudregions := []SCloudregion{}
	q := CloudregionManager.Query()
	q = q.In("provider", CloudproviderManager.GetPublicProviderProvidersQuery())
	err := db.FetchModelObjects(CloudregionManager, q, &cloudregions)
	if err != nil {
		log.Errorf("SyncElasticCacheSkus.FetchCloudregions failed: %v", err)
		return
	}

	meta, err := fetchSkuResourcesMeta()
	if err != nil {
		log.Errorf("SyncElasticCacheSkus.fetchSkuResourcesMeta %s", err)
		return
	}

	for i := range cloudregions {
		region := &cloudregions[i]
		meta.SetRegionFilter(region)
		result := ElasticcacheSkuManager.syncDBInstanceSkus(ctx, userCred, region, meta)
		notes := fmt.Sprintf("syncElasticCacheSkusByRegion %s result: %s", region.Name, result.Result())
		log.Infof(notes)
	}
}

// 同步Region elasticcache sku列表.
func syncElasticCacheSkusByRegion(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion) {
	meta, err := fetchSkuResourcesMeta()
	if err != nil {
		log.Errorf("syncElasticCacheSkusByRegion.fetchSkuResourcesMeta %s", err)
		return
	}

	meta.SetRegionFilter(region)
	result := ElasticcacheSkuManager.syncDBInstanceSkus(ctx, userCred, region, meta)
	notes := fmt.Sprintf("syncElasticCacheSkusByRegion %s result: %s", region.Name, result.Result())
	log.Infof(notes)
}

func fetchSkuResourcesMeta() (*SSkuResourcesMeta, error) {
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
