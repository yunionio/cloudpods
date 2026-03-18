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

package ecloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// SOpenApiRegion 用于接收 OpenAPI 区域列表返回的单个区域结构
type SRegion struct {
	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion

	client *SEcloudClient

	RegionId   string `json:"regionId"`
	RegionCode string `json:"regionCode"`
	RegionName string `json:"regionName"`
	RegionArea string `json:"regionArea"`
}

func (r *SRegion) GetId() string {
	return r.RegionId
}

func (r *SRegion) GetName() string {
	return r.RegionName
}

func (r *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", r.client.GetAccessEnv(), r.RegionId)
}

func (r *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (r *SRegion) Refresh() error {
	return nil
}

func (r *SRegion) IsEmulated() bool {
	return false
}

func (r *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	return table
}

// GetLatitude() float32
// GetLongitude() float32
func (r *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[r.RegionId]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (r *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := r.GetZones()
	if err != nil {
		return nil, err
	}
	izones := make([]cloudprovider.ICloudZone, len(zones))
	for i := range zones {
		izones[i] = &zones[i]
	}
	return izones, nil
}

func (r *SRegion) GetZones() ([]SZone, error) {
	zones := make([]SZone, 0)
	req := NewOpenApiZoneRequest(r.RegionId, nil)
	if err := r.client.doList(context.Background(), req.Base(), &zones); err != nil {
		return nil, errors.Wrap(err, "GetZones")
	}
	for i := range zones {
		zones[i].region = r
	}
	return zones, nil
}

func (r *SRegion) GetVpcs() ([]SVpc, error) {
	req := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/vpc", nil, nil)
	vpcs := make([]SVpc, 0)
	if err := r.client.doList(context.Background(), req.Base(), &vpcs); err != nil {
		return nil, err
	}
	for i := range vpcs {
		vpcs[i].region = r
	}
	return vpcs, nil
}

func (r *SRegion) GetVpc(id string) (*SVpc, error) {
	base := NewOpenApiVpcRequest(r.RegionId, fmt.Sprintf("/api/openapi-vpc/customer/v3/vpc/%s", id), nil, nil).Base()
	base.SetMethod("GET")
	resp, err := r.client.request(context.Background(), base)
	if err != nil {
		// 可能传入的是 routerId，再试详情接口 by routerId
		return r.getVpcByRouterId(id)
	}
	vpc := SVpc{}
	if err := resp.Unmarshal(&vpc); err != nil {
		return nil, errors.Wrap(err, "Unmarshal vpc")
	}
	vpc.region = r
	return &vpc, nil
}

func (r *SRegion) getVpcByRouterId(routerId string) (*SVpc, error) {
	base := NewOpenApiVpcRequest(r.RegionId, fmt.Sprintf("/api/openapi-vpc/customer/v3/vpc/router/%s", routerId), nil, nil).Base()
	base.SetMethod("GET")
	resp, err := r.client.request(context.Background(), base)
	if err != nil {
		return nil, err
	}
	vpc := SVpc{}
	if err := resp.Unmarshal(&vpc); err != nil {
		return nil, errors.Wrap(err, "Unmarshal vpc")
	}
	vpc.region = r
	return &vpc, nil
}

// CreateVpc 创建 VPC，OpenAPI POST /api/openapi-vpc/customer/v3/order/create/vpc。
// name/networkName 须 5-22 位、字母开头；body 使用 vpcOrderCreateBody 包装以兼容网关。
func (r *SRegion) CreateVpc(opts *cloudprovider.VpcCreateOptions) (*SVpc, error) {
	name := opts.NAME
	if name == "" {
		name = "vpc-default"
	}
	// 规范：必须以字母开头，长度 5-22（数字、字母、下划线）
	if name[0] < 'a' || name[0] > 'z' {
		if name[0] < 'A' || name[0] > 'Z' {
			name = "v" + name
		}
	}
	if len(name) < 5 {
		pad := 5 - len(name)
		if pad > 4 {
			pad = 4
		}
		name = name + "xxxx"[:pad]
	}
	if len(name) > 22 {
		name = name[:22]
	}
	cidr := opts.CIDR
	if cidr == "" {
		cidr = "192.168.0.0/16"
	}
	networkName := name + "-subnet1"
	if len(networkName) > 22 {
		networkName = name + "-s1"
		if len(networkName) > 22 {
			networkName = name[:20] + "-s1"
		}
	}
	poolID := regionIdToPoolId[r.RegionId]
	if poolID == "" {
		poolID = r.RegionId
	}
	inner := jsonutils.NewDict()
	inner.Set("name", jsonutils.NewString(name))
	inner.Set("cidr", jsonutils.NewString(cidr))
	inner.Set("networkName", jsonutils.NewString(networkName))
	inner.Set("region", jsonutils.NewString(poolID))
	inner.Set("specs", jsonutils.NewString("high"))
	inner.Set("networkTypeEnum", jsonutils.NewString("VM"))
	if opts.Desc != "" {
		inner.Set("description", jsonutils.NewString(opts.Desc))
	}
	body := jsonutils.NewDict()
	body.Set("vpcOrderCreateBody", inner)
	base := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/order/create/vpc", nil, body).Base()
	base.SetMethod("POST")
	_, err := r.client.request(context.Background(), base)
	if err != nil {
		if strings.Contains(err.Error(), "该可用区暂无可用的网络资源") || strings.Contains(err.Error(), "THIS_AZ_MUST_AUTH") {
			return nil, errors.Wrap(err, "当前区域/可用区暂无可用网络资源或需开通权限，请尝试其他区域或联系移动云")
		}
		return nil, errors.Wrap(err, "CreateVpc")
	}
	// 创建为订购接口，返回 orderId；轮询列表按名称查找新 VPC
	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			time.Sleep(time.Duration(3+retry*2) * time.Second)
		}
		vpcs, err := r.GetVpcs()
		if err != nil {
			continue
		}
		for i := range vpcs {
			if vpcs[i].Name == name {
				vpcs[i].region = r
				return &vpcs[i], nil
			}
		}
	}
	return nil, errors.Wrap(errors.ErrNotFound, "created vpc not found in list yet")
}

