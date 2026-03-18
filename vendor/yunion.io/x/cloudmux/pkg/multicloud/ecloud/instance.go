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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/imagetools"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SInstance struct {
	multicloud.SInstanceBase
	EcloudTags
	multicloud.SBillingBase
	SZoneRegionBase
	SCreateTime

	host *SHost

	region *SRegion

	billInfoFetched bool
	billInfo        *SInstanceBillInfo

	// OpenAPI v4/list/describe-instances 返回字段
	ActualSystemImage string `json:"actualSystemImage"`
	AdminPaused       bool   `json:"adminPaused"`
	AdminPausedTip    string `json:"adminPausedTip"`

	ZoneId string `json:"zoneId"`

	CredibleStatus string `json:"credibleStatus"`

	CrossRegionMigrate bool `json:"crossRegionMigrate"`

	Description string `json:"description"`

	GroupId string `json:"groupId"`

	InstanceId   string `json:"instanceId"`
	InstanceName string `json:"instanceName"`

	ImageId     string `json:"imageId"`
	ImageName   string `json:"imageName"`
	ImageOsType string `json:"imageOsType"`

	KeyName string `json:"keyName"`

	MainPortId string `json:"mainPortId"`
	MaxPorts   int    `json:"maxPorts"`
	MaxVolumes int    `json:"maxVolumes"`

	AvailableZone string `json:"availableZone"`

	ProductType string `json:"productType"`

	Recycle       bool   `json:"recycle"`
	RecycleCount  int    `json:"recycleCount"`
	RecycleStatus string `json:"recycleStatus"`
	RecycleTime   string `json:"recycleTime"`

	ReleaseProtect bool `json:"releaseProtect"`

	FlavorName   string `json:"flavorName"`
	ServerType   string `json:"serverType"`
	ServerVmType string `json:"serverVmType"`

	StoppedMode bool `json:"stoppedMode"`

	SupportVolumeMount string `json:"supportVolumeMount"`

	UserName string `json:"userName"`

	Vcpu    int `json:"vcpu"`
	Vmemory int `json:"vmemory"`

	Vdisk          int    `json:"vdisk"`
	SystemDiskId   string `json:"systemDiskId"`
	Status         string `json:"status"`
	BootVolumeType string `json:"bootVolumeType"`
}

func (i *SInstance) GetBillingType() string {
	region := i.getRegion()
	if region != nil && !i.billInfoFetched {
		i.billInfoFetched = true
		infos, err := region.GetInsatnceBillInfo(context.Background(), []string{i.GetId()})
		if err != nil {
			log.Debugf("ecloud GetInsatnceBillInfo(%s) error: %v", i.GetId(), err)
		} else {
			for idx := range infos {
				if infos[idx].InstanceId == i.GetId() {
					i.billInfo = &infos[idx]
					break
				}
			}
		}
	}
	if i.billInfo != nil {
		// chargingMode: 1=包周期(预付费) 2=按需(后付费)
		switch i.billInfo.ChargingMode {
		case 1:
			return billing_api.BILLING_TYPE_PREPAID
		case 2:
			return billing_api.BILLING_TYPE_POSTPAID
		}
	}
	// 查询失败或无返回时兜底按后付费处理
	return billing_api.BILLING_TYPE_POSTPAID
}

func (i *SInstance) getRegion() *SRegion {
	if i.host != nil && i.host.zone != nil && i.host.zone.region != nil {
		return i.host.zone.region
	}
	return i.region
}

