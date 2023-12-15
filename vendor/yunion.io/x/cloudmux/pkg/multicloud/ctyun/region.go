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

package ctyun

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SNoLbRegion
	multicloud.SNoObjectStorageRegion

	client *SCtyunClient

	product *SProduct

	IsMultiZones bool
	RegionParent string
	RegionId     string
	RegionType   string
	ZoneList     []string
	RegionName   string
}

func (self *SRegion) list(service, res string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	if gotypes.IsNil(params) {
		params = map[string]interface{}{}
	}
	params["regionID"] = self.RegionId
	return self.client.list(service, res, params)
}

func (self *SRegion) post(service, res string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	if gotypes.IsNil(params) {
		params = map[string]interface{}{}
	}
	params["regionID"] = self.RegionId
	return self.client.post(service, res, params)
}

func (self *SRegion) CreateVpc(opts *cloudprovider.VpcCreateOptions) (*SVpc, error) {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"name":        opts.NAME,
		"description": opts.Desc,
		"CIDR":        opts.CIDR,
	}
	resp, err := self.post(SERVICE_VPC, "/v4/vpc/create", params)
	if err != nil {
		return nil, err
	}
	vpcId, err := resp.GetString("returnObj", "vpcID")
	if err != nil {
		return nil, errors.Wrapf(err, "get vpcID")
	}
	return self.GetVpc(vpcId)
}

func (self *SRegion) GetClient() *SCtyunClient {
	return self.client
}

func (self *SRegion) GetCloudEnv() string {
	return api.CLOUD_PROVIDER_CTYUN
}

func (self *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.GetSecurityGroups()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = self
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	sec, err := self.GetSecurityGroup(secgroupId)
	if err != nil {
		return nil, err
	}
	return sec, nil
}

func (self *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	secgroup, err := self.CreateSecurityGroup(opts)
	if err != nil {
		return nil, errors.Wrap(err, "CreateISecurityGroup")
	}

	return secgroup, nil
}

func (self *SRegion) GetId() string {
	id, ok := CtyunRegionIdMap[self.RegionId]
	if ok {
		return id
	}
	return self.RegionId
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_CTYUN_CN, self.RegionName)
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_CTYUN_EN, self.RegionName)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SRegion) getProduct() (*SProduct, error) {
	if !gotypes.IsNil(self.product) {
		return self.product, nil
	}
	var err error
	self.product, err = self.GetProduct()
	return self.product, err
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.client.GetAccessEnv(), self.GetId())
}

func (self *SRegion) GetStatus() string {
	product, err := self.getProduct()
	if err != nil {
		return api.CLOUD_REGION_STATUS_OUTOFSERVICE
	}
	if len(product.Other.Region) == 0 {
		return api.CLOUD_REGION_STATUS_OUTOFSERVICE
	}
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	id := self.GetId()
	if info, ok := LatitudeAndLongitude[id]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) getDefaultZone() *SZone {
	return &SZone{
		region:        self,
		AzDisplayName: "默认可用区",
		Name:          "default",
	}
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		zones[i].region = self
		ret = append(ret, &zones[i])
	}
	if len(ret) == 0 {
		ret = append(ret, self.getDefaultZone())
	}
	return ret, nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		vpcs[i].region = self
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := self.GetEips("")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetIEips.GetEips")
	}

	ret := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eips[i].region = self
		ret = append(ret, &eips[i])
	}

	return ret, nil
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(ivpcs); i += 1 {
		if ivpcs[i].GetGlobalId() == id {
			return ivpcs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEip(id)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.GetInstance(id)
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.GetDisk(id)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	vpc, err := self.CreateVpc(opts)
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	eip, err := self.CreateEip(opts)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return []cloudprovider.ICloudSnapshot{}, nil
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := make([]cloudprovider.ICloudHost, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneHost, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, iZoneHost...)
	}
	return iHosts, nil
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneStores, err := izones[i].GetIStorages()
		if err != nil {
			return nil, err
		}
		iStores = append(iStores, iZoneStores...)
	}
	return iStores, nil
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		istore, err := izones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_CTYUN
}

func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	vms, err := self.GetInstances("", []string{id})
	if err != nil {
		return nil, err
	}
	for i := range vms {
		if vms[i].GetGlobalId() == id {
			return &vms[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetInstances(zoneId string, ids []string) ([]SInstance, error) {
	pageNo := 1
	params := map[string]interface{}{
		"pageNo":   pageNo,
		"pageSize": 50,
	}
	if len(ids) > 0 {
		params["instanceIDList"] = strings.Join(ids, ",")
	}
	if len(zoneId) > 0 {
		params["azName"] = zoneId
	}
	ret := []SInstance{}
	for {
		resp, err := self.post(SERVICE_ECS, "/v4/ecs/list-instances", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			ReturnObj struct {
				Results []SInstance
			}
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ReturnObj.Results...)
		if len(ret) >= part.TotalCount || len(part.ReturnObj.Results) == 0 {
			break
		}
		pageNo++
		params["pageNo"] = pageNo
	}
	return ret, nil
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := region.GetInstances("", nil)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		ret = append(ret, &vms[i])
	}
	return ret, nil
}
