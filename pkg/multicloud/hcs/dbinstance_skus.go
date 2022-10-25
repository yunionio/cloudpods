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

package hcs

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SDBInstanceSku struct {
	Vcpus        int
	Ram          int //单位GB
	SpecCode     string
	InstanceMode string //实例模型

	StorageType   string
	Az1           string
	Az2           string
	Engine        string
	EngineVersion string
}

func (self *SDBInstanceSku) GetName() string {
	return self.SpecCode
}

func (self *SDBInstanceSku) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s-%s", self.Engine, self.EngineVersion, self.InstanceMode, self.SpecCode, self.StorageType, self.GetZoneId())
}

func (self *SDBInstanceSku) GetCategory() string {
	return self.InstanceMode
}

func (self *SDBInstanceSku) GetDiskSizeStep() int {
	return 10
}

func (self *SDBInstanceSku) GetIOPS() int {
	return 0
}

func (self *SDBInstanceSku) GetTPS() int {
	return 0
}

func (self *SDBInstanceSku) GetZone1Id() string {
	return self.Az1
}

func (self *SDBInstanceSku) GetZone2Id() string {
	return self.Az2
}

func (self *SDBInstanceSku) GetZone3Id() string {
	return ""
}

func (self *SDBInstanceSku) GetZoneId() string {
	zones := []string{}
	for _, zone := range []string{self.Az1, self.Az2} {
		if len(zone) > 0 {
			zones = append(zones, zone)
		}
	}
	return strings.Join(zones, ",")
}

func (self *SDBInstanceSku) GetMaxConnections() int {
	return 0
}

func (self *SDBInstanceSku) GetMaxDiskSizeGb() int {
	return 4000
}

func (self *SDBInstanceSku) GetMinDiskSizeGb() int {
	return 40
}

func (self *SDBInstanceSku) GetVcpuCount() int {
	return self.Vcpus
}

func (self *SDBInstanceSku) GetVmemSizeMb() int {
	return self.Ram * 1024
}

func (self *SDBInstanceSku) GetStorageType() string {
	return self.StorageType
}

func (sku *SDBInstanceSku) GetEngine() string {
	return sku.Engine
}

func (sku *SDBInstanceSku) GetEngineVersion() string {
	return sku.EngineVersion
}

func (self *SDBInstanceSku) GetStatus() string {
	return api.DBINSTANCE_SKU_AVAILABLE
}

func (sku *SDBInstanceSku) GetQPS() int {
	return 0
}

type SDBInstanceFlavor struct {
	Vcpus        int
	Ram          int //单位GB
	SpecCode     string
	InstanceMode string //实例模型

	StorageType   string
	AzStatus      map[string]string
	Engine        string
	EngineVersion string
}

func (self *SDBInstanceFlavor) GetISkus() []cloudprovider.ICloudDBInstanceSku {
	ret := []cloudprovider.ICloudDBInstanceSku{}
	switch self.InstanceMode {
	case api.HUAWEI_DBINSTANCE_CATEGORY_HA:
		for az1, status1 := range self.AzStatus {
			if status1 != "normal" {
				continue
			}
			for az2, status2 := range self.AzStatus {
				if status2 != "normal" {
					continue
				}
				sku := &SDBInstanceSku{
					Vcpus:         self.Vcpus,
					Ram:           self.Ram,
					SpecCode:      self.SpecCode,
					InstanceMode:  self.InstanceMode,
					StorageType:   self.StorageType,
					Az1:           az1,
					Az2:           az2,
					Engine:        self.Engine,
					EngineVersion: self.EngineVersion,
				}
				ret = append(ret, sku)
			}
		}
	case api.HUAWEI_DBINSTANCE_CATEGORY_SINGLE:
		for az, status := range self.AzStatus {
			if status != "normal" {
				continue
			}
			sku := &SDBInstanceSku{
				Vcpus:         self.Vcpus,
				Ram:           self.Ram,
				SpecCode:      self.SpecCode,
				InstanceMode:  self.InstanceMode,
				StorageType:   self.StorageType,
				Az1:           az,
				Engine:        self.Engine,
				EngineVersion: self.EngineVersion,
			}
			ret = append(ret, sku)
		}
	}
	return ret
}

func (self *SRegion) GetIDBInstanceSkus() ([]cloudprovider.ICloudDBInstanceSku, error) {
	skus, err := self.GetDBInstanceSkus()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDBInstanceSku{}
	for i := range skus {
		ret = append(ret, skus[i].GetISkus()...)
	}
	return ret, nil
}

type SDBInstanceDatastore struct {
	Id   string
	Name string
}

func (self *SRegion) GetDBInstanceSkus() ([]SDBInstanceFlavor, error) {
	skus := []SDBInstanceFlavor{}
	for _, engine := range []string{api.DBINSTANCE_TYPE_MYSQL, api.DBINSTANCE_TYPE_POSTGRESQL, api.DBINSTANCE_TYPE_SQLSERVER} {
		stores, err := self.GetDBInstanceDatastores(engine)
		if err != nil {
			return nil, err
		}
		for _, version := range stores {
			storages, err := self.GetDBInstanceStorages(engine, version.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "GetDBInstanceStorages(%s,%s)", engine, version.Name)
			}
			flavors, err := self.GetDBInstanceFlavors(engine, version.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "GetDBInstanceFlavors(%s, %s)", engine, version.Name)
			}
			for i := range flavors {
				flavors[i].Engine = engine
				flavors[i].EngineVersion = version.Name
				for j := range storages {
					flavors[i].StorageType = storages[j].Name
					flavors[i].AzStatus = storages[j].AzStatus
					skus = append(skus, flavors[i])
				}
			}
		}
	}
	return skus, nil
}

// rds规格信息
func (region *SRegion) GetDBInstanceFlavors(engine string, version string) ([]SDBInstanceFlavor, error) {
	flavors := []SDBInstanceFlavor{}
	query := url.Values{}
	query.Add("version_name", version)
	resource := fmt.Sprintf("%s/%s", "flavor", engine)
	err := region.rdsList(resource, query, flavors)
	if err != nil {
		return nil, err
	}
	return flavors, nil
}

// rds数据库引擎版本信息
func (region *SRegion) GetDBInstanceDatastores(engine string) ([]SDBInstanceDatastore, error) {
	stores := []SDBInstanceDatastore{}
	resource := fmt.Sprintf("%s/%s", "datastores", engine)
	err := region.rdsList(resource, nil, stores)
	if err != nil {
		return nil, err
	}
	return stores, nil
}

type SDBInstanceStorage struct {
	Name     string
	AzStatus map[string]string
}

// rds数据库磁盘类型
func (region *SRegion) GetDBInstanceStorages(engine, engineVersion string) ([]SDBInstanceStorage, error) {
	storages := []SDBInstanceStorage{}
	resource := fmt.Sprintf("%s/%s", "storage-type", engine)
	query := url.Values{}
	query.Add("version_name", engineVersion)
	err := region.rdsList(resource, query, storages)
	if err != nil {
		return nil, err
	}
	return storages, nil
}
