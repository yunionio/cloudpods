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

package aliyun

import (
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion

	client    *SAliyunClient
	sdkClient *sdk.Client
	ossClient *oss.Client

	Debug bool

	RegionId  string
	LocalName string

	izones []cloudprovider.ICloudZone

	ivpcs []cloudprovider.ICloudVpc

	lbEndpints map[string]string

	storageCache *SStoragecache

	instanceTypes []SInstanceType

	latitude      float64
	longitude     float64
	fetchLocation bool
}

func (self *SRegion) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetClient() *SAliyunClient {
	return self.client
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) getSdkClient() (*sdk.Client, error) {
	if self.sdkClient == nil {
		cli, err := sdk.NewClientWithAccessKey(self.RegionId, self.client.accessKey, self.client.secret)
		if err != nil {
			return nil, err
		}
		self.sdkClient = cli
	}
	return self.sdkClient, nil
}

func (self *SRegion) getOSSExternalDomain() string {
	return getOSSExternalDomain(self.RegionId)
}

func (self *SRegion) getOSSInternalDomain() string {
	return getOSSInternalDomain(self.RegionId)
}

func (self *SRegion) GetOssClient() (*oss.Client, error) {
	if self.ossClient == nil {
		cli, err := self.client.getOssClient(self.RegionId)
		if err != nil {
			return nil, errors.Wrap(err, "self.client.getOssClient")
		}
		self.ossClient = cli
	}
	return self.ossClient, nil
}

func (self *SRegion) ecsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(client, "ecs.aliyuncs.com", ALIYUN_API_VERSION, apiName, params, self.client.Debug)
}

func (self *SRegion) rdsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(client, "rds.aliyuncs.com", ALIYUN_API_VERION_RDS, apiName, params, self.client.Debug)
}

func (self *SRegion) vpcRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(client, "vpc.aliyuncs.com", ALIYUN_API_VERSION_VPC, action, params, self.client.Debug)
}

func (self *SRegion) kvsRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(client, "r-kvstore.aliyuncs.com", ALIYUN_API_VERSION_KVS, action, params, self.client.Debug)
}

type LBRegion struct {
	RegionEndpoint string
	RegionId       string
}

func (self *SRegion) fetchLBRegions(client *sdk.Client) error {
	if len(self.lbEndpints) > 0 {
		return nil
	}
	params := map[string]string{}
	result, err := self._lbRequest(client, "DescribeRegions", "slb.aliyuncs.com", params)
	if err != nil {
		return err
	}
	self.lbEndpints = map[string]string{}
	regions := []LBRegion{}
	if err := result.Unmarshal(&regions, "Regions", "Region"); err != nil {
		return err
	}
	for _, region := range regions {
		self.lbEndpints[region.RegionId] = region.RegionEndpoint
	}
	return nil
}

func (self *SRegion) lbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	domain := "slb.aliyuncs.com"
	if !utils.IsInStringArray(apiName, []string{"DescribeRegions", "DescribeZones"}) {
		if regionId, ok := params["RegionId"]; ok {
			if err := self.fetchLBRegions(client); err != nil {
				return nil, err
			}
			endpoint, ok := self.lbEndpints[regionId]
			if !ok {
				return nil, fmt.Errorf("failed to find endpoint for lb region %s", regionId)
			}
			domain = endpoint
		}
	}
	return self._lbRequest(client, apiName, domain, params)
}

