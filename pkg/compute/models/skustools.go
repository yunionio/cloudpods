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
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ServerSkus struct {
	zone    *SkusZone
	skus    []jsonutils.JSONObject
	total   int
	updated int
	created int
}

type SkusZone struct {
	Provider         string
	RegionId         string
	ZoneId           string
	ExternalZoneId   string
	ExternalRegionId string

	serverSkus ServerSkus
}

func mergeSkuData(odata, ndata jsonutils.JSONObject) jsonutils.JSONObject {
	data, ok := odata.(*jsonutils.JSONDict)
	if !ok {
		log.Debugf("invalid sku dict data: %s", odata)
	}

	new := processSkuData(ndata)
	if !ok {
		log.Debugf("invalid sku dict data: %s", ndata)
	}

	// merge os_name
	o_osname, _ := data.GetString("os_name")
	if o_osname != "Any" {
		n_osname, nerr := new.GetString("os_name")
		if nerr != nil || n_osname != o_osname {
			data.Set("os_name", jsonutils.NewString("Any"))
		}
	}

	// merge data_disk_typs
	o_disks, oerr := data.GetString("data_disk_types")
	n_disks, nerr := new.GetString("data_disk_types")
	if oerr != nil || nerr != nil || o_disks == "" {
		data.Set("data_disk_types", jsonutils.NewString(""))
	} else {
		if n_disks == "" {
			data.Set("data_disk_types", jsonutils.NewString(""))
		} else {
			data.Set("data_disk_types", jsonutils.NewString(fmt.Sprintf("%s,%s", o_disks, n_disks)))
		}
	}

	return data
}

func processSkuData(ndata jsonutils.JSONObject) jsonutils.JSONObject {
	// 从返回结果中。将os_name统一成windows|Linux|any
	// 将external_id 统一替换成 id
	data, ok := ndata.(*jsonutils.JSONDict)
	if !ok {
		log.Debugf("invalid sku dict data: %s", ndata)
	}

	// 处理os name
	os_name, _ := ndata.GetString("os_name")
	os_name = strings.ToLower(os_name)
	if strings.Contains(os_name, "any") || strings.Contains(os_name, "na") || os_name == "" {
		data.Set("os_name", jsonutils.NewString("Any"))
	} else if os_name != "windows" {
		data.Set("os_name", jsonutils.NewString("Linux"))
	} else {
		data.Set("os_name", jsonutils.NewString("Windows"))
	}

	// 将external_id 统一替换成 id.
	id, err := ndata.GetString("id")
	if err != nil {
		data.Set("external_id", jsonutils.NewString(""))
	} else {
		data.Set("external_id", jsonutils.NewString(id))
		data.Remove("id")
	}

	return data
}

func (self *ServerSkus) Init() error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	p, r, z := self.zone.getExternalZone()
	limit := 1024
	offset := 0
	total := 1024

	records := map[string]jsonutils.JSONObject{}
	for offset < total {
		ret, e := modules.CloudmetaSkus.GetSkus(s, p, r, z, limit, offset)
		if e != nil {
			log.Debugf("SkusZone %s init failed, %s", z, e.Error())
			return e
		}

		for _, sku := range ret.Data {
			name, err := sku.GetString("name")
			if err != nil {
				log.Debugf("SkusZone sku name empty : %s", sku)
				return err
			}

			if odata, exists := records[name]; exists {
				records[name] = mergeSkuData(odata, sku)
			} else {
				records[name] = processSkuData(sku)
			}
		}

		offset += limit
		total = ret.Total
	}

	filtedData := []jsonutils.JSONObject{}
	for _, item := range records {
		filtedData = append(filtedData, item)
	}

	self.total = len(records)
	self.skus = filtedData
	return nil
}

