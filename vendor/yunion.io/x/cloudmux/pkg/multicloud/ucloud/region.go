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

package ucloud

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SNoLbRegion
	client *SUcloudClient

	RegionID string

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	latitude      float64
	longitude     float64
	fetchLocation bool
}

func (self *SRegion) GetId() string {
	return self.RegionID
}

func (self *SRegion) GetName() string {
	if name, exist := UCLOUD_REGION_NAMES[self.GetId()]; exist {
		return fmt.Sprintf("%s %s", CLOUD_PROVIDER_UCLOUD_CN, name)
	}

	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_UCLOUD_CN, self.GetId())
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	var en string
	if name, exist := UCLOUD_REGION_NAMES_EN[self.GetId()]; exist {
		en = fmt.Sprintf("%s %s", CLOUD_PROVIDER_UCLOUD, name)
	} else {
		en = fmt.Sprintf("%s %s", CLOUD_PROVIDER_UCLOUD, self.GetId())
	}

	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_UCLOUD, self.GetId())
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

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.GetId()]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	instance, err := self.GetInstanceByID(id)
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
}

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

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if self.ivpcs == nil {
		err := self.fetchInfrastructure()
		if err != nil {
			return nil, err
		}
	}
	return self.ivpcs, nil
}

// https://docs.ucloud.cn/api/unet-api/describe_eip
func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	params := NewUcloudParams()
	eips := make([]SEip, 0)
	err := self.DoListAll("DescribeEIP", params, &eips)
	if err != nil {
		return nil, err
	}

	ieips := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eip := eips[i]
		eip.region = self
		ieips = append(ieips, &eip)
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

func (self *SRegion) GetEipById(eipId string) (SEip, error) {
	params := NewUcloudParams()
	params.Set("EIPIds.0", eipId)
	eips := make([]SEip, 0)
	err := self.DoListAll("DescribeEIP", params, &eips)
	if err != nil {
		return SEip{}, err
	}

	if len(eips) == 1 {
		eip := eips[0]
		eip.region = self
		return eip, nil
	} else if len(eips) == 0 {
		return SEip{}, cloudprovider.ErrNotFound
	} else {
		return SEip{}, fmt.Errorf("GetEipById %d eip found", len(eips))
	}
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEipById(id)
	return &eip, err
}

// https://docs.ucloud.cn/api/unet-api/delete_firewall
func (self *SRegion) DeleteSecurityGroup(secgroupId string) error {
	params := NewUcloudParams()
	params.Set("FWId", secgroupId)
	return self.DoAction("DeleteFirewall", params, nil)
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

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return self.GetSecurityGroup(secgroupId)
}

func (self *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	externalId, err := self.CreateSecurityGroup(opts.Name, opts.Desc)
	if err != nil {
		return nil, err
	}
	return self.GetISecurityGroupById(externalId)
}

// https://docs.ucloud.cn/api/unet-api/describe_firewall
// 绑定防火墙组的资源类型，默认为全部资源类型。枚举值为："unatgw"，NAT网关； "uhost"，云主机； "upm"，物理云主机； "hadoophost"，hadoop节点； "fortresshost"，堡垒机； "udhost"，私有专区主机；"udockhost"，容器；"dbaudit"，数据库审计.
// todo: 是否需要过滤出仅绑定云主机的安全组？

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	params := NewUcloudParams()
	params.Set("Name", opts.NAME)
	params.Set("Remark", opts.Desc)
	for i, cidr := range strings.Split(opts.CIDR, ",") {
		params.Set(fmt.Sprintf("Network.%d", i), cidr)
	}

	vpcId := ""
	err := self.DoAction("CreateVPC", params, &vpcId)
	if err != nil {
		return nil, err
	}

	return self.GetIVpcById(vpcId)
}

// https://docs.ucloud.cn/api/udisk-api/describe_udisk_snapshot
func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	params := NewUcloudParams()
	snapshots := make([]SSnapshot, 0)
	err := self.DoListAll("DescribeUDiskSnapshot", params, &snapshots)
	if err != nil {
		return nil, err
	}

	isnapshots := make([]cloudprovider.ICloudSnapshot, 0)
	for i := range snapshots {
		snapshots[i].region = self
		isnapshots = append(isnapshots, &snapshots[i])
	}

	return isnapshots, nil
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	if len(snapshotId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}

	params := NewUcloudParams()
	snapshots := make([]SSnapshot, 0)
	err := self.DoListAll("DescribeUDiskSnapshot", params, &snapshots)
	if err != nil {
		return nil, err
	}

	for i := range snapshots {
		if snapshots[i].SnapshotID == snapshotId {
			snapshot := snapshots[i]
			snapshot.region = self
			return &snapshot, nil
		}
	}

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

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
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

func (self *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_UCLOUD
}

func (self *SRegion) GetCloudEnv() string {
	return ""
}

func (self *SRegion) DoListAll(action string, params SParams, result interface{}) error {
	params.Set("Region", self.GetId())
	return self.client.DoListAll(action, params, result)
}

