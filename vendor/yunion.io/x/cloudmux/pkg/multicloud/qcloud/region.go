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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion

	client *SQcloudClient

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	instanceTypes []SInstanceType

	Region      string
	RegionName  string
	RegionState string

	Latitude      float64
	Longitude     float64
	fetchLocation bool
}

// 腾讯云不支持acl
func (self *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return []cloudprovider.ICloudLoadbalancerAcl{}, nil
}

func (self *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetId() string {
	return self.Region
}

func (self *SRegion) GetName() string {
	return self.RegionName
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := self.RegionName
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_QCLOUD, self.Region)
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_QCLOUD
}

func (self *SRegion) GetCloudEnv() string {
	return ""
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	params := make(map[string]string)
	if len(opts.CIDR) > 0 {
		params["CidrBlock"] = opts.CIDR
	}
	if len(opts.NAME) > 0 {
		params["VpcName"] = opts.NAME
	}
	body, err := self.vpcRequest("CreateVpc", params)
	if err != nil {
		return nil, err
	}
	vpcId, err := body.GetString("Vpc", "VpcId")
	if err != nil {
		return nil, err
	}
	err = self.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	return self.GetIVpcById(vpcId)
}

func (self *SRegion) GetCosClient(bucket *SBucket) (*cos.Client, error) {
	return self.client.getCosClient(bucket)
}

func (self *SRegion) GetClient() *SQcloudClient {
	return self.client
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return self.GetInstance(id)
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	if len(eipId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
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
	for i := 0; i < len(eips); i++ {
		ret[i] = &eips[i]
	}
	return ret, nil
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

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return self.storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (self *SRegion) updateInstance(instId string, name, desc, passwd, hostname string) error {
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
	_, err := self.cvmRequest("ModifyInstanceAttribute", params, true)
	return err
}

func (self *SRegion) UpdateInstancePassword(instId string, passwd string) error {
	return self.updateInstance(instId, "", "", passwd, "")
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(ivpcs); i++ {
		if ivpcs[i].GetGlobalId() == id {
			return ivpcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		zone := izones[i].(*SZone)
		if zone.Zone == id {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
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

func (self *SRegion) _fetchZones() error {
	params := make(map[string]string)
	zones := make([]SZone, 0)
	body, err := self.cvmRequest("DescribeZones", params, true)
	if err != nil {
		return err
	}
	err = body.Unmarshal(&zones, "ZoneSet")
	if err != nil {
		return err
	}
	self.izones = make([]cloudprovider.ICloudZone, len(zones))
	for i := 0; i < len(zones); i++ {
		zones[i].region = self
		self.izones[i] = &zones[i]
	}
	return nil
}

func (self *SRegion) fetchInfrastructure() error {
	err := self._fetchZones()
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

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := make(map[string]string)
	params["VpcId"] = vpcId

	_, err := self.vpcRequest("DeleteVpc", params)
	return err
}

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	vpcs, total, err := self.GetVpcs([]string{vpcId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	vpcs[0].region = self
	return &vpcs[0], nil
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

func (self *SRegion) GetVpcs(vpcIds []string, offset int, limit int) ([]SVpc, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	if vpcIds != nil && len(vpcIds) > 0 {
		for index, vpcId := range vpcIds {
			params[fmt.Sprintf("VpcIds.%d", index)] = vpcId
		}
	}
	body, err := self.vpcRequest("DescribeVpcs", params)
	if err != nil {
		return nil, 0, err
	}
	vpcs := make([]SVpc, 0)
	err = body.Unmarshal(&vpcs, "VpcSet")
	if err != nil {
		log.Errorf("Unmarshal vpc fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return vpcs, int(total), nil
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.Region]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetStatus() string {
	if self.RegionState == "AVAILABLE" {
		return api.CLOUD_REGION_STATUS_INSERVER
	}
	return api.CLOUD_REGION_STATUS_OUTOFSERVICE
}

func (self *SRegion) Refresh() error {
	// do nothing
	return nil
}

// 容器
func (self *SRegion) tkeRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.tkeRequest(apiName, params)
}

func (self *SRegion) vpcRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.vpcRequest(apiName, params)
}

func (self *SRegion) auditRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.auditRequest(apiName, params)
}

func (self *SRegion) cvmRequest(apiName string, params map[string]string, retry bool) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.jsonRequest(apiName, params, retry)
}

func (self *SRegion) cbsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.cbsRequest(apiName, params)
}

func (self *SRegion) clbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.clbRequest(apiName, params)
}

func (self *SRegion) cdbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.cdbRequest(apiName, params)
}

func (self *SRegion) mariadbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.mariadbRequest(apiName, params)
}

func (self *SRegion) postgresRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.postgresRequest(apiName, params)
}

func (self *SRegion) sqlserverRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.sqlserverRequest(apiName, params)
}

