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

package qcloud

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
)

type SellTypeConfig struct {
	Device     string
	Type       string
	CdbType    string
	Memory     int
	Cpu        int
	VolumeMin  int
	VolumeMax  int
	VolumeStep int
	Connection int
	Qps        int
	Iops       int
	Info       string
	Status     string
	Tag        string
}

type SSellType struct {
	TypeName      string
	EngineVersion []string
	Configs       []SellTypeConfig
}

type SZoneConf struct {
	DeployMode []int
	MasterZone []string
	SlaveZone  []string
	BackupZone []string
}

type SZoneSellConf struct {
	Status                 int
	ZoneName               string
	IsCustom               bool
	IsSupportDr            bool
	IsSupportVpc           bool
	HourInstanceSaleMaxNum int
	IsDefaultZone          bool
	IsBm                   bool
	PayType                []string
	ProtectMode            string
	Zone                   string
	SellType               []SSellType
	ZoneConf               SZoneConf
	DrZone                 []string
	IsSupportRemoteRo      bool
}

type SRegionSellConf struct {
	RegionName      string
	Area            string
	IsDefaultRegion bool
	Region          string
	ZonesConf       []SZoneSellConf
}

func (self *SRegion) DescribeDBZoneConfig() ([]SRegionSellConf, error) {
	resp, err := self.cdbRequest("DescribeDBZoneConfig", map[string]string{})
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBZoneConfig")
	}
	skus := []SRegionSellConf{}
	err = resp.Unmarshal(&skus, "Items")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return skus, nil
}

type SDBInstanceSku struct {
	Region        string
	Engine        string
	EngineVersion string
	Category      string
	Iops          int
	Qps           int
	MaxConnection int
	Cpu           int
	MemoryMb      int
	StorageType   string
	StorageMin    int
	StorageMax    int
	StorageStep   int
	Status        string
	Description   string
	Zone1         string
	Zone2         string
	Zone3         string
}

func (self *SRegion) ListMysqlSkus() ([]SDBInstanceSku, error) {
	conf, err := self.DescribeDBZoneConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBZoneConfig")
	}
	skus := []SDBInstanceSku{}
	for _, item := range conf {
		if item.Region != self.Region {
			continue
		}
		sku := SDBInstanceSku{
			Region: self.Region,
			Engine: api.DBINSTANCE_TYPE_MYSQL,
			Status: api.DBINSTANCE_SKU_SOLDOUT,
		}
		for _, zone := range item.ZonesConf {
			//0-未上线；1-上线；2-开放；3-停售；4-不展示
			if zone.Status == 0 || zone.Status == 4 {
				continue
			}
			if zone.Status == 1 || zone.Status == 2 {
				sku.Status = api.DBINSTANCE_SKU_AVAILABLE
			}
			if zone.IsBm { // 跳过黑石
				continue
			}
			for _, sellType := range zone.SellType {
				for _, sellConf := range sellType.Configs {
					sku.Cpu = sellConf.Cpu
					sku.MemoryMb = sellConf.Memory
					sku.Iops = sellConf.Iops
					sku.MaxConnection = sellConf.Connection
					sku.Qps = sellConf.Qps
					sku.StorageMin = sellConf.VolumeMin
					sku.StorageMax = sellConf.VolumeMax
					sku.StorageStep = sellConf.VolumeStep
					sku.Zone1 = zone.Zone
					for _, engineVersion := range sellType.EngineVersion {
						sku.EngineVersion = engineVersion
						sku.Category = sellConf.Type
						sku.Zone2, sku.Zone3 = "", ""
						sku.StorageType = api.STORAGE_LOCAL_SSD
						switch sellConf.Type {
						case "高可用版":
							for _, zone2 := range zone.ZoneConf.SlaveZone {
								sku.Zone2 = zone2
								skus = append(skus, sku)
							}
							if utils.IsInStringArray(engineVersion, []string{"5.6", "5.7", "8.0"}) {
								sku.Category = "金融版"
								for _, zone2 := range zone.ZoneConf.SlaveZone {
									sku.Zone2 = zone2
									sku.Zone3 = zone2
									skus = append(skus, sku)
								}
							}
						case "基础版":
							sku.StorageType = api.STORAGE_CLOUD_SSD
							skus = append(skus, sku)
						default:
							log.Errorf("unknow %s", sellConf.Type)
						}
					}
				}
			}
		}
	}
	return skus, nil
}