func (self *ServerSkus) SyncToLocalDB() error {
	log.Debugf("SkusZone %s start sync.", self.zone.ExternalZoneId)
	// 更新已经soldout的sku
	localIds, err := ServerSkuManager.FetchAllAvailableSkuIdByZoneId(self.zone.ZoneId)
	if err != nil {
		return err
	}

	// 本次已被更新的sku id
	updatedIds := make([]string, 0)
	for _, sku := range self.skus {
		name, _ := sku.GetString("name")

		if obj, err := ServerSkuManager.FetchByZoneId(self.zone.ZoneId, name); err != nil {
			if err != sql.ErrNoRows {
				log.Debugf("SyncToLocalDB zone %s name %s : %s", self.zone.ZoneId, name, err.Error())
				return err
			}
			data := SServerSku{}
			if e := sku.Unmarshal(&data); e != nil {
				log.Debugf("sku Unmarshal failed: %s, %s", sku, e.Error())
				return e
			}
			if err := self.doCreate(data); err != nil {
				return err
			}
		} else {
			odata, ok := obj.(*SServerSku)
			if !ok {
				return fmt.Errorf("SkusZone model assertion error. %s", obj)
			}

			if err := self.doUpdate(odata, sku); err != nil {
				return err
			}

			updatedIds = append(updatedIds, odata.Id)
		}
	}

	// 处理已经下架的sku： 将本次未更新且处于available状态的sku置为soldout状态
	abandonIds := diff(localIds, updatedIds)
	log.Debugf("SyncToLocalDB abandon sku %s", abandonIds)
	err = ServerSkuManager.MarkAllAsSoldout(abandonIds)
	if err != nil {
		return err
	}

	defer log.Debugf("SkusZone %s sync to local db.total %d,created %d,updated %d. abandoned %d", self.zone.ExternalZoneId, self.total, self.created, self.updated, len(abandonIds))
	return nil
}

func (self *ServerSkus) doCreate(data SServerSku) error {
	data.CloudregionId = self.zone.RegionId
	data.ZoneId = self.zone.ZoneId
	data.Provider = self.zone.Provider
	data.Status = api.SkuStatusReady
	data.Enabled = true
	if err := ServerSkuManager.TableSpec().Insert(&data); err != nil {
		log.Debugf("SkusZone doCreate fail: %s", err.Error())
		return err
	}

	self.created += 1
	return nil
}

func (self *ServerSkus) doUpdate(odata *SServerSku, sku jsonutils.JSONObject) error {
	_, err := db.Update(odata, func() error {
		if err := sku.Unmarshal(&odata); err != nil {
			return err
		}
		odata.CloudregionId = self.zone.RegionId
		odata.ZoneId = self.zone.ZoneId
		odata.Provider = self.zone.Provider
		// 公有云默认都是ready并启用
		odata.Status = api.SkuStatusReady
		odata.Enabled = true
		return nil
	})

	if err != nil {
		log.Debugf("SkusZone doUpdate fail: %s", err.Error())
		return err
	}

	self.updated += 1
	return nil
}

func (self *SkusZone) Init() error {
	self.serverSkus = ServerSkus{zone: self}

	err := self.serverSkus.Init()
	if err != nil {
		return errors.Wrap(err, "SkusZone.Init.serverSkus")
	}

	return nil
}

func (self *SkusZone) SyncToLocalDB() error {
	err := self.serverSkus.SyncToLocalDB()
	if err != nil {
		return err
	}

	return nil
}

func (self *SkusZone) getExternalZone() (string, string, string) {
	parts := strings.Split(self.ExternalZoneId, "/")
	if len(parts) == 3 {
		// provider, region, zone
		return parts[0], parts[1], parts[2]
	} else if len(parts) == 2 && parts[0] == api.CLOUD_PROVIDER_AZURE {
		// azure 没有zone的概念
		return parts[0], parts[1], parts[1]
	}

	log.Debugf("SkusZone invalid external zone id %s", self.ExternalZoneId)
	return "", "", ""
}

type SkusZoneList struct {
	Data      []*SkusZone
	total     int
	scuccesed int
	failed    int
}

