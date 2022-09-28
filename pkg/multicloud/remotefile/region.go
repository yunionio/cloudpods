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

package remotefile

import (
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	SResourceBase
	multicloud.SRegion
	multicloud.SRegionSecurityGroupBase
	multicloud.SRegionOssBase
	multicloud.SRegionLbBase

	client *SRemoteFileClient
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s%s", self.client.GetCloudRegionExternalIdPrefix(), self.Id)
}

func (self *SRegion) GetProvider() string {
	return self.client.cpcfg.Vendor
}

func (self *SRegion) GetCloudEnv() string {
	return ""
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := self.client.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		if zones[i].RegionId != self.GetId() {
			continue
		}
		zones[i].region = self
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := self.client.GetZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		if zones[i].RegionId != self.GetId() {
			continue
		}
		if zones[i].GetGlobalId() == id {
			zones[i].region = self
			return &zones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.client.GetVpcs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		if vpcs[i].RegionId != self.GetId() {
			continue
		}
		vpcs[i].region = self
		ret = append(ret, &vpcs[i])
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
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := self.client.GetEips()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := range eips {
		if eips[i].RegionId != self.GetId() {
			continue
		}
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eips, err := self.GetIEips()
	if err != nil {
		return nil, err
	}
	for i := range eips {
		if eips[i].GetGlobalId() == id {
			return eips[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.client.GetSecgroups()
	if err != nil {
		return nil, err
	}
	for i := range secgroups {
		if secgroups[i].GetGlobalId() == secgroupId {
			return &secgroups[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.client.GetSecgroups()
	if err != nil {
		return nil, err
	}
	for i := range secgroups {
		if secgroups[i].GetName() == opts.Name {
			return &secgroups[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	lbs, err := self.client.GetLoadbalancers()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudLoadbalancer{}
	for i := range lbs {
		if lbs[i].RegionId != self.GetId() {
			continue
		}
		lbs[i].region = self
		ret = append(ret, &lbs[i])
	}
	return ret, nil
}

func (self *SRegion) GetILoadBalancerById(id string) (cloudprovider.ICloudLoadbalancer, error) {
	lbs, err := self.GetILoadBalancers()
	if err != nil {
		return nil, err
	}
	for i := range lbs {
		if lbs[i].GetGlobalId() == id {
			return lbs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	buckets, err := self.client.GetBuckets()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudBucket{}
	for i := range buckets {
		if buckets[i].RegionId != self.GetId() {
			continue
		}
		buckets[i].region = self
		ret = append(ret, &buckets[i])
	}
	return ret, nil
}

func (self *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	buckets, err := self.GetIBuckets()
	if err != nil {
		return nil, err
	}
	for i := range buckets {
		if buckets[i].GetName() == name {
			return buckets[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	buckets, err := self.GetIBuckets()
	if err != nil {
		return nil, err
	}
	for i := range buckets {
		if buckets[i].GetGlobalId() == name {
			return buckets[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) CreateIBucket(name string, storageClassStr string, acl string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIMiscResources() ([]cloudprovider.ICloudMiscResource, error) {
	misc, err := self.client.GetMisc()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudMiscResource{}
	for i := range misc {
		ret = append(ret, &misc[i])
	}
	return ret, nil
}

func (self *SRegion) GetCapabilities() []string {
	return self.client.GetCapabilities()
}
