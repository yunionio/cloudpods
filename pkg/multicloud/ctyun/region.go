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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion

	client       *SCtyunClient
	storageCache *SStoragecache

	Description    string `json:"description"`
	ID             string `json:"id"`
	ParentRegionID string `json:"parent_region_id"`
	Type           string `json:"type"`

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc
}

func (self *SRegion) fetchIVpcs() error {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return errors.Wrap(err, "SRegion.fetchIVpcs")
	}

	self.ivpcs = make([]cloudprovider.ICloudVpc, 0)
	for i := range vpcs {
		vpc := vpcs[i]
		vpc.region = self
		self.ivpcs = append(self.ivpcs, &vpc)
	}

	return nil
}

func (self *SRegion) fetchInfrastructure() error {
	if err := self.fetchIVpcs(); err != nil {
		return err
	}

	for i := 0; i < len(self.ivpcs); i += 1 {
		vpc := self.ivpcs[i].(*SVpc)
		wire := SWire{region: self, vpc: vpc}
		vpc.addWire(&wire)

		for j := 0; j < len(self.izones); j += 1 {
			zone := self.izones[j].(*SZone)
			zone.addWire(&wire)
		}

		vpc.fetchNetworks()
	}
	return nil
}

func (self *SRegion) GetVpcs() ([]SVpc, error) {
	vpcs := make([]SVpc, 0)
	params := map[string]string{
		"regionId": self.GetId(),
	}
	resp, err := self.client.DoGet("/apiproxy/v3/getVpcs", params)
	if err != nil {
		return nil, err
	}

	err = resp.Unmarshal(&vpcs, "returnObj")
	if err != nil {
		return nil, err
	}

	return vpcs, nil
}

func (self *SRegion) CreateVpc(name, cidr string) (*SVpc, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"name":     jsonutils.NewString(name),
		"cidr":     jsonutils.NewString(cidr),
	}
	resp, err := self.client.DoPost("/apiproxy/v3/createVPC", params)
	if err != nil {
		return nil, err
	}

	vpc := &SVpc{}
	err = resp.Unmarshal(vpc, "returnObj")
	if err != nil {
		return nil, err
	}

	vpc.region = self
	return vpc, nil
}

func (self *SRegion) GetClient() *SCtyunClient {
	return self.client
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return self.GetSecurityGroupDetails(secgroupId)
}

func (self *SRegion) GetISecurityGroupByName(vpcId string, name string) (cloudprovider.ICloudSecurityGroup, error) {
	segroups, err := self.GetSecurityGroups(vpcId)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetISecurityGroupByName.GetSecurityGroups")
	}

	for i := range segroups {
		if segroups[i].Name == name {
			return &segroups[i], nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "SRegion.GetISecurityGroupByName.GetSecurityGroups")
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetId() string {
	return self.ID
}

func (self *SRegion) GetName() string {
	return self.ID
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.client.GetAccessEnv(), self.ID)
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) Refresh() error {
	return nil
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.ID]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

// http://ctyun-api-url/apiproxy/v3/order/getZoneConfig
func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if self.izones == nil {
		var err error
		err = self.fetchInfrastructure()
		if err != nil {
			return nil, err
		}
	}
	return self.izones, nil
}

// http://ctyun-api-url/apiproxy/v3/getVpcs
// http://ctyun-api-url/apiproxy/v3/getVpcs
func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if self.ivpcs == nil {
		err := self.fetchInfrastructure()
		if err != nil {
			return nil, err
		}
	}
	return self.ivpcs, nil
}

// http://ctyun-api-url/apiproxy/v3/ondemand/queryIps
func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := self.GetEips()
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetIEips.GetEips")
	}

	ieips := make([]cloudprovider.ICloudEIP, len(eips))
	for i := range eips {
		ieips[i] = &eips[i]
	}

	return ieips, nil
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
	return self.GetEip(id)
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return self.GetVMById(id)
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
}

func (self *SRegion) DeleteSecurityGroup(vpcId, secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetISnapshots.GetSnapshots")
	}

	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := range snapshots {
		isnapshots[i] = &snapshots[i]
	}

	return isnapshots, nil
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetISnapshotById.GetSnapshots")
	}

	for i := range snapshots {
		if snapshots[i].GetId() == snapshotId {
			return &snapshots[i], nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "SRegion.GetISnapshotById")
}

func (self *SRegion) CreateSnapshotPolicy(*cloudprovider.SnapshotPolicyInput) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SRegion) UpdateSnapshotPolicy(*cloudprovider.SnapshotPolicyInput, string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) DeleteSnapshotPolicy(string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	polices, err := self.GetDiskBackupPolices()
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetISnapshotPolicies.GetDiskBackupPolices")
	}

	ipolices := make([]cloudprovider.ICloudSnapshotPolicy, len(polices))
	for i := range polices {
		ipolices[i] = &polices[i]
	}

	return ipolices, nil
}

func (self *SRegion) GetISnapshotPolicyById(snapshotPolicyId string) (cloudprovider.ICloudSnapshotPolicy, error) {
	return self.GetDiskBackupPolicy(snapshotPolicyId)
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
		} else if err != cloudprovider.ErrNotFound {
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
		} else if err != cloudprovider.ErrNotFound {
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

func (self *SRegion) GetInstances(instanceId string) ([]SInstance, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	if len(instanceId) > 0 {
		params["instanceId"] = instanceId
	}

	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryVMs", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetInstances.DoGet")
	}

	ret := make([]SInstance, 0)
	err = resp.Unmarshal(&ret, "returnObj", "servers")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetInstances.Unmarshal")
	}

	return ret, nil
}

func (self *SRegion) GetInstanceFlavors() ([]FlavorObj, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	resp, err := self.client.DoGet("/apiproxy/v3/order/getFlavors", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetInstanceFlavors.DoGet")
	}

	ret := make([]FlavorObj, 0)
	err = resp.Unmarshal(&ret, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetInstanceFlavors.Unmarshal")
	}

	return ret, nil
}