func (self *SRegion) _lbRequest(client *sdk.Client, apiName string, domain string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(client, domain, ALIYUN_API_VERSION_LB, apiName, params, self.Debug)
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) GetId() string {
	return self.RegionId
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_ALIYUN_CN, self.LocalName)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_ALIYUN, self.RegionId)
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_ALIYUN
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.RegionId]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) Refresh() error {
	// do nothing
	return nil
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

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) _fetchZones(chargeType TChargeType, spotStrategy SpotStrategyType) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	if len(chargeType) > 0 {
		params["InstanceChargeType"] = string(chargeType)
	}
	if len(spotStrategy) > 0 {
		params["SpotStrategy"] = string(spotStrategy)
	}
	body, err := self.ecsRequest("DescribeZones", params)
	if err != nil {
		return err
	}

	zones := make([]SZone, 0)
	err = body.Unmarshal(&zones, "Zones", "Zone")
	if err != nil {
		return err
	}

	self.izones = make([]cloudprovider.ICloudZone, len(zones))

	for i := 0; i < len(zones); i += 1 {
		zones[i].region = self
		self.izones[i] = &zones[i]
	}

	return nil
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		zone := izones[i].(*SZone)
		if zone.ZoneId == id {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return self.GetInstance(id)
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.getDisk(id)
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

func (self *SRegion) fetchIVpcs() error {
	vpcs := make([]SVpc, 0)
	for {
		part, total, err := self.GetVpcs(nil, len(vpcs), 50)
		if err != nil {
			return err
		}
		vpcs = append(vpcs, part...)
		if len(vpcs) >= total {
			break
		}
	}
	self.ivpcs = make([]cloudprovider.ICloudVpc, len(vpcs))
	for i := 0; i < len(vpcs); i += 1 {
		vpcs[i].region = self
		self.ivpcs[i] = &vpcs[i]
	}
	return nil
}

func (self *SRegion) fetchInfrastructure() error {
	err := self._fetchZones(PostPaidInstanceChargeType, NoSpotStrategy)
	if err != nil {
		return err
	}
	err = self.fetchIVpcs()
	if err != nil {
		return err
	}
	for i := 0; i < len(self.ivpcs); i += 1 {
		for j := 0; j < len(self.izones); j += 1 {
			zone := self.izones[j].(*SZone)
			vpc := self.ivpcs[i].(*SVpc)
			wire := SWire{zone: zone, vpc: vpc}
			zone.addWire(&wire)
			vpc.addWire(&wire)
		}
	}
	return nil
}

func (self *SRegion) GetVpcs(vpcId []string, offset int, limit int) ([]SVpc, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	if vpcId != nil && len(vpcId) > 0 {
		params["VpcId"] = strings.Join(vpcId, ",")
	}

	body, err := self.ecsRequest("DescribeVpcs", params)
	if err != nil {
		log.Errorf("GetVpcs fail %s", err)
		return nil, 0, err
	}

	vpcs := make([]SVpc, 0)
	err = body.Unmarshal(&vpcs, "Vpcs", "Vpc")
	if err != nil {
		log.Errorf("Unmarshal vpc fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return vpcs, int(total), nil
}

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	vpcs, total, err := self.GetVpcs([]string{vpcId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	vpcs[0].region = self
	return &vpcs[0], nil
}

func (self *SRegion) GetVRouters(offset int, limit int) ([]SVRouter, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	body, err := self.ecsRequest("DescribeVRouters", params)
	if err != nil {
		log.Errorf("GetVRouters fail %s", err)
		return nil, 0, err
	}

	vrouters := make([]SVRouter, 0)
	err = body.Unmarshal(&vrouters, "VRouters", "VRouter")
	if err != nil {
		log.Errorf("Unmarshal vrouter fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return vrouters, int(total), nil
}

func (self *SRegion) GetRouteTables(ids []string, offset int, limit int) ([]SRouteTable, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	if ids != nil && len(ids) > 0 {
		params["RouteTableId"] = strings.Join(ids, ",")
	}

	body, err := self.ecsRequest("DescribeRouteTables", params)
	if err != nil {
		log.Errorf("GetRouteTables fail %s", err)
		return nil, 0, err
	}

	routetables := make([]SRouteTable, 0)
	err = body.Unmarshal(&routetables, "RouteTables", "RouteTable")
	if err != nil {
		log.Errorf("Unmarshal routetables fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return routetables, int(total), nil
}

func (self *SRegion) GetMatchInstanceTypes(cpu int, memMB int, gpu int, zoneId string) ([]SInstanceType, error) {
	if self.instanceTypes == nil {
		types, err := self.GetInstanceTypes()
		if err != nil {
			log.Errorf("GetInstanceTypes %s", err)
			return nil, err
		}
		self.instanceTypes = types
	}
	var available []string
	if len(zoneId) > 0 {
		zone, err := self.getZoneById(zoneId)
		if err != nil {
			return nil, err
		}
		available = zone.AvailableInstanceTypes.InstanceTypes
	}
	ret := make([]SInstanceType, 0)
	for _, t := range self.instanceTypes {
		if t.CpuCoreCount == cpu && memMB == t.memoryMB() && gpu == t.GPUAmount {
			if available == nil || utils.IsInStringArray(t.InstanceTypeId, available) {
				ret = append(ret, t)
			}
		}
	}
	return ret, nil
}

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, vswitchId string, passwd string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.LocalName)
		net := z.getNetworkById(vswitchId)
		if net != nil {
			desc := &cloudprovider.SManagedVMCreateConfig{
				Name:              name,
				ExternalImageId:   imgId,
				SysDisk:           cloudprovider.SDiskInfo{SizeGB: 0, StorageType: storageType},
				Cpu:               cpu,
				MemoryMB:          memGB * 1024,
				ExternalNetworkId: vswitchId,
				Password:          passwd,
				DataDisks:         []cloudprovider.SDiskInfo{},
				PublicKey:         publicKey,
			}
			for _, sizeGB := range dataDiskSizesGB {
				desc.DataDisks = append(desc.DataDisks, cloudprovider.SDiskInfo{SizeGB: sizeGB, StorageType: storageType})
			}
			inst, err := z.getHost().CreateVM(desc)
			if err != nil {
				return nil, err
			}
			return inst.(*SInstance), nil
		}
	}
	return nil, fmt.Errorf("cannot find vswitch %s", vswitchId)
}

func (self *SRegion) instanceOperation(instanceId string, opname string, extra map[string]string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	if extra != nil && len(extra) > 0 {
		for k, v := range extra {
			params[k] = v
		}
	}
	_, err := self.ecsRequest(opname, params)
	return err
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.Status, nil
}

func (self *SRegion) GetInstanceVNCUrl(instanceId string) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	body, err := self.ecsRequest("DescribeInstanceVncUrl", params)
	if err != nil {
		return "", err
	}
	return body.GetString("VncUrl")
}

func (self *SRegion) ModifyInstanceVNCUrlPassword(instanceId string, passwd string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	params["VncPassword"] = passwd // must be 6 digital + alphabet
	_, err := self.ecsRequest("ModifyInstanceVncPasswd", params)
	return err
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	params := make(map[string]string)
	if len(cidr) > 0 {
		params["CidrBlock"] = cidr
	}
	if len(name) > 0 {
		params["VpcName"] = name
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	params["ClientToken"] = utils.GenRequestId(20)
	body, err := self.ecsRequest("CreateVpc", params)
	if err != nil {
		return nil, err
	}
	vpcId, err := body.GetString("VpcId")
	if err != nil {
		return nil, err
	}
	err = self.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	return self.GetIVpcById(vpcId)
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := make(map[string]string)
	params["VpcId"] = vpcId

	_, err := self.ecsRequest("DeleteVpc", params)
	return err
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

func (self *SRegion) updateInstance(instId string, name, desc, passwd, hostname, userData string) error {
	params := make(map[string]string)
	params["InstanceId"] = instId
	if len(name) > 0 {
		params["InstanceName"] = name
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	if len(passwd) > 0 {
		params["Password"] = passwd
	}
	if len(hostname) > 0 {
		params["HostName"] = hostname
	}
	if len(userData) > 0 {
		params["UserData"] = userData
	}
	_, err := self.ecsRequest("ModifyInstanceAttribute", params)
	return err
}

func (self *SRegion) UpdateInstancePassword(instId string, passwd string) error {
	return self.updateInstance(instId, "", "", passwd, "", "")
}

// func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
// 	eips, total, err := self.GetSnapshots("", 0, 50)
// 	if err != nil {
// 		return nil, err
// 	}
// 	for len(eips) < total {
// 		var parts []SEipAddress
// 		parts, total, err = self.GetEips("", len(eips), 50)
// 		if err != nil {
// 			return nil, err
// 		}
// 		eips = append(eips, parts...)
// 	}
// 	ret := make([]cloudprovider.ICloudEIP, len(eips))
// 	for i := 0; i < len(eips); i += 1 {
// 		ret[i] = &eips[i]
// 	}
// 	return ret, nil
// }

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, total, err := self.GetEips("", "", 0, 50)
	if err != nil {
		return nil, err
	}
	for len(eips) < total {
		var parts []SEipAddress
		parts, total, err = self.GetEips("", "", len(eips), 50)
		if err != nil {
			return nil, err
		}
		eips = append(eips, parts...)
	}
	ret := make([]cloudprovider.ICloudEIP, len(eips))
	for i := 0; i < len(eips); i += 1 {
		ret[i] = &eips[i]
	}
	return ret, nil
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	eips, total, err := self.GetEips(eipId, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &eips[0], nil
}

func (region *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	secgroup, err := region.GetSecurityGroupDetails(secgroupId)
	if err != nil {
		return nil, err
	}
	vpc, err := region.getVpc(secgroup.VpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "region.getVpc(%s)", secgroup.VpcId)
	}
	secgroup.vpc = vpc
	return secgroup, nil
}

func (region *SRegion) GetISecurityGroupByName(vpcId string, name string) (cloudprovider.ICloudSecurityGroup, error) {
	secgroups, total, err := region.GetSecurityGroups(vpcId, name, []string{}, 0, 0)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &secgroups[0], nil
}

func (region *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	externalId, err := region.CreateSecurityGroup(conf.VpcId, conf.Name, conf.Desc)
	if err != nil {
		return nil, err
	}
	return region.GetISecurityGroupById(externalId)
}

func (region *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	if len(secgroupId) > 0 {
		_, total, err := region.GetSecurityGroups("", "", []string{secgroupId}, 0, 1)
		if err != nil {
			return "", err
		}
		if total == 0 {
			secgroupId = ""
		}
	}
	if len(secgroupId) == 0 {
		extID, err := region.CreateSecurityGroup(vpcId, name, desc)
		if err != nil {
			return "", err
		}
		secgroupId = extID
	}
	return secgroupId, region.syncSecgroupRules(secgroupId, rules)
}

func (region *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	lbs, err := region.GetLoadbalancers(nil)
	if err != nil {
		return nil, err
	}
	ilbs := []cloudprovider.ICloudLoadbalancer{}
	for i := 0; i < len(lbs); i++ {
		lbs[i].region = region
		ilbs = append(ilbs, &lbs[i])
	}
	return ilbs, nil
}

func (region *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return region.GetLoadbalancerDetail(loadbalancerId)
}

func (region *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	certs, err := region.GetLoadbalancerServerCertificates()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(certs); i++ {
		if certs[i].GetGlobalId() == certId {
			certs[i].region = region
			return &certs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["ServerCertificateName"] = cert.Name
	params["PrivateKey"] = cert.PrivateKey
	params["ServerCertificate"] = cert.Certificate
	body, err := region.lbRequest("UploadServerCertificate", params)
	if err != nil {
		return nil, err
	}
	certID, err := body.GetString("ServerCertificateId")
	if err != nil {
		return nil, err
	}
	return region.GetILoadBalancerCertificateById(certID)
}

func (region *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return region.GetLoadbalancerAclDetail(aclId)
}

func (region *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	acls, err := region.GetLoadBalancerAcls()
	if err != nil {
		return nil, err
	}
	iAcls := []cloudprovider.ICloudLoadbalancerAcl{}
	for i := 0; i < len(acls); i++ {
		acls[i].region = region
		iAcls = append(iAcls, &acls[i])
	}
	return iAcls, nil
}

func (region *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	certificates, err := region.GetLoadbalancerServerCertificates()
	if err != nil {
		return nil, err
	}
	iCertificates := []cloudprovider.ICloudLoadbalancerCertificate{}
	for i := 0; i < len(certificates); i++ {
		certificates[i].region = region
		iCertificates = append(iCertificates, &certificates[i])
	}
	return iCertificates, nil
}

func (region *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerName"] = loadbalancer.Name
	if len(loadbalancer.ZoneID) > 0 {
		params["MasterZoneId"] = loadbalancer.ZoneID
	}

	if len(loadbalancer.VpcID) > 0 {
		params["VpcId"] = loadbalancer.VpcID
	}

	if len(loadbalancer.NetworkIDs) > 0 {
		params["VSwitchId"] = loadbalancer.NetworkIDs[0]
	}

	if len(loadbalancer.Address) > 0 {
		params["Address"] = loadbalancer.Address
	}

	if len(loadbalancer.AddressType) > 0 {
		params["AddressType"] = loadbalancer.AddressType
	}

	if len(loadbalancer.LoadbalancerSpec) > 0 {
		params["LoadBalancerSpec"] = loadbalancer.LoadbalancerSpec
	}

	if len(loadbalancer.ChargeType) > 0 {
		params["InternetChargeType"] = "payby" + loadbalancer.ChargeType
	}

	if loadbalancer.ChargeType == api.LB_CHARGE_TYPE_BY_BANDWIDTH && loadbalancer.EgressMbps > 0 {
		params["Bandwidth"] = fmt.Sprintf("%d", loadbalancer.EgressMbps)
	}

	body, err := region.lbRequest("CreateLoadBalancer", params)
	if err != nil {
		return nil, err
	}
	loadBalancerID, err := body.GetString("LoadBalancerId")
	if err != nil {
		return nil, err
	}
	iLoadbalancer, err := region.GetLoadbalancerDetail(loadBalancerID)
	if err != nil {
		return nil, err
	}
	return iLoadbalancer, cloudprovider.WaitStatus(iLoadbalancer, api.LB_STATUS_ENABLED, time.Second*5, time.Minute*5)
}

func (region *SRegion) AddAccessControlListEntry(aclId string, entrys []cloudprovider.SLoadbalancerAccessControlListEntry) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["AclId"] = aclId
	aclArray := jsonutils.NewArray()
	for i := 0; i < len(entrys); i++ {
		//阿里云AclEntrys参数必须是CIDR格式的。
		if regutils.MatchIPAddr(entrys[i].CIDR) {
			entrys[i].CIDR += "/32"
		}
		aclArray.Add(jsonutils.Marshal(map[string]string{"entry": entrys[i].CIDR, "comment": entrys[i].Comment}))
	}
	if aclArray.Length() == 0 {
		return nil
	}
	params["AclEntrys"] = aclArray.String()
	_, err := region.lbRequest("AddAccessControlListEntry", params)
	return err
}

func (region *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["AclName"] = acl.Name
	body, err := region.lbRequest("CreateAccessControlList", params)
	if err != nil {
		return nil, err
	}
	aclId, err := body.GetString("AclId")
	if err != nil {
		return nil, err
	}
	iAcl, err := region.GetLoadbalancerAclDetail(aclId)
	if err != nil {
		return nil, err
	}
	return iAcl, region.AddAccessControlListEntry(aclId, acl.Entrys)
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "getIBuckets")
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range iBuckets {
		loc := iBuckets[i].GetLocation()
		// remove oss- prefix
		if loc[4:] != region.GetId() {
			continue
		}
		ret = append(ret, iBuckets[i])
	}
	return ret, nil
}

func str2StorageClass(storageClassStr string) (oss.StorageClassType, error) {
	storageClass := oss.StorageStandard
	if strings.EqualFold(storageClassStr, string(oss.StorageStandard)) {
		//
	} else if strings.EqualFold(storageClassStr, string(oss.StorageIA)) {
		storageClass = oss.StorageIA
	} else if strings.EqualFold(storageClassStr, string(oss.StorageArchive)) {
		storageClass = oss.StorageArchive
	} else {
		return storageClass, errors.Error("not supported storageClass")
	}
	return storageClass, nil
}

func str2Acl(aclStr string) (oss.ACLType, error) {
	acl := oss.ACLPrivate
	if strings.EqualFold(aclStr, string(oss.ACLPrivate)) {
		// private, default
	} else if strings.EqualFold(aclStr, string(oss.ACLPublicRead)) {
		acl = oss.ACLPublicRead
	} else if strings.EqualFold(aclStr, string(oss.ACLPublicReadWrite)) {
		acl = oss.ACLPublicReadWrite
	} else {
		return acl, errors.Error("not supported acl")
	}
	return acl, nil
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	osscli, err := region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "region.GetOssClient")
	}
	opts := make([]oss.Option, 0)
	if len(storageClassStr) > 0 {
		storageClass, err := str2StorageClass(storageClassStr)
		if err != nil {
			return err
		}
		opts = append(opts, oss.StorageClass(storageClass))
	}
	if len(aclStr) > 0 {
		acl, err := str2Acl(aclStr)
		if err != nil {
			return err
		}
		opts = append(opts, oss.ACL(acl))
	}
	err = osscli.CreateBucket(name, opts...)
	if err != nil {
		return errors.Wrap(err, "oss.CreateBucket")
	}
	region.client.invalidateIBuckets()
	return nil
}

func ossErrorCode(err error) int {
	if srvErr, ok := err.(oss.ServiceError); ok {
		return srvErr.StatusCode
	}
	if srvErr, ok := err.(*oss.ServiceError); ok {
		return srvErr.StatusCode
	}
	return -1
}

func (region *SRegion) DeleteIBucket(name string) error {
	osscli, err := region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "region.GetOssClient")
	}
	err = osscli.DeleteBucket(name)
	if err != nil {
		if ossErrorCode(err) == 404 {
			return nil
		}
		return errors.Wrap(err, "DeleteBucket")
	}
	region.client.invalidateIBuckets()
	return nil
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	osscli, err := region.GetOssClient()
	if err != nil {
		return false, errors.Wrap(err, "region.GetOssClient")
	}
	exist, err := osscli.IsBucketExist(name)
	if err != nil {
		return false, errors.Wrap(err, "IsBucketExist")
	}
	return exist, nil
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	osscli, err := region.GetOssClient()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetOssClient")
	}
	bi, err := osscli.GetBucketInfo(name)
	if err != nil {
		return nil, errors.Wrap(err, "Bucket")
	}
	bInfo := bi.BucketInfo
	b := SBucket{
		region:       region,
		Name:         bInfo.Name,
		Location:     bInfo.Location,
		CreationDate: bInfo.CreationDate,
		StorageClass: bInfo.StorageClass,
	}
	return &b, nil
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	caches, err := self.GetElasticCaches(nil)
	if err != nil {
		return nil, err
	}

	icaches := make([]cloudprovider.ICloudElasticcache, len(caches))
	for i := range caches {
		caches[i].region = self
		icaches[i] = &caches[i]
	}

	return icaches, nil
}
