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
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/httputils"
	v "yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

const (
	EMPTY_MD5 = "d751713988987e9331980363e24189ce"
)

/*
资源套餐下载连接信息
server: 虚拟机
elasticcache: 弹性缓存(redis&memcached)
*/
type SSkuResourcesMeta struct {
	DBInstanceBase   string `json:"dbinstance_base"`
	ServerBase       string `json:"server_base"`
	ElasticCacheBase string `json:"elastic_cache_base"`
	ImageBase        string `json:"image_base"`
	NatBase          string `json:"nat_base"`
	NasBase          string `json:"nas_base"`
	WafBase          string `json:"waf_base"`
}

var skuIndex = map[string]string{}
var imageIndex = map[string]string{}

func (self *SSkuResourcesMeta) getZoneIdBySuffix(zoneMaps map[string]string, suffix string) string {
	for externalId, id := range zoneMaps {
		if strings.HasSuffix(externalId, suffix) {
			return id
		}
	}
	return ""
}

func (self *SSkuResourcesMeta) GetCloudimages(regionExternalId string) ([]SCachedimage, error) {
	objs, err := self.getObjsByRegion(self.ImageBase, regionExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "getObjsByRegion")
	}
	images := []SCachedimage{}
	err = jsonutils.Update(&images, objs)
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Update")
	}
	return images, nil
}

func (self *SSkuResourcesMeta) GetDBInstanceSkusByRegionExternalId(regionExternalId string) ([]SDBInstanceSku, error) {
	regionId, zoneMaps, err := self.GetRegionIdAndZoneMaps(regionExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionIdAndZoneMaps")
	}
	result := []SDBInstanceSku{}
	objs, err := self.getObjsByRegion(self.DBInstanceBase, regionExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "getSkusByRegion")
	}

	noZoneIds, cnt := []string{}, 0
	for _, obj := range objs {
		sku := SDBInstanceSku{}
		sku.SetModelManager(DBInstanceSkuManager, &sku)
		err = obj.Unmarshal(&sku)
		if err != nil {
			return nil, errors.Wrapf(err, "obj.Unmarshal")
		}
		if len(sku.Zone1) > 0 {
			zoneId := self.getZoneIdBySuffix(zoneMaps, sku.Zone1) // Huawei rds sku zone1 maybe is cn-north-4f
			if len(zoneId) == 0 {
				if !utils.IsInStringArray(sku.Zone1, noZoneIds) {
					noZoneIds = append(noZoneIds, sku.Zone1)
				}
				cnt++
				continue
			}
			sku.Zone1 = zoneId
		}

		if len(sku.Zone2) > 0 {
			zoneId := self.getZoneIdBySuffix(zoneMaps, sku.Zone2)
			if len(zoneId) == 0 {
				if !utils.IsInStringArray(sku.Zone2, noZoneIds) {
					noZoneIds = append(noZoneIds, sku.Zone2)
				}
				cnt++
				continue
			}
			sku.Zone2 = zoneId
		}

		if len(sku.Zone3) > 0 {
			zoneId := self.getZoneIdBySuffix(zoneMaps, sku.Zone3)
			if len(zoneId) == 0 {
				if !utils.IsInStringArray(sku.Zone3, noZoneIds) {
					noZoneIds = append(noZoneIds, sku.Zone3)
				}
				cnt++
				continue
			}
			sku.Zone3 = zoneId
		}

		sku.Id = ""
		sku.CloudregionId = regionId

		result = append(result, sku)
	}
	if len(noZoneIds) > 0 {
		log.Warningf("can not fetch rds sku %d zone %s for %s", cnt, noZoneIds, regionExternalId)
	}
	return result, nil
}

func (self *SSkuResourcesMeta) GetNatSkusByRegionExternalId(regionExternalId string) ([]SNatSku, error) {
	regionId, zoneMaps, err := self.GetRegionIdAndZoneMaps(regionExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionIdAndZoneMaps")
	}
	result := []SNatSku{}
	objs, err := self.getObjsByRegion(self.NatBase, regionExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "getSkusByRegion")
	}
	noZoneIds, cnt := []string{}, 0
	for _, obj := range objs {
		sku := SNatSku{}
		sku.SetModelManager(NatSkuManager, &sku)
		err = obj.Unmarshal(&sku)
		if err != nil {
			return nil, errors.Wrapf(err, "obj.Unmarshal")
		}
		if len(sku.ZoneIds) > 0 {
			zoneIds := []string{}
			for _, zoneExtId := range strings.Split(sku.ZoneIds, ",") {
				zoneId := self.getZoneIdBySuffix(zoneMaps, zoneExtId) // Huawei rds sku zone1 maybe is cn-north-4f
				if len(zoneId) == 0 {
					if !utils.IsInStringArray(zoneExtId, noZoneIds) {
						noZoneIds = append(noZoneIds, zoneExtId)
					}
					cnt++
					continue
				}
				zoneIds = append(zoneIds, zoneId)
			}
			sku.ZoneIds = strings.Join(zoneIds, ",")
		}
		sku.Id = ""
		sku.CloudregionId = regionId
		result = append(result, sku)
	}
	if len(noZoneIds) > 0 {
		log.Warningf("can not fetch nat sku %d zone %s for %s", cnt, noZoneIds, regionExternalId)
	}
	return result, nil
}

