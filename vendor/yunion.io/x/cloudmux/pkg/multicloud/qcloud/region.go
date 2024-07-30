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
	vpc, err := self.GetVpc(vpcId)
	if err != nil {
		return nil, err
	}
	return vpc, nil
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
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		if zones[i].Zone == id {
			return &zones[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetVpcs(nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
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
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func getReplaceKey(zoneId string) string {
	replceKey := map[string]string{"1": "一", "2": "二", "3": "三", "4": "四", "5": "五", "6": "六", "7": "七", "8": "八", "9": "九"}
	info := strings.Split(zoneId, "-")
	if len(info) >= 3 {
		return replceKey[info[len(info)-1]]
	}
	return ""
}

func (self *SRegion) GetZones() ([]SZone, error) {
	body, err := self.cvmRequest("DescribeZones", map[string]string{}, true)
	if err != nil {
		return nil, err
	}
	zones := make([]SZone, 0)
	err = body.Unmarshal(&zones, "ZoneSet")
	if err != nil {
		return nil, err
	}
	zoneName, zoneNameReplace := "", ""
	zoneMap := map[string]bool{}
	for i := 0; i < len(zones); i++ {
		if len(zoneName) == 0 {
			zoneName = zones[i].ZoneName
			zoneNameReplace = getReplaceKey(zones[i].Zone)
		}
		zones[i].region = self
		zoneMap[zones[i].Zone] = true
	}
	networks, err := self.GetNetworks(nil, "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetworks")
	}
	for _, network := range networks {
		if _, ok := zoneMap[network.Zone]; !ok {
			zoneMap[network.Zone] = true
			zone := SZone{region: self, Zone: network.Zone, ZoneState: "Unknown"}
			newKey := getReplaceKey(network.Zone)
			if len(zoneNameReplace) > 0 && len(newKey) > 0 {
				zone.ZoneName = strings.Replace(zoneName, zoneNameReplace, newKey, 1)
			}
			zones = append(zones, zone)
		}
	}
	return zones, nil
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := make(map[string]string)
	params["VpcId"] = vpcId
	_, err := self.vpcRequest("DeleteVpc", params)
	return err
}

func (self *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	vpcs, err := self.GetVpcs([]string{vpcId})
	if err != nil {
		return nil, err
	}
	for i := range vpcs {
		if vpcs[i].VpcId == vpcId {
			vpcs[i].region = self
			return &vpcs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetVpc(%s)", vpcId)
}

func (self *SRegion) GetVpcs(vpcIds []string) ([]SVpc, error) {
	params := map[string]string{
		"Limit": "100",
	}
	for index, vpcId := range vpcIds {
		params[fmt.Sprintf("VpcIds.%d", index)] = vpcId
	}
	ret := []SVpc{}
	for {
		resp, err := self.vpcRequest("DescribeVpcs", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			VpcSet     []SVpc
			TotalCount float64
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		for i := range part.VpcSet {
			part.VpcSet[i].region = self
			ret = append(ret, part.VpcSet[i])
		}
		if len(ret) >= int(part.TotalCount) || len(part.VpcSet) == 0 {
			break
		}
		params["Offset"] = fmt.Sprintf("%d", len(ret))
	}
	return ret, nil
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

func (self *SRegion) wafRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.wafRequest(apiName, params)
}

func (self *SRegion) memcachedRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.memcachedRequest(apiName, params)
}

func (self *SRegion) GetNetworks(ids []string, vpcId, zone string) ([]SNetwork, error) {
	params := map[string]string{
		"Limit": "100",
	}
	for index, networkId := range ids {
		params[fmt.Sprintf("SubnetIds.%d", index)] = networkId
	}
	base := 0
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", base)] = "vpc-id"
		params[fmt.Sprintf("Filters.%d.Values.0", base)] = vpcId
		base++
	}
	if len(zone) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", base)] = "zone"
		params[fmt.Sprintf("Filters.%d.Values.0", base)] = zone
		base++
	}
	ret := []SNetwork{}
	for {
		resp, err := self.vpcRequest("DescribeSubnets", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			SubnetSet  []SNetwork
			TotalCount float64
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.SubnetSet...)
		if len(ret) >= int(part.TotalCount) || len(part.SubnetSet) == 0 {
			break
		}
		params["Offset"] = fmt.Sprintf("%d", len(ret))
	}
	return ret, nil
}

func (self *SRegion) GetNetwork(id string) (*SNetwork, error) {
	networks, err := self.GetNetworks([]string{id}, "", "")
	if err != nil {
		return nil, err
	}
	for i := range networks {
		if networks[i].SubnetId == id {
			return &networks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, networkId string, passwd string, publicKey string, secgroup string, tags map[string]string) (*SInstance, error) {
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	net, err := self.GetNetwork(networkId)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		zone := zones[i]
		log.Debugf("Search in zone %s", zone.Zone)
		if zone.ZoneName != net.Zone {
			continue
		}
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
		inst, err := zone.getHost().CreateVM(desc)
		if err != nil {
			return nil, err
		}
		return inst.(*SInstance), nil
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