// DeleteVpc 退订 VPC，OpenAPI POST /api/openapi-vpc/customer/v3/order/delete。
// 参数 idOrRouterId 可为 vpc Id 或 routerId；若为 vpc Id 会先查详情取 routerId 再退订。
func (r *SRegion) DeleteVpc(idOrRouterId string) error {
	routerId := idOrRouterId
	if vpc, err := r.GetVpc(idOrRouterId); err == nil {
		routerId = vpc.RouterId
	}
	commonBody := jsonutils.NewDict()
	commonBody.Set("resourceId", jsonutils.NewString(routerId))
	commonBody.Set("productType", jsonutils.NewString("router"))
	reqBody := jsonutils.NewDict()
	reqBody.Set("commonMopOrderDeleteVpcBody", commonBody)
	base := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/order/delete", nil, reqBody).Base()
	base.SetMethod("POST")
	_, err := r.client.request(context.Background(), base)
	return errors.Wrap(err, "DeleteVpc")
}

// DeleteNetwork 删除 VPC 下网络（子网），与 ecloudsdkvpc DeleteNetwork 一致。
func (r *SRegion) DeleteNetwork(networkId string) error {
	base := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/network/"+networkId, nil, nil).Base()
	base.SetMethod("DELETE")
	_, err := r.client.request(context.Background(), base)
	return errors.Wrap(err, "DeleteNetwork")
}