func (self *SSkuResourcesMeta) GetNasSkusByRegionExternalId(regionExternalId string) ([]SNasSku, error) {
	regionId, zoneMaps, err := self.GetRegionIdAndZoneMaps(regionExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionIdAndZoneMaps")
	}
	result := []SNasSku{}
	objs, err := self.getObjsByRegion(self.NasBase, regionExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "getSkusByRegion")
	}
	noZoneIds, cnt := []string{}, 0
	for _, obj := range objs {
		sku := SNasSku{}
		sku.SetModelManager(NasSkuManager, &sku)
		err = obj.Unmarshal(&sku)
		if err != nil {
			return nil, errors.Wrapf(err, "obj.Unmarshal")
		}
		if len(sku.ZoneIds) > 0 {
			zoneIds := []string{}
			for _, zoneExtId := range strings.Split(sku.ZoneIds, ",") {
				zoneId := self.getZoneIdBySuffix(zoneMaps, zoneExtId) // Huawei rds sku zone1 maybe is cn-north-4f
				if len(zoneId) == 0 {
					if !utils.IsInStringArray(zoneExtId, noZoneIds) {
						noZoneIds = append(noZoneIds, zoneExtId)
					}
					cnt++
					continue
				}
				zoneIds = append(zoneIds, zoneId)
			}
			sku.ZoneIds = strings.Join(zoneIds, ",")
		}
		sku.Id = ""
		sku.CloudregionId = regionId
		result = append(result, sku)
	}
	if len(noZoneIds) > 0 {
		log.Warningf("can not fetch nas sku %d zone %s for %s", cnt, noZoneIds, regionExternalId)
	}
	return result, nil
}

func (self *SSkuResourcesMeta) getCloudregion(regionExternalId string) (*SCloudregion, error) {
	region, err := db.FetchByExternalId(CloudregionManager, regionExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchByExternalId(%s)", regionExternalId)
	}
	return region.(*SCloudregion), nil
}

func (self *SSkuResourcesMeta) GetRegionIdAndZoneMaps(regionExternalId string) (string, map[string]string, error) {
	region, err := self.getCloudregion(regionExternalId)
	if err != nil {
		return "", nil, errors.Wrap(err, "getCloudregion")
	}
	zones, err := region.GetZones()
	if err != nil {
		return "", nil, errors.Wrap(err, "GetZones")
	}
	zoneMaps := map[string]string{}
	for _, zone := range zones {
		zoneMaps[zone.ExternalId] = zone.Id
	}
	return region.Id, zoneMaps, nil
}

func (self *SSkuResourcesMeta) GetServerSkusByRegionExternalId(regionExternalId string) ([]SServerSku, error) {
	regionId, zoneMaps, err := self.GetRegionIdAndZoneMaps(regionExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionIdAndZoneMaps")
	}
	result := []SServerSku{}
	objs, err := self.getObjsByRegion(self.ServerBase, regionExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "getSkusByRegion")
	}
	noZoneIds, cnt := []string{}, 0
	for _, obj := range objs {
		sku := SServerSku{}
		sku.SetModelManager(ServerSkuManager, &sku)
		err = obj.Unmarshal(&sku)
		if err != nil {
			return nil, errors.Wrapf(err, "obj.Unmarshal")
		}
		if len(sku.ZoneId) > 0 {
			zoneId := self.getZoneIdBySuffix(zoneMaps, sku.ZoneId)
			if len(zoneId) == 0 {
				if !utils.IsInStringArray(sku.ZoneId, noZoneIds) {
					noZoneIds = append(noZoneIds, sku.ZoneId)
				}
				cnt++
				continue
			}
			sku.ZoneId = zoneId
		}
		sku.Id = ""
		sku.CloudregionId = regionId
		result = append(result, sku)
	}
	if len(noZoneIds) > 0 {
		log.Warningf("can not fetch server sku %d zone id %s for region %s", cnt, noZoneIds, regionExternalId)
	}
	return result, nil
}

