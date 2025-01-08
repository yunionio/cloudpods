// Copyright 2023 Yunion
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

package volcengine

import (
	"context"
	"fmt"
	"strings"
	"time"

	tos "github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
	sdk "github.com/volcengine/volc-sdk-golang/base"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

var RegionLocations = map[string]string{
	"cn-beijing":     "华东2（北京）",
	"cn-shanghai":    "华东2（上海）",
	"cn-guangzhou":   "华东2（广州）",
	"ap-southeast-1": "亚太东南（柔佛）",
	"cn-hongkong":    "中国香港",
}

var RegionEndpoint = map[string]string{
	"cn-beijing":     "cn-beijing.volces.com",
	"cn-shanghai":    "cn-shanghai.volces.com",
	"cn-guangzhou":   "cn-beijing.volces.com",
	"ap-southeast-1": "ap-southeast-1.volces.com",
	"cn-hongkong":    "cn-hongkong.volces.com",
}

type sStorageType struct {
	Id    string
	Zones []string
}

type SRegion struct {
	multicloud.SRegion
	multicloud.SNoLbRegion

	client    *SVolcEngineClient
	tosClient *tos.ClientV2
	RegionId  string

	ivpcs []cloudprovider.ICloudVpc

	storageTypes []sStorageType
	storageCache *SStoragecache
}

func (region *SRegion) GetClient() *SVolcEngineClient {
	return region.client
}

func (region *SRegion) Refresh() error {
	return nil
}

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_VOLCENGINE
}

func (region *SRegion) GetCloudEnv() string {
	return region.client.cloudEnv
}

func (region *SRegion) GetId() string {
	return region.RegionId
}

func (region *SRegion) GetName() string {
	if localName, ok := RegionLocations[region.RegionId]; ok {
		return fmt.Sprintf("%s %s", CLOUD_PROVIDER_VOLCENGINE_CN, localName)
	}
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_VOLCENGINE_CN, region.RegionId)
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", region.client.GetAccessEnv(), region.RegionId)
}

func (region *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	return cloudprovider.SModelI18nTable{}
}