func (self *SRegion) sslRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return self.client.sslRequest(apiName, params)
}

func (self *SRegion) kafkaRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.kafkaRequest(apiName, params)
}

func (self *SRegion) redisRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.redisRequest(apiName, params)
}

func (self *SRegion) dcdbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.dcdbRequest(apiName, params)
}

func (self *SRegion) mongodbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.mongodbRequest(apiName, params)
}

// Elasticsearch
func (self *SRegion) esRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.esRequest(apiName, params)
}

func (self *SRegion) memcachedRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.memcachedRequest(apiName, params)
}

func (self *SRegion) GetNetworks(ids []string, vpcId string, offset int, limit int) ([]SNetwork, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	base := 0
	if ids != nil && len(ids) > 0 {
		for index, networkId := range ids {
			params[fmt.Sprintf("SubnetIds.%d", index)] = networkId
		}
		base += len(ids)
	}
	if len(vpcId) > 0 {
		params["Filters.0.Name"] = "vpc-id"
		params["Filters.0.Values.0"] = vpcId
	}

	body, err := self.vpcRequest("DescribeSubnets", params)
	if err != nil {
		log.Errorf("DescribeSubnets fail %s", err)
		return nil, 0, err
	}

	networks := make([]SNetwork, 0)
	err = body.Unmarshal(&networks, "SubnetSet")
	if err != nil {
		log.Errorf("Unmarshal network fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return networks, int(total), nil
}

func (self *SRegion) GetNetwork(networkId string) (*SNetwork, error) {
	networks, total, err := self.GetNetworks([]string{networkId}, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &networks[0], nil
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
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
		available = zone.getAvaliableInstanceTypes()
	}
	ret := make([]SInstanceType, 0)
	for _, t := range self.instanceTypes {
		if t.CPU == cpu && memMB == t.memoryMB() && gpu == t.GPU {
			if available == nil || utils.IsInStringArray(t.InstanceType, available) {
				ret = append(ret, t)
			}
		}
	}
	return ret, nil
}

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, networkId string, passwd string, publicKey string, secgroup string, tags map[string]string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.Zone)
		net := z.getNetworkById(networkId)
		if net != nil {
			desc := &cloudprovider.SManagedVMCreateConfig{
				Name:              name,
				ExternalImageId:   imgId,
				SysDisk:           cloudprovider.SDiskInfo{SizeGB: 0, StorageType: storageType},
				Cpu:               cpu,
				MemoryMB:          memGB * 1024,
				ExternalNetworkId: networkId,
				Password:          passwd,
				DataDisks:         []cloudprovider.SDiskInfo{},
				PublicKey:         publicKey,

				Tags: tags,

				ExternalSecgroupIds: []string{secgroup},
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
	return nil, fmt.Errorf("cannot find network %s", networkId)
}

func (self *SRegion) instanceOperation(instanceId string, opname string, extra map[string]string, retry bool) error {
	params := make(map[string]string)
	params["InstanceIds.0"] = instanceId
	if extra != nil && len(extra) > 0 {
		for k, v := range extra {
			params[k] = v
		}
	}
	_, err := self.cvmRequest(opname, params, retry)
	return err
}

func (self *SRegion) GetInstanceVNCUrl(instanceId string) (string, error) {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	body, err := self.cvmRequest("DescribeInstanceVncUrl", params, true)
	if err != nil {
		return "", err
	}
	return body.GetString("InstanceVncUrl")
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.InstanceState, nil
}

func (self *SRegion) QueryAccountBalance() (*SAccountBalance, error) {
	return self.client.QueryAccountBalance()
}

func (self *SRegion) getCosEndpoint() string {
	return fmt.Sprintf("cos.%s.myqcloud.com", self.GetId())
}

func (self *SRegion) getCosWebsiteEndpoint() string {
	return fmt.Sprintf("cos-website.%s.myqcloud.com", self.GetId())
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range iBuckets {
		bucket := iBuckets[i].(*SBucket)
		if bucket.region.GetId() != region.GetId() {
			continue
		}
		ret = append(ret, iBuckets[i])
	}
	return ret, nil
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	bucket := &SBucket{
		region: region,
		Name:   name,
	}
	coscli, err := region.GetCosClient(bucket)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.BucketPutOptions{}
	if len(aclStr) > 0 {
		if utils.IsInStringArray(aclStr, []string{
			"private", "public-read", "public-read-write", "authenticated-read",
		}) {
			opts.XCosACL = aclStr
		} else {
			return errors.Error("invalid acl")
		}
	}
	_, err = coscli.Bucket.Put(context.Background(), opts)
	if err != nil {
		return errors.Wrap(err, "coscli.Bucket.Put")
	}
	region.client.invalidateIBuckets()
	return nil
}

func cosHttpCode(err error) int {
	if httpErr, ok := err.(*cos.ErrorResponse); ok {
		return httpErr.Response.StatusCode
	}
	return -1
}

func (region *SRegion) DeleteIBucket(name string) error {
	bucket := &SBucket{
		region: region,
		Name:   name,
	}
	coscli, err := region.GetCosClient(bucket)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	_, err = coscli.Bucket.Delete(context.Background())
	if err != nil {
		if cosHttpCode(err) == 404 {
			return nil
		}
		return errors.Wrap(err, "DeleteBucket")
	}
	return nil
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	bucket := &SBucket{
		region: region,
		Name:   name,
	}
	coscli, err := region.GetCosClient(bucket)
	if err != nil {
		return false, errors.Wrap(err, "GetCosClient")
	}
	_, err = coscli.Bucket.Head(context.Background())
	if err != nil {
		if cosHttpCode(err) == 404 {
			return false, nil
		}
		return false, errors.Wrap(err, "BucketExists")
	}
	return true, nil
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(region, name)
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}

func (self *SRegion) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	secgroup, err := self.GetSecurityGroup(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroups(%s)", id)
	}
	return secgroup, nil
}

func (self *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	ret := []cloudprovider.ICloudSecurityGroup{}
	for {
		part, total, err := self.GetSecurityGroups(nil, "", len(ret), 100)
		if err != nil {
			return nil, err
		}
		for i := range part {
			part[i].region = self
			ret = append(ret, &part[i])
		}
		if len(part) == 0 || len(ret) >= total {
			break
		}
	}
	return ret, nil
}

func (self *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	group, err := self.CreateSecurityGroup(opts)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	caches, err := region.GetCloudElasticcaches("")
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcaches")
	}

	ret := []cloudprovider.ICloudElasticcache{}
	for i := range caches {
		cache := caches[i]
		cache.region = region
		ret = append(ret, &cache)
	}

	mems := []SMemcached{}
	offset := 0
	for {
		part, total, err := region.GetMemcaches(nil, 100, offset)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported {
				return ret, nil
			}
			return nil, errors.Wrapf(err, "GetMemcaches")
		}
		mems = append(mems, part...)
		if len(mems) >= total {
			break
		}
		offset += len(part)
	}
	for i := range mems {
		mems[i].region = region
		ret = append(ret, &mems[i])
	}

	return ret, nil
}