// return total,lenght,error
func (self *SRegion) DoListPart(action string, limit int, offset int, params SParams, result interface{}) (int, int, error) {
	params.Set("Region", self.GetId())
	return self.client.DoListPart(action, limit, offset, params, result)
}

func (self *SRegion) DoAction(action string, params SParams, result interface{}) error {
	params.Set("Region", self.GetId())
	return self.client.DoAction(action, params, result)
}

func (self *SRegion) fetchInfrastructure() error {
	if err := self.fetchZones(); err != nil {
		return err
	}

	if err := self.fetchIVpcs(); err != nil {
		return err
	}

	for i := 0; i < len(self.ivpcs); i += 1 {
		vpc := self.ivpcs[i].(*SVPC)
		wire := SWire{region: self, vpc: vpc}
		vpc.addWire(&wire)

		for j := 0; j < len(self.izones); j += 1 {
			zone := self.izones[j].(*SZone)
			zone.addWire(&wire)
		}
	}
	return nil
}

func (self *SRegion) fetchZones() error {
	type Region struct {
		RegionID   int64  `json:"RegionId"`
		RegionName string `json:"RegionName"`
		IsDefault  bool   `json:"IsDefault"`
		BitMaps    string `json:"BitMaps"`
		Region     string `json:"Region"`
		Zone       string `json:"Zone"`
	}

	params := NewUcloudParams()
	regions := make([]Region, 0)
	err := self.client.DoListAll("GetRegion", params, &regions)
	if err != nil {
		return err
	}

	for _, r := range regions {
		if r.Region != self.GetId() {
			continue
		}

		szone := SZone{}
		szone.ZoneId = r.Zone
		szone.RegionId = r.Region
		szone.region = self
		self.izones = append(self.izones, &szone)
	}

	return nil
}

func (self *SRegion) fetchIVpcs() error {
	vpcs := make([]SVPC, 0)
	params := NewUcloudParams()
	err := self.DoListAll("DescribeVPC", params, &vpcs)
	if err != nil {
		return err
	}

	for i := range vpcs {
		vpc := vpcs[i]
		vpc.region = self
		self.ivpcs = append(self.ivpcs, &vpc)
	}

	return nil
}

// https://docs.ucloud.cn/api/uhost-api/describe_uhost_instance
func (self *SRegion) GetInstanceByID(instanceId string) (SInstance, error) {
	params := NewUcloudParams()
	params.Set("UHostIds.0", instanceId)
	instances := make([]SInstance, 0)
	err := self.DoAction("DescribeUHostInstance", params, &instances)
	if err != nil {
		return SInstance{}, err
	}

	if len(instances) == 1 {
		return instances[0], nil
	} else if len(instances) == 0 {
		return SInstance{}, cloudprovider.ErrNotFound
	} else {
		return SInstance{}, fmt.Errorf("GetInstanceByID %s %d found.", instanceId, len(instances))
	}
}

func (self *SRegion) GetClient() *SUcloudClient {
	return self.client
}

// https://docs.ucloud.cn/api/ufile-api/describe_bucket
func (client *SUcloudClient) listBuckets(name string, offset int, limit int) ([]SBucket, error) {
	params := NewUcloudParams()
	if len(name) > 0 {
		params.Set("BucketName", name)
	} else {
		params.Set("Limit", limit)
		params.Set("Offset", offset)
	}
	buckets := make([]SBucket, 0)
	// request without RegionId
	err := client.DoAction("DescribeBucket", params, &buckets)
	if err != nil {
		return nil, errors.Wrap(err, "DoAction DescribeBucket")
	}
	return buckets, nil
}

// https://docs.ucloud.cn/api/ufile-api/update_bucket
func (region *SRegion) updateBucket(name string, aclType string) error {
	params := NewUcloudParams()
	params.Set("BucketName", name)
	params.Set("ProjectId", region.client.projectId)
	params.Set("Type", aclType)
	return region.client.DoAction("UpdateBucket", params, nil)
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "getIBuckets")
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range iBuckets {
		if iBuckets[i].GetLocation() != region.GetId() {
			continue
		}
		ret = append(ret, iBuckets[i])
	}
	return ret, nil
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	if aclStr != "private" && aclStr != "public" {
		return errors.Error("invalid acl")
	}
	return region.CreateBucket(name, aclStr)
}

func (region *SRegion) DeleteIBucket(name string) error {
	err := region.DeleteBucket(name)
	if err != nil {
		if strings.Index(err.Error(), "bucket not found") >= 0 {
			return nil
		}
		return errors.Wrap(err, "region.DeleteBucket")
	}
	return nil
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	parts, err := region.client.listBuckets(name, 0, 1)
	if err != nil {
		return false, errors.Wrap(err, "region.listBuckets")
	}
	if len(parts) == 0 {
		return false, cloudprovider.ErrNotFound
	}
	return true, nil
}

func (region *SRegion) GetIBucketById(bucketId string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(region, bucketId)
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketByName(name)
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := region.GetInstances("", "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		ret = append(ret, &vms[i])
	}
	return ret, nil
}