func (i *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (i *SInstance) GetId() string {
	return i.InstanceId
}

func (i *SInstance) GetName() string {
	return i.InstanceName
}

func (i *SInstance) GetHostname() string {
	return ""
}

func (i *SInstance) GetGlobalId() string {
	return i.GetId()
}

func (i *SInstance) GetStatus() string {
	switch i.Status {
	case "active":
		return api.VM_RUNNING
	case "suspended", "paused":
		return api.VM_SUSPEND
	case "build", "rebuild", "resize", "verify_resize", "revert_resize", "password":
		return api.VM_STARTING
	case "reboot", "hard_reboot":
		return api.VM_STOPPING
	case "stopped", "shutoff":
		return api.VM_READY
	case "migrating":
		return api.VM_MIGRATING
	case "backuping":
		return api.VM_BACKUP_CREATING
	default:
		return api.VM_UNKNOWN
	}
}

func (i *SInstance) Refresh() error {
	vm, err := i.host.zone.region.GetInstance(i.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(i, vm)
}

func (self *SInstance) GetBootOrder() string {
	return "dcn"
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (i *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(i.ImageOsType)
}

func (i *SInstance) GetFullOsName() string {
	return i.ImageName
}

func (i *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.BIOS
}

func (i *SInstance) GetOsArch() string {
	return ""
}

func (i *SInstance) GetOsDist() string {
	osInfo := imagetools.NormalizeImageInfo(i.ImageName, "", i.ImageOsType, "", "")
	return osInfo.OsDistro
}

func (i *SInstance) GetOsVersion() string {
	osInfo := imagetools.NormalizeImageInfo(i.ImageName, "", i.ImageOsType, "", "")
	return osInfo.OsVersion
}

func (i *SInstance) GetOsLang() string {
	osInfo := imagetools.NormalizeImageInfo(i.ImageName, "", i.ImageOsType, "", "")
	return osInfo.OsLang
}

func (i *SInstance) GetMachine() string {
	return "pc"
}

func (i *SInstance) GetInstanceType() string {
	return i.FlavorName
}

func (in *SInstance) GetProjectId() string {
	return ""
}

func (in *SInstance) GetIHost() cloudprovider.ICloudHost {
	return in.host
}

func (in *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	sysDisk, err := in.GetSysDisk()
	if err != nil {
		return nil, err
	}
	dataDisks, err := in.GetDataDisks()
	if err != nil {
		return nil, err
	}
	return append([]cloudprovider.ICloudDisk{sysDisk}, dataDisks...), nil
}

func (in *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	region := in.getRegion()
	if region == nil {
		return []cloudprovider.ICloudNic{}, nil
	}
	nics, err := region.GetInstanceNics(context.Background(), in.GetId())
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudNic, len(nics))
	for i := range nics {
		nics[i].instance = in
		ret[i] = &nics[i]
	}
	return ret, nil
}

func (in *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	nics, err := in.host.zone.region.GetInstanceNics(context.Background(), in.GetId())
	if err != nil {
		return nil, err
	}
	for i := range nics {
		if len(nics[i].FipAddress) == 0 {
			continue
		}
		eip, err := in.getRegion().GetEipByAddr(nics[i].FipAddress)
		if err != nil {
			return nil, err
		}
		return eip, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (in *SInstance) GetSecurityGroupIds() ([]string, error) {
	nics, err := in.host.zone.region.GetInstanceNics(context.Background(), in.GetId())
	if err != nil {
		return nil, err
	}
	securityGroupIds := make([]string, 0)
	for i := range nics {
		for _, sg := range nics[i].SecurityGroups {
			securityGroupIds = append(securityGroupIds, sg.Id)
		}
	}
	return securityGroupIds, nil
}

func (in *SInstance) GetVcpuCount() int {
	return in.Vcpu
}

func (in *SInstance) GetVmemSizeMB() int {
	return in.Vmemory
}

func (in *SInstance) SetSecurityGroups(ids []string) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_ECLOUD
}

func (in *SInstance) StartVM(ctx context.Context) error {
	region := in.getRegion()
	if region == nil {
		return errors.Wrap(cloudprovider.ErrNotImplemented, "missing region")
	}
	return region.StartInstance(ctx, in.GetId())
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	region := self.getRegion()
	if region == nil {
		return errors.Wrap(cloudprovider.ErrNotImplemented, "missing region")
	}
	return region.StopInstance(ctx, self.GetId())
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	region := self.getRegion()
	if region == nil {
		return errors.Wrap(cloudprovider.ErrNotImplemented, "missing region")
	}
	return region.DeleteInstance(ctx, self.GetId(), false, false)
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return errors.ErrNotImplemented
}

func (in *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	url, err := in.host.zone.region.GetInstanceVNCUrl(in.GetId())
	if err != nil {
		return nil, err
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:        url,
		Protocol:   "ecloud",
		InstanceId: in.GetId(),
		Hypervisor: api.HYPERVISOR_ECLOUD,
	}
	return ret, nil
}

func (in *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetError() error {
	return nil
}

func (in *SInstance) GetSysDisk() (cloudprovider.ICloudDisk, error) {
	storage, err := in.host.zone.GetStorageByType(storageTypeConstMap[in.BootVolumeType])
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorageByType(%s)", in.BootVolumeType)
	}
	disk := &SDisk{
		storage: storage,
		ManualAttr: SDiskManualAttr{
			IsVirtual:  true,
			TemplateId: in.ImageId,
			ServerId:   in.InstanceId,
		},
		SCreateTime:     in.SCreateTime,
		SZoneRegionBase: in.SZoneRegionBase,
		ServerId:        []string{in.InstanceId},
		IsShare:         false,
		IsDelete:        false,
		SizeGB:          in.Vdisk,
		ID:              in.SystemDiskId,
		Name:            fmt.Sprintf("%s-root", in.InstanceName),
		Status:          "in-use",
		Type:            api.STORAGE_ECLOUD_SYSTEM,
	}
	return disk, nil
}

func (region *SRegion) GetDataDisks(serverId string) ([]SDisk, error) {
	request := NewOpenApiEbsRequest(region.RegionId, "/api/ebs/acl/v3/volume/mount/list", map[string]string{"serverId": serverId}, nil)
	var ret []SDisk
	err := region.client.doList(context.Background(), request.Base(), &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDataDisks")
	}
	return ret, nil
}

func (in *SInstance) GetDataDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := in.host.zone.region.GetDataDisks(in.InstanceId)
	if err != nil {
		return nil, err
	}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := range disks {
		storage, err := in.host.zone.GetStorageByType(disks[i].Type)
		if err != nil {
			return nil, errors.Wrapf(err, "GetStorageByType(%s)", disks[i].Type)
		}
		disks[i].storage = storage
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (r *SRegion) GetInstances(zoneId string, serverId string) ([]SInstance, error) {
	body := jsonutils.NewDict()
	if len(zoneId) > 0 {
		body.Add(jsonutils.NewString(zoneId), "zoneId")
	}
	if len(serverId) > 0 {
		body.Add(jsonutils.NewString(serverId), "instanceId")
	}
	req := NewOpenApiInstanceRequest(r.RegionId, body)
	ret := make([]SInstance, 0)
	err := r.client.doPostList(context.Background(), req.Base(), &ret)
	if err != nil {
		return nil, err
	}
	for i := range ret {
		ret[i].region = r
	}
	return ret, nil
}

func (r *SRegion) GetInstance(id string) (*SInstance, error) {
	instances, err := r.GetInstances("", id)
	if err != nil {
		return nil, err
	}
	for i := range instances {
		if instances[i].InstanceId == id {
			instances[i].region = r
			return &instances[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetInstanceVNCUrl(instanceId string) (string, error) {
	req := NewOpenApiInstanceActionRequest(r.RegionId, "/api/openapi-instance/v4/vnc-url", nil)
	base := req.Base()
	base.SetMethod("GET")
	base.GetQueryParams()["instanceId"] = instanceId
	resp := struct {
		VncUrl string `json:"vncUrl"`
	}{}
	if err := r.client.doGet(context.Background(), base, &resp); err != nil {
		return "", err
	}
	return resp.VncUrl, nil
}

type sInstanceBatchResult struct {
	Result     bool   `json:"result"`
	InstanceId string `json:"instanceId"`
	Message    string `json:"message"`
}

type sInstanceBatchResp struct {
	InstanceBatchResult []sInstanceBatchResult `json:"instanceBatchResult"`
}

func (r *SRegion) StartInstance(ctx context.Context, instanceId string) error {
	body := jsonutils.NewDict()
	body.Set("batchStartInstancesBody", jsonutils.Marshal(map[string][]string{
		"instanceIds": {instanceId},
	}))
	req := NewOpenApiInstanceActionRequest(r.RegionId, "/api/openapi-instance/v4/batch-start-instances", body)
	resp := sInstanceBatchResp{}
	if err := r.client.doPost(ctx, req.Base(), &resp); err != nil {
		return err
	}
	for i := range resp.InstanceBatchResult {
		if resp.InstanceBatchResult[i].InstanceId != instanceId {
			continue
		}
		if !resp.InstanceBatchResult[i].Result {
			return errors.Errorf("start instance %s failed: %s", instanceId, resp.InstanceBatchResult[i].Message)
		}
		return nil
	}
	return nil
}

func (r *SRegion) StopInstance(ctx context.Context, instanceId string) error {
	body := jsonutils.NewDict()
	body.Set("batchStopInstancesBody", jsonutils.Marshal(map[string][]string{
		"instanceIds": {instanceId},
	}))
	req := NewOpenApiInstanceActionRequest(r.RegionId, "/api/openapi-instance/v4/batch-stop-instances", body)
	resp := sInstanceBatchResp{}
	if err := r.client.doPost(ctx, req.Base(), &resp); err != nil {
		return err
	}
	for i := range resp.InstanceBatchResult {
		if resp.InstanceBatchResult[i].InstanceId != instanceId {
			continue
		}
		if !resp.InstanceBatchResult[i].Result {
			return errors.Errorf("stop instance %s failed: %s", instanceId, resp.InstanceBatchResult[i].Message)
		}
		return nil
	}
	return nil
}

func (r *SRegion) DeleteInstance(ctx context.Context, instanceId string, deletePublicNetwork, deleteDataVolumes bool) error {
	body := jsonutils.NewDict()
	body.Set("deleteInstancesBody", jsonutils.Marshal(map[string]interface{}{
		"instanceIds":         []string{instanceId},
		"deletePublicNetwork": deletePublicNetwork,
		"deleteDataVolumes":   deleteDataVolumes,
		"dryRun":              false,
	}))
	req := NewOpenApiInstanceActionRequest(r.RegionId, "/api/openapi-instance/v4/delete-instances", body)
	resp := sInstanceBatchResp{}
	if err := r.client.doPost(ctx, req.Base(), &resp); err != nil {
		return err
	}
	for i := range resp.InstanceBatchResult {
		if resp.InstanceBatchResult[i].InstanceId != instanceId {
			continue
		}
		if !resp.InstanceBatchResult[i].Result {
			return errors.Errorf("delete instance %s failed: %s", instanceId, resp.InstanceBatchResult[i].Message)
		}
		return nil
	}
	return nil
}

type sDescribeVirtualNetworksResp struct {
	MacAddress    string `json:"macAddress"`
	Name          string `json:"name"`
	BandwidthSize int    `json:"bandwidthSize"`
	FixedIpResps  []struct {
		SubnetId   string `json:"subnetId"`
		SubnetCidr string `json:"subnetCidr"`
		VpcName    string `json:"vpcName"`
		IpVersion  int32  `json:"ipVersion"`
		VpcId      string `json:"vpcId"`
		IpAddress  string `json:"ipAddress"`
		SubnetName string `json:"subnetName"`
	} `json:"fixedIpResps"`
	CreatedTime        string `json:"createdTime"`
	PublicIp           string `json:"publicIp"`
	Id                 string `json:"id"`
	SubnetName         string `json:"subnetName"`
	SecurityGroupResps []struct {
		Name        string `json:"name"`
		CreatedTime string `json:"createdTime"`
		Description string `json:"description"`
		Id          string `json:"id"`
	} `json:"securityGroupResps"`
}

// GetInstanceNics 查询实例绑定的网卡详情列表：
// GET /api/openapi-instance/v4/virtual-networks?instanceId=xxx&page=1&pageSize=100
func (r *SRegion) GetInstanceNics(ctx context.Context, instanceId string) ([]SInstanceNic, error) {
	req := NewOpenApiInstanceActionRequest(r.RegionId, "/api/openapi-instance/v4/virtual-networks", nil)
	query := req.Base().GetQueryParams()
	query["instanceId"] = instanceId
	nets := make([]sDescribeVirtualNetworksResp, 0)
	if err := r.client.doList(ctx, req.Base(), &nets); err != nil {
		return nil, err
	}
	ret := make([]SInstanceNic, 0, len(nets))
	for i := range nets {
		n := nets[i]
		nic := SInstanceNic{
			Id:               n.Id,
			PortId:           n.Id,
			PortName:         n.Name,
			PrivateIp:        "",
			FipAddress:       n.PublicIp,
			FipBandwidthSize: n.BandwidthSize,
		}
		nic.MacAddress = n.MacAddress
		nic.PublicIp = n.PublicIp
		for j := range n.SecurityGroupResps {
			sg := n.SecurityGroupResps[j]
			nic.SecurityGroups = append(nic.SecurityGroups, SSecurityGroupRef{
				Id:   sg.Id,
				Name: sg.Name,
			})
		}
		for j := range n.FixedIpResps {
			ip := n.FixedIpResps[j]
			if nic.PrivateIp == "" && ip.IpAddress != "" {
				nic.PrivateIp = ip.IpAddress
			}
			if nic.NetworkId == "" && ip.SubnetId != "" {
				nic.NetworkId = ip.SubnetId
			}
			nic.FixedIpDetails = append(nic.FixedIpDetails, SFixedIpDetail{
				IpAddress: ip.IpAddress,
				IpVersion: fmt.Sprintf("%d", ip.IpVersion),
				SubnetId:  ip.SubnetId,
				SubnetName: func() string {
					if ip.SubnetName != "" {
						return ip.SubnetName
					}
					return n.SubnetName
				}(),
			})
		}
		ret = append(ret, nic)
	}
	return ret, nil
}

type SInstanceBillInfo struct {
	AutoEndTime  string `json:"autoEndTime"`
	PriceUnit    string `json:"priceUnit"`
	ChargingMode int32  `json:"chargingMode"`
	RespDesc     string `json:"respDesc"`
	InstanceId   string `json:"instanceId"`
	AutoRenew    string `json:"autoRenew"`
	EndTime      string `json:"endTime"`
}

// GetInstanceBillInfo 查询实例计费信息：
// POST /api/openapi-instance/v4/batch-query-price-info
func (r *SRegion) GetInstanceBillInfo(ctx context.Context, instanceIds []string) ([]SInstanceBillInfo, error) {
	if len(instanceIds) == 0 {
		return []SInstanceBillInfo{}, nil
	}
	body := jsonutils.NewDict()
	body.Set("batchQueryPriceInfoBody", jsonutils.Marshal(map[string][]string{
		"instanceIds": instanceIds,
	}))
	req := NewOpenApiInstanceActionRequest(r.RegionId, "/api/openapi-instance/v4/batch-query-price-info", body)
	ret := make([]SInstanceBillInfo, 0)
	if err := r.client.doPost(ctx, req.Base(), &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// GetInsatnceBillInfo 保持兼容（历史拼写错误）
func (r *SRegion) GetInsatnceBillInfo(ctx context.Context, instanceIds []string) ([]SInstanceBillInfo, error) {
	return r.GetInstanceBillInfo(ctx, instanceIds)
}