// CreateNetwork 在 VPC 下创建网络（子网），与 ecloudsdkvpc CreateNetwork 一致。
// networkName 须 5-22 位、字母开头；cidr 如 192.168.1.0/24。
func (r *SRegion) CreateNetwork(routerId, regionPoolId, networkName, cidr string) (*SNetwork, error) {
	body := jsonutils.NewDict()
	inner := jsonutils.NewDict()
	inner.Set("routerId", jsonutils.NewString(routerId))
	inner.Set("networkName", jsonutils.NewString(networkName))
	inner.Set("networkTypeEnum", jsonutils.NewString("VM"))
	if regionPoolId != "" {
		inner.Set("availabilityZoneHints", jsonutils.NewString(regionPoolId))
	}
	subnet := jsonutils.NewDict()
	subnet.Set("cidr", jsonutils.NewString(cidr))
	subnet.Set("ipVersion", jsonutils.NewString("4"))
	inner.Set("subnets", jsonutils.NewArray(subnet))
	body.Set("createNetworkBody", inner)
	base := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/network", nil, body).Base()
	base.SetMethod("POST")
	data, err := r.client.request(context.Background(), base)
	if err != nil {
		return nil, errors.Wrap(err, "CreateNetwork")
	}
	// 响应可能含 body（网络 id）或直接返回网络信息
	var networkId string
	if data.Contains("body") {
		networkId, _ = data.GetString("body")
	}
	if networkId == "" && data.Contains("id") {
		networkId, _ = data.GetString("id")
	}
	if networkId == "" {
		// 通过名称再查一次
		networks, listErr := r.GetNetworks(routerId, "")
		if listErr != nil {
			return nil, errors.Wrapf(listErr, "CreateNetwork succeeded but list networks failed")
		}
		for i := range networks {
			if networks[i].Name == networkName {
				return &networks[i], nil
			}
		}
		return nil, errors.Errorf("CreateNetwork succeeded but could not find created network by name %q", networkName)
	}
	return r.GetNetwork(networkId)
}

func (r *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := r.GetVpcs()
	if err != nil {
		return nil, err
	}
	ivpcs := make([]cloudprovider.ICloudVpc, len(vpcs))
	for i := range vpcs {
		ivpcs[i] = &vpcs[i]
	}
	return ivpcs, nil
}

func (r *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	// 使用 OpenAPI EIP 公网 IP 列表（含带宽信息）：
	// GET /api/openapi-eip/acl/v3/floatingip/listWithBw
	base := NewOpenApiEbsRequest(r.RegionId, "/api/openapi-eip/acl/v3/floatingip/listWithBw", nil, nil).Base()
	eips := make([]SEip, 0, 20)
	if err := r.client.doList(context.Background(), base, &eips); err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudEIP, len(eips))
	for i := range eips {
		eips[i].region = r
		ret[i] = &eips[i]
	}
	return ret, nil
}

func (r *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := r.GetVpc(id)
	if err != nil {
		return nil, err
	}
	vpc.region = r
	return vpc, nil
}