func (region *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	if strings.HasPrefix(id, "cmem-") {
		memcacheds, _, err := region.GetMemcaches([]string{id}, 1, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "GetMemcaches")
		}
		for i := range memcacheds {
			if memcacheds[i].GetGlobalId() == id {
				memcacheds[i].region = region
				return &memcacheds[i], nil
			}
		}
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	caches, err := region.GetCloudElasticcaches(id)
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcaches")
	}

	for i := range caches {
		if caches[i].GetGlobalId() == id {
			caches[i].region = region
			return &caches[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

// DescribeProductInfo 可以查询在售可用区信息
// https://cloud.tencent.com/document/product/239/20026
func (r *SRegion) CreateIElasticcaches(ec *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	params := map[string]string{}
	if len(ec.ZoneIds) == 0 {
		return nil, fmt.Errorf("CreateIElasticcaches zone id should not be empty.")
	}

	zoneId, ok := zoneIdMaps[ec.ZoneIds[0]]
	if !ok {
		return nil, fmt.Errorf("can't convert zone %s to integer id", ec.ZoneIds[0])
	}

	if len(ec.ZoneIds) > 1 {
		for i := range ec.ZoneIds {
			if i == 0 {
				params[fmt.Sprintf("NodeSet.%d.NodeType", i)] = "0"
				params[fmt.Sprintf("NodeSet.%d.ZoneId", i)] = fmt.Sprintf("%d", zoneId)
			} else {
				_z, ok := zoneIdMaps[ec.ZoneIds[i]]
				if !ok {
					return nil, fmt.Errorf("can't convert zone %s to integer id", ec.ZoneIds[i])
				}
				params[fmt.Sprintf("NodeSet.%d.NodeType", i)] = "1"
				params[fmt.Sprintf("NodeSet.%d.ZoneId", i)] = fmt.Sprintf("%d", _z)
			}
		}
	}

	spec, err := parseLocalInstanceSpec(ec.InstanceType)
	if err != nil {
		return nil, errors.Wrap(err, "parseLocalInstanceSpec")
	}

	params["InstanceName"] = ec.InstanceName
	if len(ec.ProjectId) > 0 {
		params["ProjectId"] = ec.ProjectId
	}
	params["ZoneId"] = fmt.Sprintf("%d", zoneId)
	params["TypeId"] = spec.TypeId
	params["MemSize"] = strconv.Itoa(spec.MemSizeMB)
	params["RedisShardNum"] = spec.RedisShardNum
	params["RedisReplicasNum"] = spec.RedisReplicasNum
	params["GoodsNum"] = "1"
	if ec.NetworkType == api.LB_NETWORK_TYPE_VPC {
		params["VpcId"] = ec.VpcId
		params["SubnetId"] = ec.NetworkId

		for i := range ec.SecurityGroupIds {
			params[fmt.Sprintf("SecurityGroupIdList.%d", i)] = ec.SecurityGroupIds[i]
		}
	}
	params["Period"] = "1"
	params["BillingMode"] = "0"
	if ec.BillingCycle != nil && ec.BillingCycle.GetMonths() >= 1 {
		params["Period"] = strconv.Itoa(ec.BillingCycle.GetMonths())
		params["BillingMode"] = "1"
		// 自动续费
		if ec.BillingCycle.AutoRenew {
			params["AutoRenew"] = "1"
		}
	}

	if len(ec.Password) > 0 {
		params["NoAuth"] = "false"
		params["Password"] = ec.Password
	} else {
		params["NoAuth"] = "true"
	}

	resp, err := r.redisRequest("CreateInstances", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateInstances")
	}

	instanceId := ""
	if resp.Contains("InstanceIds") {
		ids := []string{}
		if err := resp.Unmarshal(&ids, "InstanceIds"); err != nil {
			log.Debugf("Unmarshal.InstanceIds %s", resp)
		} else {
			if len(ids) > 0 {
				instanceId = ids[0]
			}
		}
	}

	// try to fetch instance id from deal id
	if len(instanceId) == 0 {
		dealId, err := resp.GetString("DealId")
		if err != nil {
			return nil, errors.Wrap(err, "dealId")
		}

		// maybe is a dealId not a instance ID.
		if strings.HasPrefix(dealId, "crs-") {
			instanceId = dealId
		} else {
			err = cloudprovider.Wait(5*time.Second, 900*time.Second, func() (bool, error) {
				_realInstanceId, err := r.GetElasticcacheIdByDeal(dealId)
				if err != nil {
					return false, nil
				}

				instanceId = _realInstanceId
				return true, nil
			})
			if err != nil {
				return nil, errors.Wrap(err, "Wait.GetElasticcacheIdByDeal")
			}
		}
	}

	err = r.SetResourceTags("redis", "instance", []string{instanceId}, ec.Tags, false)
	if err != nil {
		log.Errorf("SetResourceTags(redis:%s,error:%s)", instanceId, err)
	}
	return r.GetIElasticcacheById(instanceId)
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms := make([]SInstance, 0)
	for {
		parts, total, err := region.GetInstances("", nil, len(vms), 50)
		if err != nil {
			return nil, err
		}
		vms = append(vms, parts...)
		if len(vms) >= total {
			break
		}
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i++ {
		ivms[i] = &vms[i]
	}
	return ivms, nil
}
