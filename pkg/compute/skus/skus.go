package skus

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SkusZone struct {
	Provider         string
	RegionId         string
	ZoneId           string
	ExternalZoneId   string
	ExternalRegionId string
	skus             []jsonutils.JSONObject
	total            int
	updated          int
	created          int
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
	os_name, err := ndata.GetString("os_name")
	if err != nil {
		log.Debugf("no os name %s", ndata)
	}

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

func (self *SkusZone) Init() error {
	s := auth.GetAdminSession(options.Options.Region, "")
	p, r, z := self.getExternalZone()
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

func (self *SkusZone) doCreate(data models.SServerSku) error {
	data.CloudregionId = self.RegionId
	data.ZoneId = self.ZoneId
	data.Provider = self.Provider
	if err := models.ServerSkuManager.TableSpec().Insert(&data); err != nil {
		log.Debugf("SkusZone doCreate fail: %s", err.Error())
		return err
	}

	self.created += 1
	return nil
}

func (self *SkusZone) doUpdate(odata *models.SServerSku, sku jsonutils.JSONObject) error {
	_, err := models.ServerSkuManager.TableSpec().Update(odata, func() error {
		if err := sku.Unmarshal(&odata); err != nil {
			return err
		}
		odata.CloudregionId = self.RegionId
		odata.ZoneId = self.ZoneId
		odata.Provider = self.Provider
		return nil
	})

	if err != nil {
		log.Debugf("SkusZone doUpdate fail: %s", err.Error())
		return err
	}

	self.updated += 1
	return nil
}

func (self *SkusZone) SyncToLocalDB() error {
	log.Debugf("SkusZone %s start sync.", self.ExternalZoneId)
	for _, sku := range self.skus {
		name, _ := sku.GetString("name")

		if obj, err := models.ServerSkuManager.FetchByZoneId(self.ZoneId, name); err != nil {
			if err != sql.ErrNoRows {
				log.Debugf("SyncToLocalDB zone %s name %s : %s", self.ZoneId, name, err.Error())
				return err
			}

			data := models.SServerSku{}
			if e := sku.Unmarshal(&data); e != nil {
				log.Debugf("sku Unmarshal failed: %s, %s", sku, e.Error())
				return e
			}

			if err := self.doCreate(data); err != nil {
				return err
			}
		} else {
			odata, ok := obj.(*models.SServerSku)
			if !ok {
				return fmt.Errorf("SkusZone model assertion error. %s", obj)
			}

			if err := self.doUpdate(odata, sku); err != nil {
				return err
			}
		}
	}

	defer log.Debugf("SkusZone %s sync to local db.total %d,created %d,updated %d", self.ExternalZoneId, self.total, self.created, self.updated)
	return nil
}

func (self *SkusZone) getExternalZone() (string, string, string) {
	parts := strings.Split(self.ExternalZoneId, "/")
	if len(parts) == 3 {
		// provider, region, zone
		return parts[0], parts[1], parts[2]
	} else if len(parts) == 2 && parts[0] == models.CLOUD_PROVIDER_AZURE {
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

func (self *SkusZoneList) initData(provider string, region models.SCloudregion, zones []models.SZone) {
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
		regions, e := models.CloudregionManager.GetRegionByProvider(p)
		if e != nil {
			return e
		}

		for _, r := range regions {
			zones, e := models.ZoneManager.GetZonesByRegion(&r)
			if e != nil {
				return e
			}

			self.initData(p, r, zones)
		}
	}

	self.total = len(self.Data)
	self.scuccesed = 0
	self.failed = 0
	return nil
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

func SyncSkus(ctx context.Context, userCred mcclient.TokenCredential) {
	skus := SkusZoneList{}
	if e := skus.Refresh(nil); e != nil {
		log.Errorf("SyncSkus refresh failed, %s", e.Error())
	}

	if e := skus.SyncToLocalDB(); e != nil {
		log.Errorf("SyncSkus sync to local db failed, %s", e.Error())
	}
}


func SyncSkusByProviderIds(providerIds []string) error {
	skus := SkusZoneList{}
	if e := skus.Refresh(&providerIds); e != nil {
		return fmt.Errorf("SyncSkus refresh failed, %s", e.Error())
	}

	if e := skus.SyncToLocalDB(); e != nil {
		return fmt.Errorf("SyncSkus sync to local db failed, %s", e.Error())
	}

	return nil
}