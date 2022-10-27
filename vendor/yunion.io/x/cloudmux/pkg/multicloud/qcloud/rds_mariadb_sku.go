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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
)

var (
	SUPPORTED_ENGINE_VERSION = []string{"5.7", "10.0", "10.1"}
)

type SaleZoneInfo struct {
	Zone     string
	ZoneId   string
	ZoneName string
}

type SAvailableChoice struct {
	MasterZone SaleZoneInfo
	SlaveZones []SaleZoneInfo
}

type SRegionSaleInfo struct {
	AvailableChoice []SAvailableChoice
	Region          string
	RegionId        string
	RegionName      string
	ZoneList        []SaleZoneInfo
}

func (self *SRegion) DescribeSaleInfo() ([]SRegionSaleInfo, error) {
	resp, err := self.mariadbRequest("DescribeSaleInfo", map[string]string{})
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSaleInfo")
	}
	saleInfo := []SRegionSaleInfo{}
	err = resp.Unmarshal(&saleInfo, "RegionList")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return saleInfo, nil
}

type SInstanceSpec struct {
	Cpu        int
	Machine    string
	MaxStorage int
	Memory     int
	MinStorage int
	NodeCount  int
	Pid        int
	Qps        int
	SuitInfo   string
}

type SInstanceSpecs struct {
	Machine   string
	SpecInfos []SInstanceSpec
}

func (self *SRegion) DescribeDBInstanceSpecs() ([]SInstanceSpecs, error) {
	resp, err := self.mariadbRequest("DescribeDBInstanceSpecs", map[string]string{})
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBInstanceSpecs")
	}
	specs := []SInstanceSpecs{}
	err = resp.Unmarshal(&specs, "Specs")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return specs, nil
}

func (self *SRegion) ListMariadbSkus() ([]SDBInstanceSku, error) {
	skus := []SDBInstanceSku{}

	saleRegions, err := self.DescribeSaleInfo()
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSaleInfo")
	}

	for _, saleRegion := range saleRegions {
		if saleRegion.Region == self.Region {
			specs, err := self.DescribeDBInstanceSpecs()
			if err != nil {
				return nil, errors.Wrapf(err, "DescribeDBInstanceSpecs")
			}
			for _, spec := range specs {
				for _, info := range spec.SpecInfos {
					sku := SDBInstanceSku{
						Region:      self.Region,
						Engine:      api.DBINSTANCE_TYPE_MARIADB,
						Cpu:         info.Cpu,
						StorageMax:  info.MaxStorage,
						StorageMin:  info.MinStorage,
						StorageStep: 10,
						MemoryMb:    info.Memory * 1024,
						Qps:         info.Qps,
						Description: info.SuitInfo,
						Category:    "标准版",
						Status:      api.DBINSTANCE_SKU_AVAILABLE,
					}
					if info.NodeCount == 3 {
						sku.Category = "金融版"
					}
					for _, engineVersion := range SUPPORTED_ENGINE_VERSION {
						sku.EngineVersion = engineVersion
						sku.Zone2 = ""
						for _, zone := range saleRegion.AvailableChoice {
							sku.Zone1 = zone.MasterZone.Zone
							for _, slaveZone := range zone.SlaveZones {
								sku.Zone2 = slaveZone.Zone
								skus = append(skus, sku)
							}
						}
					}
				}
			}
		}
	}
	return skus, nil
}