func getElaticCacheSkuRegionExtId(regionExtId string) string {
	if strings.HasPrefix(regionExtId, apis.CLOUD_ACCESS_ENV_ALIYUN_FINANCE) && strings.HasSuffix(regionExtId, "cn-hangzhou") {
		return regionExtId + "-finance"
	}

	return regionExtId
}

func getElaticCacheSkuZoneId(zoneExtId string) string {
	if strings.HasPrefix(zoneExtId, apis.CLOUD_ACCESS_ENV_ALIYUN_FINANCE) {
		if strings.HasSuffix(zoneExtId, "cn-hangzhou-finance-b") || strings.HasSuffix(zoneExtId, "cn-hangzhou-finance-c") || strings.HasSuffix(zoneExtId, "cn-hangzhou-finance-d") {
			zoneExtId = strings.Replace(zoneExtId, "-finance", "", -1)
		} else if strings.Contains(zoneExtId, "cn-hangzhou") {
			zoneExtId = strings.Replace(zoneExtId, "-finance", "", 1)
		}
	}

	return zoneExtId
}

func (self *SSkuResourcesMeta) GetElasticCacheSkusByRegionExternalId(regionExternalId string) ([]SElasticcacheSku, error) {
	regionId, zoneMaps, err := self.GetRegionIdAndZoneMaps(regionExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionIdAndZoneMaps")
	}
	result := []SElasticcacheSku{}

	noZoneIds, cnt := []string{}, 0

	// aliyun finance cloud
	remoteRegion := getElaticCacheSkuRegionExtId(regionExternalId)
	objs, err := self.getObjsByRegion(self.ElasticCacheBase, remoteRegion)
	if err != nil {
		return nil, errors.Wrap(err, "getObjsByRegion")
	}
	for _, obj := range objs {
		sku := SElasticcacheSku{}
		sku.SetModelManager(ElasticcacheSkuManager, &sku)
		err = obj.Unmarshal(&sku)
		if err != nil {
			return nil, errors.Wrapf(err, "obj.Unmarshal")
		}
		if len(sku.ZoneId) > 0 {
			zoneId := self.getZoneIdBySuffix(zoneMaps, getElaticCacheSkuZoneId(sku.ZoneId))
			if len(zoneId) == 0 {
				if !utils.IsInStringArray(sku.ZoneId, noZoneIds) {
					noZoneIds = append(noZoneIds, sku.ZoneId)
				}
				cnt++
				continue
			}
			sku.ZoneId = zoneId
		}
		if len(sku.SlaveZoneId) > 0 {
			zoneId := self.getZoneIdBySuffix(zoneMaps, getElaticCacheSkuZoneId(sku.SlaveZoneId))
			if len(zoneId) == 0 {
				if !utils.IsInStringArray(sku.SlaveZoneId, noZoneIds) {
					noZoneIds = append(noZoneIds, sku.SlaveZoneId)
				}
				cnt++
				continue
			}
			sku.SlaveZoneId = zoneId
		}
		sku.Id = ""
		sku.CloudregionId = regionId
		result = append(result, sku)
	}
	if len(noZoneIds) > 0 {
		log.Warningf("can not fetch redis sku %d zone %s for %s", cnt, noZoneIds, regionExternalId)
	}
	return result, nil
}

func (self *SSkuResourcesMeta) getObjsByRegion(base string, region string) ([]jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/%s.json", base, region)
	items, err := self._get(url)
	if err != nil {
		return nil, errors.Wrap(err, "getSkusByRegion.get")
	}
	return items, nil
}

func (self *SSkuResourcesMeta) request(url string) (jsonutils.JSONObject, error) {
	client := httputils.GetAdaptiveTimeoutClient()

	header := http.Header{}
	header.Set("User-Agent", "vendor/yunion-OneCloud@"+v.Get().GitVersion)
	_, resp, err := httputils.JSONRequest(client, context.TODO(), httputils.GET, url, header, nil, false)
	return resp, err
}