func (region *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[region.RegionId]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (region *SRegion) getStoragecache() *SStoragecache {
	if region.storageCache == nil {
		region.storageCache = &SStoragecache{region: region}
	}
	return region.storageCache
}

func (region *SRegion) GetZones(id string) ([]SZone, error) {
	params := map[string]string{}
	params["DestinationResource"] = "InstanceType"
	// DedicatedHost is not supported
	if len(id) > 0 {
		params["ZoneId"] = id
	}
	body, err := region.ecsRequest("DescribeAvailableResource", params)
	if err != nil {
		return nil, err
	}
	ret := []SZone{}
	err = body.Unmarshal(&ret, "AvailableZones")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := region.GetZones("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetZones")
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		zones[i].region = region
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (region *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := region.GetZones("")
	if err != nil {
		return nil, errors.Wrap(err, "GetZones")
	}

	for i := range zones {
		zones[i].region = region
		if zones[i].GetId() == id || zones[i].GetGlobalId() == id {
			return &zones[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
}

// vpc
func (region *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	vpc, err := region.CreateVpc(opts)
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

func (region *SRegion) CreateVpc(opts *cloudprovider.VpcCreateOptions) (*SVpc, error) {
	params := make(map[string]string)
	if len(opts.CIDR) > 0 {
		params["CidrBlock"] = opts.CIDR
	}
	if len(opts.NAME) > 0 {
		params["VpcName"] = opts.NAME
	}
	if len(opts.Desc) > 0 {
		params["Description"] = opts.Desc
	}
	params["ClientToken"] = utils.GenRequestId(20)
	body, err := region.vpcRequest("CreateVpc", params)
	if err != nil {
		return nil, err
	}
	vpcId, err := body.GetString("VpcId")
	if err != nil {
		return nil, err
	}
	err = cloudprovider.Wait(5*time.Second, time.Minute, func() (bool, error) {
		_, err = region.getVpc(vpcId)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		} else {
			return true, err
		}
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find networks after create")
	}
	return region.getVpc(vpcId)
}

func (region *SRegion) DeleteVpc(vpcId string) error {
	params := make(map[string]string)
	params["VpcId"] = vpcId

	_, err := region.vpcRequest("DeleteVpc", params)
	return err
}

func (region *SRegion) getVpc(vpcId string) (*SVpc, error) {
	vpcs, _, err := region.GetVpcs([]string{vpcId}, 1, 50)
	if err != nil {
		return nil, err
	}
	for _, vpc := range vpcs {
		if vpc.VpcId == vpcId {
			vpc.region = region
			return &vpc, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s not found", vpcId)
}

func (region *SRegion) GetVpcs(vpcIds []string, pageNumber int, pageSize int) ([]SVpc, int, error) {
	params := make(map[string]string)
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
	if len(vpcIds) > 0 {
		for index, id := range vpcIds {
			key := fmt.Sprintf("VpcIds.%d", index+1)
			params[key] = id
		}
	}
	body, err := region.vpcRequest("DescribeVpcs", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "GetVpcs fail")
	}
	vpcs := make([]SVpc, 0)
	err = body.Unmarshal(&vpcs, "Vpcs")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal vpcs fail")
	}
	total, _ := body.Int("TotalCount")
	return vpcs, int(total), nil
}

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if region.ivpcs == nil {
		vpcs, err := region.GetAllVpcs()
		if err != nil {
			return nil, err
		}
		region.ivpcs = make([]cloudprovider.ICloudVpc, len(vpcs))
		for i := 0; i < len(vpcs); i += 1 {
			vpcs[i].region = region
			region.ivpcs[i] = &vpcs[i]
		}
	}
	return region.ivpcs, nil
}

func (region *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := region.GetIVpcs()
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

func (region *SRegion) GetAllVpcs() ([]SVpc, error) {
	vpcs := make([]SVpc, 0)
	pageNumber := 1
	for {
		part, total, err := region.GetVpcs(nil, pageNumber, 50)
		if err != nil {
			return nil, err
		}
		vpcs = append(vpcs, part...)
		if len(vpcs) >= total {
			break
		}
		pageNumber += 1
	}
	return vpcs, nil
}

// EIP
func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	eip, err := region.GetEip(eipId)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	pageNumber := 1
	eips, total, err := region.GetEips(make([]string, 0), "", make([]string, 0), pageNumber, 100)
	if err != nil {
		return nil, err
	}
	for len(eips) < total {
		var parts []SEipAddress
		pageNumber++
		parts, total, err = region.GetEips(make([]string, 0), "", make([]string, 0), pageNumber, 100)
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

// IBucket
func (region *SRegion) IBucketExist(name string) (bool, error) {
	toscli, err := region.GetTosClient()
	if err != nil {
		return false, errors.Wrap(err, "region.GetTosClient")
	}
	_, err = toscli.HeadBucket(context.Background(), &tos.HeadBucketInput{Bucket: name})
	if err != nil || tos.StatusCode(err) != 404 {
		return false, errors.Wrap(err, "IsBucketExist")
	}
	return true, nil
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	toscli, err := region.GetTosClient()
	if err != nil {
		return errors.Wrap(err, "region.GetTosClient")
	}
	_, err = toscli.CreateBucketV2(context.Background(), &tos.CreateBucketV2Input{Bucket: name, ACL: enum.ACLType(aclStr), StorageClass: enum.StorageClassType(storageClassStr)})
	if err != nil {
		return errors.Wrap(err, "tos.CreateBucketV2")
	}
	region.client.invalidateIBuckets()
	return nil
}

func (region *SRegion) DeleteIBucket(name string) error {
	toscli, err := region.GetTosClient()
	if err != nil {
		return errors.Wrapf(err, "region.GetOssClient")
	}
	_, err = toscli.DeleteBucket(context.Background(), &tos.DeleteBucketInput{Bucket: name})
	if err != nil {
		if tos.StatusCode(err) == 404 {
			return nil
		}
		return errors.Wrap(err, "DeleteBucket")
	}
	region.client.invalidateIBuckets()
	return nil
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	toscli, err := region.GetTosClient()
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetOssClient")
	}
	out, err := toscli.ListBuckets(context.Background(), &tos.ListBucketsInput{})
	if err != nil {
		return nil, errors.Wrap(err, "ListBucket")
	}
	for _, bucket := range out.Buckets {
		if bucket.Name == name {
			t, err := time.Parse(time.RFC3339, bucket.CreationDate)
			if err != nil {
				return nil, errors.Wrapf(err, "Prase CreationDate error")
			}
			b := SBucket{
				region:       region,
				Name:         name,
				Location:     bucket.Location,
				CreationDate: t,
			}
			return &b, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "Bucket Not Found")
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "getIBuckets")
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range iBuckets {
		if iBuckets[i].GetIRegion().GetId() != region.GetId() {
			continue
		}
		ret = append(ret, iBuckets[i])
	}
	return ret, nil
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

// Security Group
func (region *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	externalId, err := region.CreateSecurityGroup(opts)
	if err != nil {
		return nil, err
	}
	err = cloudprovider.Wait(5*time.Second, time.Minute, func() (bool, error) {
		_, err := region.GetISecurityGroupById(externalId)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		} else {
			return true, err
		}
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find security group after create")
	}
	return region.GetISecurityGroupById(externalId)
}

func (region *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return region.GetSecurityGroup(secgroupId)
}

func (region *SRegion) getSdkCredential(service string, token string) sdk.Credentials {
	return region.client.getSdkCredential(region.RegionId, service, token)
}

func (region *SRegion) ecsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cred := region.getSdkCredential(VOLCENGINE_SERVICE_ECS, "")
	return region.client.jsonRequest(cred, VOLCENGINE_API, VOLCENGINE_API_VERSION, apiName, params)
}

func (region *SRegion) vpcRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cred := region.getSdkCredential(VOLCENGINE_SERVICE_VPC, "")
	return region.client.jsonRequest(cred, VOLCENGINE_API, VOLCENGINE_API_VERSION, apiName, params)
}

func (region *SRegion) natRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cred := region.getSdkCredential(VOLCENGINE_SERVICE_NAT, "")
	return region.client.jsonRequest(cred, VOLCENGINE_API, VOLCENGINE_API_VERSION, apiName, params)
}

func (region *SRegion) storageRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cred := region.getSdkCredential(VOLCENGINE_SERVICE_STORAGE, "")
	return region.client.jsonRequest(cred, VOLCENGINE_API, VOLCENGINE_API_VERSION, apiName, params)
}

func (region *SRegion) GetTosClient() (*tos.ClientV2, error) {
	if region.tosClient == nil {
		cli, err := region.client.getTosClient(region.RegionId)
		if err != nil {
			return nil, errors.Wrap(err, "region.client.getOssClient")
		}
		region.tosClient = cli
	}
	return region.tosClient, nil
}

func (region *SRegion) UpdateInstancePassword(instanceId string, passwd string) error {
	params := make(map[string]string)
	params["Password"] = passwd
	return region.modifyInstanceAttribute(instanceId, params)
}

func (region *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.Status, nil
}

func (region *SRegion) instanceOperation(instanceId string, apiName string, extra map[string]string) error {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["InstanceId"] = instanceId
	if len(extra) > 0 {
		for k, v := range extra {
			params[k] = v
		}
	}
	_, err := region.ecsRequest(apiName, params)
	return err
}

func (region *SRegion) getBaseEndpoint() string {
	return RegionEndpoint[region.RegionId]
}

func (region *SRegion) getS3Endpoint() string {
	base := region.getBaseEndpoint()
	if len(base) > 0 {
		return "tos-s3-" + base
	}
	return ""
}

func (region *SRegion) getTOSExternalDomain() string {
	return getTOSExternalDomain(region.RegionId)
}

func (region *SRegion) getTOSInternalDomain() string {
	return getTOSInternalDomain(region.RegionId)
}

func (region *SRegion) GetRouteTables(ids []string, pageNumber int, pageSize int) ([]SRouteTable, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
	if len(ids) > 0 {
		params["RouteTableId"] = strings.Join(ids, ",")
	}
	body, err := region.vpcRequest("DescribeRouteTableList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "GetRoutseTables fail")
	}
	routetables := make([]SRouteTable, 0)
	err = body.Unmarshal(&routetables, "RouteTables", "RouteTable")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal routetables fail")
	}
	total, _ := body.Int("TotalCount")
	return routetables, int(total), nil
}

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := region.GetIZones()
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

func (regioin *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := make([]cloudprovider.ICloudHost, 0)

	izones, err := regioin.GetIZones()
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

func (region *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return region.GetDisk(id)
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := region.GetIZones()
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

func (region *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := region.GetIZones()
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

func (region *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return region.GetInstance(id)
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