func (self *SkusZoneList) initData(provider string, region SCloudregion, zones []SZone) {
	for _, z := range zones {
		log.Debugf("SkusZoneList initData provider %s zone %s", provider, z.GetId())
		skusZone := &SkusZone{
			Provider:         provider,
			RegionId:         region.GetId(),
			ZoneId:           z.GetId(),
			ExternalZoneId:   z.GetExternalId(),
			ExternalRegionId: region.GetExternalId(),
		}
		self.Data = append(self.Data, skusZone)
	}
}

func (self *SkusZoneList) Refresh(providerIds *[]string) error {
	self.Data = []*SkusZone{}

	var pIds []string
	if providerIds == nil {
		pIds = cloudprovider.GetRegistedProviderIds()
	} else {
		pIds = *providerIds
	}

	for _, p := range pIds {
		regions, e := CloudregionManager.GetRegionByProvider(p)
		if e != nil {
			return e
		}

		for _, r := range regions {
			zones, e := ZoneManager.GetZonesByRegion(&r)
			if e != nil {
				return e
			}

			self.initData(p, r, zones)
		}
	}

	self.refresh()
	return nil
}

func (self *SkusZoneList) refresh() {
	self.total = len(self.Data)
	self.scuccesed = 0
	self.failed = 0
}

func (self *SkusZoneList) SyncToLocalDB() error {
	var err error
	log.Debugf("######################Start Sync Skus To LocalDB######################")
	for _, d := range self.Data {
		if e := d.Init(); e != nil {
			log.Errorf("SkusZoneList init failed: %s", e.Error())
			self.failed += 1
			err = e
			continue
		}

		if e := d.SyncToLocalDB(); e != nil {
			log.Errorf("SkusZoneList SyncToLocalDB failed: %s", e.Error())
			self.failed += 1
			err = e
			continue
		}

		self.scuccesed += 1

		remain := self.total - self.scuccesed - self.failed
		log.Infof("SkusZoneList total %d, success %d.fail %d, remain %d. sync zone %s.", self.total, self.scuccesed, self.failed, remain, d.ExternalZoneId)
	}
	log.Debugf("######################Finished Sync Skus To LocalDB######################")
	return err
}

// 全量同步sku列表.
func SyncSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
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
	skulist := SkusZoneList{}
	if e := skulist.Refresh(nil); e != nil {
		log.Errorf("SyncSkus refresh failed, %s", e.Error())
	}

	if e := skulist.SyncToLocalDB(); e != nil {
		log.Errorf("SyncSkus sync to local db failed, %s", e.Error())
	}

	// 清理无效的sku
	log.Debugf("DeleteInvalidSkus in processing...")
	ServerSkuManager.PendingDeleteInvalidSku()
}

// 同步指定provider sku列表
func SyncSkusByProviderIds(providerIds []string) error {
	skulist := SkusZoneList{}
	log.Debugf("SyncSkusByProviderIds %s", providerIds)
	if e := skulist.Refresh(&providerIds); e != nil {
		return fmt.Errorf("SyncSkus refresh failed, %s", e.Error())
	}

	if e := skulist.SyncToLocalDB(); e != nil {
		return fmt.Errorf("SyncSkus sync to local db failed, %s", e.Error())
	}

	return nil
}

// 同步指定region sku列表
func syncSkusByRegion(region *SCloudregion) error {
	skulist := SkusZoneList{}
	zones, err := ZoneManager.GetZonesByRegion(region)
	if err != nil {
		return err
	}

	log.Debugf("SyncSkusByRegion %s", region.GetName())
	skulist.initData(region.Provider, *region, zones)
	skulist.refresh()

	if e := skulist.SyncToLocalDB(); e != nil {
		return fmt.Errorf("SyncSkus sync to local db failed, %s", e.Error())
	}

	return nil
}

// 找出origins中存在，但是compares中不存在的element
func diff(origins, compares []string) []string {
	ret := make([]string, 0)
	for _, o := range origins {
		if !utils.IsInStringArray(o, compares) && len(o) > 0 {
			ret = append(ret, o)
		}
	}

	return ret
}
