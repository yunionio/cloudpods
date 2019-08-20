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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion
	client *SZStackClient

	Name string

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc
}

func (region *SRegion) GetClient() *SZStackClient {
	return region.client
}

func (region *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (region *SRegion) GetId() string {
	return region.Name
}

func (region *SRegion) GetName() string {
	return region.client.providerName
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_ZSTACK, region.client.providerID)
}

func (region *SRegion) IsEmulated() bool {
	return false
}

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_ZSTACK
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
	return self.GetInstance(id)
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
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
	params := []string{}
	if len(zoneId) > 0 {
		params = append(params, "q=uuid="+zoneId)
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

func (region *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
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
	vip, err := region.CreateVirtualIP(eip.Name, "", eip.IP, networkInfo[0])
	if err != nil {
		return nil, err
	}
	return region.CreateEip(eip.Name, vip.UUID, "")
}

func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	return region.GetEip(eipId)
}

func (region *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) DeleteSecurityGroup(vpcId, secGrpId string) error {
	return cloudprovider.ErrNotImplemented
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

func (region *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	offerings, err := region.GetInstanceOfferings("", "", 0, 0)
	if err != nil {
		return nil, err
	}
	iskus := []cloudprovider.ICloudSku{}
	for i := 0; i < len(offerings); i++ {
		iskus = append(iskus, &offerings[i])
	}
	return iskus, nil
}

func (region *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	if len(secgroupId) > 0 {
		_, err := region.GetSecurityGroup(secgroupId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				secgroupId = ""
			} else {
				return "", err
			}
		}
	}
	if len(secgroupId) == 0 {
		secgroup, err := region.CreateSecurityGroup(name, desc)
		if err != nil {
			return "", err
		}
		secgroupId = secgroup.UUID
	}
	return secgroupId, region.syncSecgroupRules(secgroupId, rules)
}