func (r *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	vpc, err := r.CreateVpc(opts)
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

func (r *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := r.GetZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		if zones[i].GetGlobalId() == id || zones[i].GetId() == id {
			return &zones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (r *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := r.GetInstance(id)
	if err != nil {
		return nil, err
	}
	zones, err := r.GetZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].ZoneId == vm.ZoneId {
			vm.host = &SHost{
				zone: &zones[i],
			}
			return vm, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return r.GetDisk(id)
}

// ResizeDisk 扩容云盘，使用 OpenAPI /api/v2/volume/volume/change/ebs，与 ecloudsdkebs ChangeVolume 一致。
// newSizeGB 为目标容量（GB），只做直通调用，参数合法性由上层调用方保证。
func (r *SRegion) ResizeDisk(ctx context.Context, diskId string, newSizeGB int64) error {
	body := jsonutils.NewDict()
	body.Set("size", jsonutils.NewInt(newSizeGB))
	body.Set("changeType", jsonutils.NewString("CHANGE"))
	body.Set("volumeId", jsonutils.NewString(diskId))
	reqBody := jsonutils.NewDict()
	reqBody.Set("changeVolumeBody", body)
	base := NewOpenApiEbsRequest(r.RegionId, "/api/v2/volume/volume/change/ebs", nil, reqBody).Base()
	base.SetMethod("POST")
	_, err := r.client.request(ctx, base)
	return err
}

// PreDeleteVolume 退订/删除云盘（预删除），与 ecloudsdkebs PreDeleteResources 一致。
func (r *SRegion) PreDeleteVolume(volumeId string) error {
	body := jsonutils.NewDict()
	body.Set("resourceId", jsonutils.NewString(volumeId))
	body.Set("resourceType", jsonutils.NewString("VOLUME"))
	reqBody := jsonutils.NewDict()
	reqBody.Set("preDeleteResourcesBody", body)
	base := NewOpenApiEbsRequest(r.RegionId, "/api/ebs/acl/v3/common/resource/preDelete", nil, reqBody).Base()
	base.SetMethod("POST")
	_, err := r.client.request(context.Background(), base)
	return errors.Wrap(err, "PreDeleteVolume")
}

// CreateEbsSnapshot 创建云盘快照，与 ecloudsdkebs CreateSnapshot 一致。
func (r *SRegion) CreateEbsSnapshot(volumeId, name, description string) (string, error) {
	snapBody := jsonutils.NewDict()
	snapBody.Set("volumeId", jsonutils.NewString(volumeId))
	snapBody.Set("name", jsonutils.NewString(name))
	if description != "" {
		snapBody.Set("description", jsonutils.NewString(description))
	}
	reqBody := jsonutils.NewDict()
	reqBody.Set("createSnapshotBody", snapBody)
	base := NewOpenApiEbsRequest(r.RegionId, "/api/v2/volume/openApi/volumeSnapshot/create", nil, reqBody).Base()
	base.SetMethod("POST")
	data, err := r.client.request(context.Background(), base)
	if err != nil {
		return "", errors.Wrap(err, "CreateEbsSnapshot")
	}
	// 响应可能含 snapshotId 或 id
	if data.Contains("snapshotId") {
		id, _ := data.GetString("snapshotId")
		return id, nil
	}
	if data.Contains("id") {
		id, _ := data.GetString("id")
		return id, nil
	}
	return "", errors.Errorf("CreateEbsSnapshot response has no snapshotId/id: %s", data)
}

// DeleteEbsSnapshot 删除云盘快照，与 ecloudsdkebs Deletes 一致。
func (r *SRegion) DeleteEbsSnapshot(snapshotId string) error {
	base := NewOpenApiEbsRequest(r.RegionId, "/api/ebs/acl/v3/openApi/volumeSnapshot/"+snapshotId, nil, nil).Base()
	base.SetMethod("DELETE")
	_, err := r.client.request(context.Background(), base)
	return errors.Wrap(err, "DeleteEbsSnapshot")
}

func (r *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	izones, err := r.GetIZones()
	if err != nil {
		return nil, err
	}
	iHosts := make([]cloudprovider.ICloudHost, 0, len(izones))
	for i := range izones {
		hosts, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, hosts...)
	}
	return iHosts, nil
}

func (r *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	hosts, err := r.GetIHosts()
	if err != nil {
		return nil, err
	}
	for i := range hosts {
		if hosts[i].GetGlobalId() == id {
			return hosts[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := r.GetIZones()
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

func (r *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	istores, err := r.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := range istores {
		if istores[i].GetGlobalId() == id {
			return istores[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	sc := r.getStoragecache()
	return []cloudprovider.ICloudStoragecache{sc}, nil
}

func (r *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := r.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

type SPoolZoneInfo struct {
	ZoneId   string `json:"zoneId"`
	ZoneName string `json:"zoneName"`
	ZoneCode string `json:"zoneCode"`
}
type SPoolInfo struct {
	ZoneInfo    []SPoolZoneInfo `json:"zoneInfo"`
	PoolId      string          `json:"poolId"`
	PoolArea    string          `json:"poolArea"`
	ProductType string          `json:"productType"`
	PoolName    string          `json:"poolName"`
}
type SPoolInfosRespBody struct {
	PoolList []SPoolInfo `json:"poolList"`
}

func (r *SRegion) GetPoolInfo(productType string) ([]SPoolInfo, error) {
	query := map[string]string{
		"productType": productType,
	}
	base := NewOpenApiEbsRequest(r.RegionId, "/api/ebs/acl/v3/mop/common/getPoolInfo", query, nil).Base()
	base.SetMethod("GET")
	data, err := r.client.request(context.Background(), base)
	if err != nil {
		return nil, err
	}
	resp := SPoolInfosRespBody{}
	if err := data.Unmarshal(&resp); err != nil {
		return nil, err
	}
	return resp.PoolList, nil
}

// GetStorages 返回当前区域（可选指定可用区）的可用存储列表。
// 若 zoneCode 为空，则返回所有可用区；否则仅返回指定可用区的存储。
func (r *SRegion) GetStorages(zoneCode string) ([]SStorage, error) {
	volumeConfig, err := r.GetVolumeConfig()
	if err != nil {
		return nil, err
	}
	ret := make([]SStorage, 0)
	for _, config := range volumeConfig {
		if config.Region == zoneCode {
			ret = append(ret, config)
		}
	}
	ret = append(ret, SStorage{
		StorageType: api.STORAGE_ECLOUD_LOCAL,
		Region:      zoneCode,
	})
	return ret, nil
}

func (r *SRegion) GetVolumeConfig() ([]SStorage, error) {
	base := NewOpenApiEbsRequest(r.RegionId, "/api/ebs/customer/v3/volume/volumeType/list", nil, nil).Base()
	base.SetMethod("GET")
	data, err := r.client.request(context.Background(), base)
	if err != nil {
		return nil, errors.Wrap(err, "GetVolumeConfigs")
	}
	resp := []SStorage{}
	if err := data.Unmarshal(&resp); err != nil {
		return nil, errors.Wrap(err, "Unmarshal volume config")
	}
	return resp, nil
}

func (r *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_ECLOUD
}

func (r *SRegion) GetCapabilities() []string {
	return r.client.GetCapabilities()
}

func (r *SRegion) GetClient() *SEcloudClient {
	return r.client
}

// NewPlaceholderRegion 返回仅持有 client 的 SRegion，用于无有效 region 时仍可执行 region-list（如 OpenAPI 失败导致列表为空）。
func NewPlaceholderRegion(client *SEcloudClient, regionId string) *SRegion {
	return &SRegion{RegionId: regionId, client: client}
}

func (r *SRegion) FindZone(zoneCode string) (*SZone, error) {
	zones, err := r.GetZones()
	if err != nil {
		return nil, errors.Wrap(err, "unable to GetZones")
	}
	for i := range zones {
		if zones[i].ZoneCode == zoneCode {
			return &zones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := region.GetInstances("", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetVMs")
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := range vms {
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (r *SRegion) GetSecurityGroups() ([]SSecurityGroup, error) {
	// --- 安全组 OpenAPI（ecloudsdkvpc 路径，使用 console 主机）---
	query := map[string]string{
		"region": r.RegionId,
	}
	req := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/SecurityGroup", query, nil)
	sgs := make([]SSecurityGroup, 0, 8)
	if err := r.client.doList(context.Background(), req.Base(), &sgs); err != nil {
		return nil, err
	}
	for i := range sgs {
		sgs[i].region = r
	}
	return sgs, nil
}

// GetSecurityGroup 按 ID 获取安全组，返回底层 SSecurityGroup 结构。
func (r *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	base := NewOpenApiVpcRequest(r.RegionId, fmt.Sprintf("/api/openapi-vpc/customer/v3/SecurityGroup/%s", id), nil, nil).Base()
	base.SetMethod("GET")
	resp, err := r.client.request(context.Background(), base)
	if err != nil {
		return nil, err
	}
	sg := SSecurityGroup{}
	if err := resp.Unmarshal(&sg); err != nil {
		return nil, errors.Wrap(err, "Unmarshal security group")
	}
	sg.region = r
	return &sg, nil
}

func (r *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	name := opts.Name
	if name == "" {
		name = "sg-default"
	}
	poolID := regionIdToPoolId[r.RegionId]
	if poolID == "" {
		poolID = r.RegionId
	}
	// SDK 发送的 Body 为 createSecurityGroupBody 内层，与 VPC 一致
	body := jsonutils.NewDict()
	body.Set("name", jsonutils.NewString(name))
	body.Set("region", jsonutils.NewString(poolID))
	body.Set("type", jsonutils.NewString("VM"))
	if opts.Desc != "" {
		body.Set("description", jsonutils.NewString(opts.Desc))
	}
	base := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/SecurityGroup", nil, body).Base()
	base.SetMethod("POST")
	resp, err := r.client.request(context.Background(), base)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSecurityGroup")
	}
	sg := SSecurityGroup{}
	if err := resp.Unmarshal(&sg); err != nil {
		return nil, errors.Wrap(err, "Unmarshal created security group")
	}
	sg.region = r
	return &sg, nil
}

func (r *SRegion) DeleteSecurityGroup(id string) error {
	base := NewOpenApiVpcRequest(r.RegionId, fmt.Sprintf("/api/openapi-vpc/customer/v3/SecurityGroup/%s", id), nil, nil).Base()
	base.SetMethod("DELETE")
	_, err := r.client.request(context.Background(), base)
	return errors.Wrap(err, "DeleteSecurityGroup")
}

func (r *SRegion) GetSecurityGroupRules(sgId string) ([]SSecurityGroupRule, error) {
	query := map[string]string{"securityGroupId": sgId, "page": "1", "pageSize": "100"}
	req := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/SecurityGroupRule", query, nil)
	rules := make([]SSecurityGroupRule, 0)
	if err := r.client.doList(context.Background(), req.Base(), &rules); err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].region = r
		rules[i].SecgroupId = sgId
	}
	return rules, nil
}

func (r *SRegion) CreateSecurityGroupRule(sgId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) (*SSecurityGroupRule, error) {
	body := jsonutils.NewDict()
	body.Set("securityGroupId", jsonutils.NewString(sgId))
	body.Set("remoteType", jsonutils.NewString("cidr"))
	body.Set("direction", jsonutils.NewString(string(opts.Direction)))
	proto := strings.ToUpper(opts.Protocol)
	if proto == "" || proto == "ANY" {
		proto = "ANY"
	}
	body.Set("protocol", jsonutils.NewString(proto))
	if opts.CIDR != "" {
		body.Set("remoteIpPrefix", jsonutils.NewString(opts.CIDR))
	}
	if opts.Desc != "" {
		body.Set("description", jsonutils.NewString(opts.Desc))
	}
	minPort, maxPort := parsePorts(opts.Ports)
	if minPort >= 0 {
		body.Set("minPortRange", jsonutils.NewInt(int64(minPort)))
	}
	if maxPort >= 0 {
		body.Set("maxPortRange", jsonutils.NewInt(int64(maxPort)))
	}
	base := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/SecurityGroupRule", nil, body).Base()
	base.SetMethod("POST")
	resp, err := r.client.request(context.Background(), base)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSecurityGroupRule")
	}
	rule := SSecurityGroupRule{}
	if err := resp.Unmarshal(&rule); err != nil {
		return nil, errors.Wrap(err, "Unmarshal created rule")
	}
	rule.region = r
	rule.SecgroupId = sgId
	return &rule, nil
}

func (r *SRegion) DeleteSecurityGroupRule(ruleId string) error {
	base := NewOpenApiVpcRequest(r.RegionId, fmt.Sprintf("/api/openapi-vpc/customer/v3/SecurityGroupRule/%s", ruleId), nil, nil).Base()
	base.SetMethod("DELETE")
	_, err := r.client.request(context.Background(), base)
	return errors.Wrap(err, "DeleteSecurityGroupRule")
}

func (r *SRegion) UpdatePortSecurityGroups(portId string, securityGroupIds []string) error {
	body := jsonutils.NewDict()
	body.Set("id", jsonutils.NewString(portId))
	arr := jsonutils.NewArray()
	for _, id := range securityGroupIds {
		arr.Add(jsonutils.NewString(id))
	}
	body.Set("securityGroupIds", arr)
	reqBody := jsonutils.NewDict()
	reqBody.Set("updatePortSecurityGroupsBody", body)
	base := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/port/portSecurityGroups", nil, reqBody).Base()
	base.SetMethod("PUT")
	_, err := r.client.request(context.Background(), base)
	return errors.Wrap(err, "UpdatePortSecurityGroups")
}

func (r *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	sgs, err := r.GetSecurityGroups()
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudSecurityGroup, len(sgs))
	for i := range sgs {
		ret[i] = &sgs[i]
	}
	return ret, nil
}

func (r *SRegion) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	sg, err := r.GetSecurityGroup(id)
	if err != nil {
		return nil, err
	}
	return sg, nil
}

func (r *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	sg, err := r.CreateSecurityGroup(opts)
	if err != nil {
		return nil, err
	}
	return sg, nil
}

// parsePorts 解析 "80" 或 "80-443" 为 min, max；无法解析返回 -1,-1。
func parsePorts(ports string) (minPort, maxPort int32) {
	if ports == "" {
		return -1, -1
	}
	parts := strings.Split(ports, "-")
	if len(parts) == 1 {
		p, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil {
			return -1, -1
		}
		return int32(p), int32(p)
	}
	if len(parts) == 2 {
		p1, e1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
		p2, e2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
		if e1 != nil || e2 != nil {
			return -1, -1
		}
		return int32(p1), int32(p2)
	}
	return -1, -1
}
