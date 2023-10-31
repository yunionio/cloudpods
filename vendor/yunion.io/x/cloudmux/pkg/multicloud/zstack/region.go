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

package zstack

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SNoLbRegion
	multicloud.SNoObjectStorageRegion

	client *SZStackClient

	Name string

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc
}

func (region *SRegion) GetClient() *SZStackClient {
	return region.client
}

func (region *SRegion) GetId() string {
	return region.Name
}

func (region *SRegion) GetName() string {
	return region.client.cpcfg.Name
}

func (region *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(region.GetName()).CN(region.GetName())
	return table
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_ZSTACK, region.client.cpcfg.Id)
}

func (region *SRegion) IsEmulated() bool {
	return false
}

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_ZSTACK
}

func (region *SRegion) GetCloudEnv() string {
	return ""
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (region *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (region *SRegion) Refresh() error {
	// do nothing
	return nil
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

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return region.GetHost(id)
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := region.getIStorages("")
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i++ {
		if storages[i].GetGlobalId() == id {
			return storages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := region.GetHosts("", "")
	if err != nil {
		return nil, err
	}
	ihosts := []cloudprovider.ICloudHost{}
	for i := 0; i < len(hosts); i++ {
		ihosts = append(ihosts, &hosts[i])
	}
	return ihosts, nil
}

func (region *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return region.getIStorages("")
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := region.GetIStoragecaches()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(caches); i++ {
		if caches[i].GetGlobalId() == id {
			return caches[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	zones, err := region.GetZones("")
	if err != nil {
		return nil, err
	}
	icaches := []cloudprovider.ICloudStoragecache{}
	for i := 0; i < len(zones); i++ {
		icaches = append(icaches, &SStoragecache{ZoneId: zones[i].UUID, region: region})
	}
	return icaches, nil
}

func (region *SRegion) GetIVpcById(vpcId string) (cloudprovider.ICloudVpc, error) {
	return &SVpc{region: region}, nil
}

func (region *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i++ {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetZone(zoneId string) (*SZone, error) {
	zone := &SZone{region: region}
	return zone, region.client.getResource("zones", zoneId, zone)
}

func (region *SRegion) GetZones(zoneId string) ([]SZone, error) {
	zones := []SZone{}
	params := url.Values{}
	if len(zoneId) > 0 {
		params.Add("q", "uuid="+zoneId)
	}
	err := region.client.listAll("zones", params, &zones)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i++ {
		zones[i].region = region
	}
	return zones, nil
}

func (region *SRegion) fetchZones() {
	if region.izones == nil || len(region.izones) == 0 {
		zones, err := region.GetZones("")
		if err != nil {
			log.Errorf("failed to get zones error: %v", err)
			return
		}
		region.izones = []cloudprovider.ICloudZone{}
		for i := 0; i < len(zones); i++ {
			region.izones = append(region.izones, &zones[i])
		}
	}
}

func (region *SRegion) fetchInfrastructure() error {
	region.fetchZones()
	region.GetIVpcs()
	return nil
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if region.izones == nil {
		if err := region.fetchInfrastructure(); err != nil {
			return nil, err
		}
	}
	return region.izones, nil
}

func (region *SRegion) GetVpc() *SVpc {
	return &SVpc{region: region}
}

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	region.ivpcs = []cloudprovider.ICloudVpc{region.GetVpc()}
	return region.ivpcs, nil
}

func (region *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (region *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	if len(eip.NetworkExternalId) == 0 {
		return nil, fmt.Errorf("networkId cannot be empty")
	}
	networkInfo := strings.Split(eip.NetworkExternalId, "/")
	if len(networkInfo) != 2 {
		return nil, fmt.Errorf("invalid network externalId, it should be `l3networId/networkId` format")
	}
	_, err := region.GetL3Network(networkInfo[0])
	if err != nil {
		return nil, err
	}
	vip, err := region.CreateVirtualIP(eip.Name, "", eip.Ip, networkInfo[0])
	if err != nil {
		return nil, err
	}
	return region.CreateEip(eip.Name, vip.UUID, "")
}

func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	return region.GetEip(eipId)
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := region.GetEips("", "")
	if err != nil {
		return nil, err
	}
	ieips := []cloudprovider.ICloudEIP{}
	for i := 0; i < len(eips); i++ {
		ieips = append(ieips, &eips[i])
	}
	return ieips, nil
}

func (region *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := region.GetSnapshots("", "")
	if err != nil {
		return nil, err
	}
	isnapshots := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = region
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}

func (region *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return region.GetSnapshot(snapshotId)
}

func (region *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	offerings, err := region.GetInstanceOfferings("", "", 0, 0)
	if err != nil {
		return nil, err
	}
	iskus := []cloudprovider.ICloudSku{}
	for i := 0; i < len(offerings); i++ {
		offerings[i].region = region
		iskus = append(iskus, &offerings[i])
	}
	return iskus, nil
}

func (region *SRegion) DeleteISkuByName(name string) error {
	offerings, err := region.GetInstanceOfferings("", name, 0, 0)
	if err != nil {
		return errors.Wrap(err, "region.GetInstanceOfferings")
	}
	for _, offering := range offerings {
		err = offering.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.GetSecurityGroups("", "", "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].region = self
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (region *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return region.GetSecurityGroup(secgroupId)
}

func (region *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return region.CreateSecurityGroup(opts)
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := region.GetInstances("", "", "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		ret = append(ret, &vms[i])
	}
	return ret, nil
}
