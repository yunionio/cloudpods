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
	region       *SCloudregion
	caches       map[string][]jsonutils.JSONObject
	zoneCaches   map[string]*SZone
	regionCaches map[string]*SCloudregion

	Server       string
	ElasticCache string
}

// todo: 待测试
func (self *SSkuResourcesMeta) GetServerSkus() ([]SServerSku, error) {
	result := []SServerSku{}
	err := self.getSkus(self.Server, &result)
	if err != nil {
		return nil, errors.Wrap(err, "SkuResourcesMeta.GetServerSkus")
	}

	// todo: process data here
	return result, nil
}

func (self *SSkuResourcesMeta) GetElasticCacheSkus() ([]SElasticcacheSku, error) {
	result := []SElasticcacheSku{}
	err := self.getSkus(self.ElasticCache, &result)
	if err != nil {
		return nil, errors.Wrap(err, "SkuResourcesMeta.GetElasticCacheSkus")
	}

	// 处理数据
	for i := range result {
		provider := result[i].Provider
		region := result[i].CloudregionId

		result[i].Id = ""
		r, err := self.fetchRegion(provider, region)
		if err != nil {
			return nil, errors.Wrap(err, "SkuResourcesMeta.GetElasticCacheSkus.fetchRegion")
		}
		result[i].CloudregionId = r.GetId()

		if len(result[i].ZoneId) > 0 {
			zone, err := self.fetchZone(provider, region, result[i].ZoneId)
			if err != nil {
				return nil, errors.Wrap(err, "SkuResourcesMeta.GetElasticCacheSkus.MasterZone")
			}

			result[i].ZoneId = zone.GetId()
		}

		if len(result[i].SlaveZoneId) > 0 {
			zone, err := self.fetchZone(provider, region, result[i].SlaveZoneId)
			if err != nil {
				return nil, errors.Wrap(err, "SkuResourcesMeta.GetElasticCacheSkus.SlaveZone")
			}

			result[i].SlaveZoneId = zone.GetId()
		}
	}

	return result, nil
}

func (self *SSkuResourcesMeta) fetchZone(provider, region, zone string) (*SZone, error) {
	if self.zoneCaches == nil {
		self.zoneCaches = map[string]*SZone{}
	}

	zoneExternalId := strings.Join([]string{provider, region, zone}, "/")
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

func (self *SSkuResourcesMeta) fetchRegion(provider, region string) (*SCloudregion, error) {
	if self.regionCaches == nil {
		self.regionCaches = map[string]*SCloudregion{}
	}

	regionExternalId := strings.Join([]string{provider, region}, "/")
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

func (self *SSkuResourcesMeta) getSkus(url string, result interface{}) error {
	objs, err := self.get(self.ElasticCache)
	if err != nil {
		return errors.Wrap(err, "SkuResourcesMeta.getSkus")
	}

	objArray := jsonutils.Marshal(objs)
	err = objArray.Unmarshal(result)
	if err != nil {
		return errors.Wrap(err, "SkuResourcesMeta.Unmarshal")
	}

	return nil
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