func (self *SSkuResourcesMeta) getServerSkuIndex() (map[string]string, error) {
	resp, err := self.request(fmt.Sprintf("%s/index.json", self.ServerBase))
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "request")
	}
	ret := map[string]string{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SSkuResourcesMeta) getCloudimageIndex() (map[string]string, error) {
	resp, err := self.request(fmt.Sprintf("%s/index.json", self.ImageBase))
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "request")
	}
	ret := map[string]string{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SSkuResourcesMeta) getWafIndex() (map[string]string, error) {
	resp, err := self.request(fmt.Sprintf("%s/index.json", self.WafBase))
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "request")
	}
	ret := map[string]string{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SSkuResourcesMeta) _get(url string) ([]jsonutils.JSONObject, error) {
	if !strings.HasPrefix(url, "http") {
		return nil, fmt.Errorf("SkuResourcesMeta.get invalid url %s.expected has prefix 'http'", url)
	}

	jsonContent, err := self.request(url)
	if err != nil {
		return nil, errors.Wrapf(err, "request %s", url)
	}
	var ret []jsonutils.JSONObject
	err = jsonContent.Unmarshal(&ret)
	if err != nil {
		return nil, fmt.Errorf("SkuResourcesMeta.get.Unmarshal %s content: %s url: %s", err, jsonContent, url)
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
	if !region.GetDriver().IsSupportedElasticcache() {
		notes := fmt.Sprintf("SyncElasticCacheSkusByRegion %s not support elasticcache", region.Name)
		log.Infof(notes)
		return nil
	}

	meta, err := FetchSkuResourcesMeta()
	if err != nil {
		return errors.Wrap(err, "SyncElasticCacheSkusByRegion.FetchSkuResourcesMeta")
	}

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

	index, err := meta.getServerSkuIndex()
	if err != nil {
		log.Errorf("getServerSkuIndex error: %v", err)
		return
	}

	cloudregions := fetchSkuSyncCloudregions()
	for i := range cloudregions {
		region := &cloudregions[i]
		oldMd5, _ := skuIndex[region.ExternalId]
		newMd5, ok := index[region.ExternalId]
		if ok {
			skuIndex[region.ExternalId] = newMd5
		}

		if newMd5 == EMPTY_MD5 {
			log.Infof("%s Server Skus is empty skip syncing", region.Name)
			continue
		}

		if len(oldMd5) > 0 && newMd5 == oldMd5 {
			log.Infof("%s Server Skus not Changed skip syncing", region.Name)
			continue
		}

		result := ServerSkuManager.SyncServerSkus(ctx, userCred, region, meta)
		notes := fmt.Sprintf("SyncServerSkusByRegion %s result: %s", region.Name, result.Result())
		log.Infof(notes)
	}

	// 清理无效的sku
	log.Debugf("DeleteInvalidSkus in processing...")
	ServerSkuManager.PendingDeleteInvalidSku()
}

// 同步指定region sku列表
func SyncServerSkusByRegion(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, extSkuMeta *SSkuResourcesMeta) compare.SyncResult {
	result := compare.SyncResult{}
	var err error
	if extSkuMeta == nil {
		extSkuMeta, err = FetchSkuResourcesMeta()
		if err != nil {
			result.AddError(errors.Wrap(err, "SyncServerSkusByRegion.FetchSkuResourcesMeta"))
			return result
		}
	}

	result = ServerSkuManager.SyncServerSkus(ctx, userCred, region, extSkuMeta)
	notes := fmt.Sprintf("SyncServerSkusByRegion %s result: %s", region.Name, result.Result())
	log.Infof(notes)

	return result
}

func FetchSkuResourcesMeta() (*SSkuResourcesMeta, error) {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	transport := httputils.GetTransport(true)
	transport.Proxy = options.Options.HttpTransportProxyFunc()
	client := &http.Client{Transport: transport}
	meta, err := compute.OfflineCloudmeta.GetSkuSourcesMeta(s, client)
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

func fetchCloudEnvs() ([]string, error) {
	accounts := []SCloudaccount{}
	q := CloudaccountManager.Query("provider", "access_url").In("provider", CloudproviderManager.GetPublicProviderProvidersQuery()).Distinct()
	err := q.All(&accounts)
	if err != nil {
		return nil, errors.Wrapf(err, "q.All")
	}
	ret := []string{}
	for i := range accounts {
		ret = append(ret, apis.GetCloudEnv(accounts[i].Provider, accounts[i].AccessUrl))
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

type sWafGroup struct {
	SWafRuleGroup
	Rules []SWafRule
}

func (self sWafGroup) GetGlobalId() string {
	return self.ExternalId
}

func (self SWafRule) GetGlobalId() string {
	return self.ExternalId
}

func (self *SSkuResourcesMeta) getCloudWafGroups(cloudEnv string) ([]sWafGroup, error) {
	url := fmt.Sprintf("%s/%s.json", self.WafBase, cloudEnv)
	resp, err := self.request(url)
	if err != nil {
		return nil, errors.Wrapf(err, "_get(%s)", url)
	}
	ret := []sWafGroup{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}
