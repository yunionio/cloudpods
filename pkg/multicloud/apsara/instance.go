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

package apsara

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	// Running：运行中
	//Starting：启动中
	//Stopping：停止中
	//Stopped：已停止

	InstanceStatusStopped  = "Stopped"
	InstanceStatusRunning  = "Running"
	InstanceStatusStopping = "Stopping"
	InstanceStatusStarting = "Starting"
)

type SDedicatedHostAttribute struct {
	DedicatedHostId   string
	DedicatedHostName string
}

type SIpAddress struct {
	IpAddress []string
}

type SNetworkInterfaces struct {
	NetworkInterface []SNetworkInterface
}

type SOperationLocks struct {
	LockReason []string
}

type SSecurityGroupIds struct {
	SecurityGroupId []string
}

// {"NatIpAddress":"","PrivateIpAddress":{"IpAddress":["192.168.220.214"]},"VSwitchId":"vsw-2ze9cqwza4upoyujq1thd","VpcId":"vpc-2zer4jy8ix3i8f0coc5uw"}

type SVpcAttributes struct {
	NatIpAddress     string
	PrivateIpAddress SIpAddress
	VSwitchId        string
	VpcId            string
}

type SInstance struct {
	multicloud.SInstanceBase

	host *SHost

	// idisks []cloudprovider.ICloudDisk

	AutoReleaseTime         string
	ClusterId               string
	Cpu                     int
	CreationTime            time.Time
	DedicatedHostAttribute  SDedicatedHostAttribute
	Description             string
	DeviceAvailable         bool
	EipAddress              SEipAddress
	ExpiredTime             time.Time
	GPUAmount               int
	GPUSpec                 string
	HostName                string
	ImageId                 string
	InnerIpAddress          SIpAddress
	InstanceChargeType      TChargeType
	InstanceId              string
	InstanceName            string
	InstanceNetworkType     string
	InstanceType            string
	InstanceTypeFamily      string
	InternetChargeType      TInternetChargeType
	InternetMaxBandwidthIn  int
	InternetMaxBandwidthOut int
	IoOptimized             bool
	KeyPairName             string
	Memory                  int
	NetworkInterfaces       SNetworkInterfaces
	OSName                  string
	OSType                  string
	OperationLocks          SOperationLocks
	PublicIpAddress         SIpAddress
	Recyclable              bool
	RegionId                string
	ResourceGroupId         string
	SaleCycle               string
	SecurityGroupIds        SSecurityGroupIds
	SerialNumber            string
	SpotPriceLimit          string
	SpotStrategy            string
	StartTime               time.Time
	Status                  string
	StoppedMode             string
	VlanId                  string
	VpcAttributes           SVpcAttributes
	ZoneId                  string
}

// {"AutoReleaseTime":"","ClusterId":"","Cpu":1,"CreationTime":"2018-05-23T07:58Z","DedicatedHostAttribute":{"DedicatedHostId":"","DedicatedHostName":""},"Description":"","DeviceAvailable":true,"EipAddress":{"AllocationId":"","InternetChargeType":"","IpAddress":""},"ExpiredTime":"2018-05-30T16:00Z","GPUAmount":0,"GPUSpec":"","HostName":"iZ2ze57isp1ali72tzkjowZ","ImageId":"centos_7_04_64_20G_alibase_201701015.vhd","InnerIpAddress":{"IpAddress":[]},"InstanceChargeType":"PrePaid","InstanceId":"i-2ze57isp1ali72tzkjow","InstanceName":"gaoxianqi-test-7days","InstanceNetworkType":"vpc","InstanceType":"ecs.t5-lc2m1.nano","InstanceTypeFamily":"ecs.t5","InternetChargeType":"PayByBandwidth","InternetMaxBandwidthIn":-1,"InternetMaxBandwidthOut":0,"IoOptimized":true,"Memory":512,"NetworkInterfaces":{"NetworkInterface":[{"MacAddress":"00:16:3e:10:f0:c9","NetworkInterfaceId":"eni-2zecqsagtpztl6x5hu2r","PrimaryIpAddress":"192.168.220.214"}]},"OSName":"CentOS  7.4 64位","OSType":"linux","OperationLocks":{"LockReason":[]},"PublicIpAddress":{"IpAddress":[]},"Recyclable":false,"RegionId":"cn-beijing","ResourceGroupId":"","SaleCycle":"Week","SecurityGroupIds":{"SecurityGroupId":["sg-2zecqsagtpztl6x9zynl"]},"SerialNumber":"df05d9b4-df3d-4400-88d1-5f843f0dd088","SpotPriceLimit":0.000000,"SpotStrategy":"NoSpot","StartTime":"2018-05-23T07:58Z","Status":"Running","StoppedMode":"Not-applicable","VlanId":"","VpcAttributes":{"NatIpAddress":"","PrivateIpAddress":{"IpAddress":["192.168.220.214"]},"VSwitchId":"vsw-2ze9cqwza4upoyujq1thd","VpcId":"vpc-2zer4jy8ix3i8f0coc5uw"},"ZoneId":"cn-beijing-f"}

func (self *SRegion) GetInstances(zoneId string, ids []string, offset int, limit int) ([]SInstance, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	if len(zoneId) > 0 {
		params["ZoneId"] = zoneId
	}

	if ids != nil && len(ids) > 0 {
		params["InstanceIds"] = jsonutils.Marshal(ids).String()
	}

	body, err := self.ecsRequest("DescribeInstances", params)
	if err != nil {
		log.Errorf("GetInstances fail %s", err)
		return nil, 0, err
	}

	instances := make([]SInstance, 0)
	err = body.Unmarshal(&instances, "Instances", "Instance")
	if err != nil {
		log.Errorf("Unmarshal security group details fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return instances, int(total), nil
}

func (self *SRegion) fetchTags(resourceType string, resourceId string) (*jsonutils.JSONDict, error) {
	// 资源类型。取值范围：
	// disk
	// instance
	// image
	// securitygroup
	// snapshot
	var page int64 = 1
	var pageSize int64 = 50
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["ResourceType"] = resourceType
	params["ResourceId"] = resourceId
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", page)
	ret, err := self.ecsRequest("DescribeTags", params)
	if err != nil {
		return nil, err
	}

	tags := jsonutils.NewDict()
	result, _ := ret.GetArray("Tags", "Tag")
	for _, item := range result {
		k, _ := item.GetString("TagKey")
		v, _ := item.Get("TagValue")
		if len(k) > 0 {
			tags.Set(k, v)
		}
	}

	total, _ := ret.Int("TotalCount")
	for ; total > page*pageSize; page++ {
		params["PageSize"] = fmt.Sprintf("%d", pageSize)
		params["PageNumber"] = fmt.Sprintf("%d", page)
		ret, err := self.ecsRequest("DescribeTags", params)
		if err != nil {
			return nil, err
		}

		result, _ := ret.GetArray("Tags", "Tag")
		for _, item := range result {
			k, _ := item.GetString("TagKey")
			v, _ := item.Get("TagValue")
			if len(k) > 0 {
				tags.Set(k, v)
			}
		}
	}

	return tags, nil
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return self.SecurityGroupIds.SecurityGroupId, nil
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()

	// The pricingInfo key structure is 'RegionId::InstanceType::NetworkType::OSType::IoOptimized'
	optimized := "optimized"
	if !self.IoOptimized {
		optimized = "none"
	}
	priceKey := fmt.Sprintf("%s::%s::%s::%s::%s", self.RegionId, self.InstanceType, self.InstanceNetworkType, self.OSType, optimized)
	data.Add(jsonutils.NewString(priceKey), "price_key")

	tags, err := self.host.zone.region.fetchTags("instance", self.InstanceId)
	if err != nil {
		log.Errorln(err)
	}
	data.Update(tags)

	data.Add(jsonutils.NewString(self.host.zone.GetGlobalId()), "zone_ext_id")
	if len(self.ImageId) > 0 {
		if image, err := self.host.zone.region.GetImage(self.ImageId); err != nil {
			log.Errorf("Failed to find image %s for instance %s", self.ImageId, self.GetName())
		} else if meta := image.GetMetadata(); meta != nil {
			data.Update(meta)
		}
	}
	return data
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetId() string {
	return self.InstanceId
}

func (self *SInstance) GetName() string {
	if len(self.InstanceName) > 0 {
		return self.InstanceName
	}
	return self.HostName
}

func (self *SInstance) GetGlobalId() string {
	return self.InstanceId
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetInstanceType() string {
	return self.InstanceType
}

func (self *SInstance) getVpc() (*SVpc, error) {
	return self.host.zone.region.getVpc(self.VpcAttributes.VpcId)
}

type byAttachedTime []SDisk

func (a byAttachedTime) Len() int      { return len(a) }
func (a byAttachedTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byAttachedTime) Less(i, j int) bool {
	switch a[i].GetDiskType() {
	case api.DISK_TYPE_SYS:
		return true
	case api.DISK_TYPE_SWAP:
		switch a[j].GetDiskType() {
		case api.DISK_TYPE_SYS:
			return false
		case api.DISK_TYPE_DATA:
			return true
		}
	case api.DISK_TYPE_DATA:
		if a[j].GetDiskType() != api.DISK_TYPE_DATA {
			return false
		}
	}
	return a[i].AttachedTime.Before(a[j].AttachedTime)
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := []SDisk{}
	for {
		part, total, err := self.host.zone.region.GetDisks(self.InstanceId, "", "", nil, len(disks), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDisks for %s", self.InstanceId)
		}
		disks = append(disks, part...)
		if len(disks) >= total {
			break
		}
	}

	sort.Sort(byAttachedTime(disks))

	log.Debugf("%s", jsonutils.Marshal(&disks))

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		store, err := self.host.zone.getStorageByCategory(disks[i].Category)
		if err != nil {
			return nil, err
		}
		disks[i].storage = store
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	var (
		networkInterfaces = self.NetworkInterfaces.NetworkInterface
		nics              []cloudprovider.ICloudNic
	)
	for _, networkInterface := range networkInterfaces {
		nic := SInstanceNic{
			instance: self,
			ipAddr:   networkInterface.PrimaryIpAddress,
			macAddr:  networkInterface.MacAddress,
		}
		nics = append(nics, &nic)
	}
	return nics, nil
}

func (self *SInstance) GetVcpuCount() int {
	return self.Cpu
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.Memory
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

func (self *SInstance) GetOSType() string {
	return osprofile.NormalizeOSType(self.OSType)
}

func (self *SInstance) GetOSName() string {
	return self.OSName
}

func (self *SInstance) GetBios() string {
	return "BIOS"
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetStatus() string {
	// Running：运行中
	//Starting：启动中
	//Stopping：停止中
	//Stopped：已停止
	switch self.Status {
	case InstanceStatusRunning:
		return api.VM_RUNNING
	case InstanceStatusStarting:
		return api.VM_STARTING
	case InstanceStatusStopping:
		return api.VM_STOPPING
	case InstanceStatusStopped:
		return api.VM_READY
	default:
		return api.VM_UNKNOWN
	}
}

func (self *SInstance) Refresh() error {
	new, err := self.host.zone.region.GetInstance(self.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

/*
func (self *SInstance) GetRemoteStatus() string {
	// Running：运行中
	//Starting：启动中
	//Stopping：停止中
	//Stopped：已停止
	switch self.Status {
	case InstanceStatusRunning:
		return cloudprovider.CloudVMStatusRunning
	case InstanceStatusStarting:
		return cloudprovider.CloudVMStatusStopped
	case InstanceStatusStopping:
		return cloudprovider.CloudVMStatusRunning
	case InstanceStatusStopped:
		return cloudprovider.CloudVMStatusStopped
	default:
		return cloudprovider.CloudVMStatusOther
	}
}
*/

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_APSARA
}

func (self *SInstance) StartVM(ctx context.Context) error {
	timeout := 300 * time.Second
	interval := 15 * time.Second

	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := self.Refresh()
		if err != nil {
			return err
		}
		log.Debugf("status %s expect %s", self.GetStatus(), api.VM_RUNNING)
		if self.GetStatus() == api.VM_RUNNING {
			return nil
		} else if self.GetStatus() == api.VM_READY {
			err := self.host.zone.region.StartVM(self.InstanceId)
			if err != nil {
				return err
			}
		}
		time.Sleep(interval)
	}
	return cloudprovider.ErrTimeout
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := self.host.zone.region.StopVM(self.InstanceId, opts.IsForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	url, err := self.host.zone.region.GetInstanceVNCUrl(self.InstanceId)
	if err != nil {
		return nil, err
	}
	passwd := seclib.RandomPassword(6)
	err = self.host.zone.region.ModifyInstanceVNCUrlPassword(self.InstanceId, passwd)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(url), "url")
	ret.Add(jsonutils.NewString(passwd), "password")
	ret.Add(jsonutils.NewString("apsara"), "protocol")
	ret.Add(jsonutils.NewString(self.InstanceId), "instance_id")
	return ret, nil
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return self.host.zone.region.UpdateVM(self.InstanceId, name, self.OSType)
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	var keypairName string
	if len(publicKey) > 0 {
		var err error
		keypairName, err = self.host.zone.region.syncKeypair(publicKey)
		if err != nil {
			return err
		}
	}

	return self.host.zone.region.DeployVM(self.InstanceId, name, password, keypairName, deleteKeypair, description)
}

func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	keypair := ""
	if len(desc.PublicKey) > 0 {
		var err error
		keypair, err = self.host.zone.region.syncKeypair(desc.PublicKey)
		if err != nil {
			return "", err
		}
	}
	diskId, err := self.host.zone.region.ReplaceSystemDisk(self.InstanceId, desc.ImageId, desc.Password, keypair, desc.SysSizeGB)
	if err != nil {
		return "", err
	}

	return diskId, nil
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if len(config.InstanceType) > 0 {
		return self.host.zone.region.ChangeVMConfig2(self.ZoneId, self.InstanceId, config.InstanceType, nil)
	}
	return self.host.zone.region.ChangeVMConfig(self.ZoneId, self.InstanceId, config.Cpu, config.MemoryMB, nil)
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.AttachDisk(self.InstanceId, diskId)
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.RetryOnError(
		func() error {
			return self.host.zone.region.DetachDisk(self.InstanceId, diskId)
		},
		[]string{
			`"Code":"InvalidOperation.Conflict"`,
		},
		4)
}

func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, _, err := self.GetInstances("", []string{instanceId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &instances[0], nil
}

func (self *SRegion) CreateInstance(name string, imageId string, instanceType string, securityGroupId string,
	zoneId string, desc string, passwd string, disks []SDisk, vSwitchId string, ipAddr string,
	keypair string, userData string, bc *billing.SBillingCycle, projectId, osType string,
	tags map[string]string,
) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["ImageId"] = imageId
	params["InstanceType"] = instanceType
	params["SecurityGroupId"] = securityGroupId
	params["ZoneId"] = zoneId
	params["InstanceName"] = name
	params["Description"] = desc
	params["InternetChargeType"] = "PayByTraffic"
	params["InternetMaxBandwidthIn"] = "200"
	params["InternetMaxBandwidthOut"] = "100"
	params["HostName"] = stringutils2.GenerateHostName(name, osType)
	if len(passwd) > 0 {
		params["Password"] = passwd
	} else {
		params["PasswordInherit"] = "True"
	}

	if len(projectId) > 0 {
		params["ResourceGroupId"] = projectId
	}
	//{"Code":"InvalidSystemDiskCategory.ValueNotSupported","HostId":"ecs.apsaracs.com","Message":"The specified parameter 'SystemDisk.Category' is not support IoOptimized Instance. Valid Values: cloud_efficiency;cloud_ssd. ","RequestId":"9C9A4E99-5196-42A2-80B6-4762F8F75C90"}
	params["IoOptimized"] = "optimized"
	for i, d := range disks {
		if i == 0 {
			params["SystemDisk.Category"] = d.Category
			if d.Category == api.STORAGE_CLOUD_ESSD_PL2 {
				params["SystemDisk.Category"] = api.STORAGE_CLOUD_ESSD
				params["SystemDisk.PerformanceLevel"] = "PL2"
			}
			if d.Category == api.STORAGE_CLOUD_ESSD_PL3 {
				params["SystemDisk.Category"] = api.STORAGE_CLOUD_ESSD
				params["SystemDisk.PerformanceLevel"] = "PL3"
			}
			params["SystemDisk.Size"] = fmt.Sprintf("%d", d.Size)
			params["SystemDisk.DiskName"] = d.GetName()
			params["SystemDisk.Description"] = d.Description
		} else {
			params[fmt.Sprintf("DataDisk.%d.Size", i)] = fmt.Sprintf("%d", d.Size)
			params[fmt.Sprintf("DataDisk.%d.Category", i)] = d.Category
			if d.Category == api.STORAGE_CLOUD_ESSD_PL2 {
				params[fmt.Sprintf("DataDisk.%d.Category", i)] = api.STORAGE_CLOUD_ESSD
				params[fmt.Sprintf("DataDisk.%d..PerformanceLevel", i)] = "PL2"
			}
			if d.Category == api.STORAGE_CLOUD_ESSD_PL3 {
				params[fmt.Sprintf("DataDisk.%d.Category", i)] = api.STORAGE_CLOUD_ESSD
				params[fmt.Sprintf("DataDisk.%d..PerformanceLevel", i)] = "PL3"
			}
			params[fmt.Sprintf("DataDisk.%d.DiskName", i)] = d.GetName()
			params[fmt.Sprintf("DataDisk.%d.Description", i)] = d.Description
			params[fmt.Sprintf("DataDisk.%d.Encrypted", i)] = "false"
		}
	}
	params["VSwitchId"] = vSwitchId
	params["PrivateIpAddress"] = ipAddr

	if len(keypair) > 0 {
		params["KeyPairName"] = keypair
	}

	if len(userData) > 0 {
		params["UserData"] = userData
	}

	if len(tags) > 0 {
		tagIdx := 0
		for k, v := range tags {
			params[fmt.Sprintf("Tag.%d.Key", tagIdx)] = k
			params[fmt.Sprintf("Tag.%d.Value", tagIdx)] = v
			tagIdx += 1
		}
	}

	if bc != nil {
		params["InstanceChargeType"] = "PrePaid"
		err := billingCycle2Params(bc, params)
		if err != nil {
			return "", err
		}
		if bc.AutoRenew {
			params["AutoRenew"] = "true"
			params["AutoRenewPeriod"] = "1"
		} else {
			params["AutoRenew"] = "False"
		}
	} else {
		params["InstanceChargeType"] = "PostPaid"
		params["SpotStrategy"] = "NoSpot"
	}

	params["ClientToken"] = utils.GenRequestId(20)

	body, err := self.ecsRequest("CreateInstance", params)
	if err != nil {
		log.Errorf("CreateInstance fail %s", err)
		return "", err
	}
	instanceId, _ := body.GetString("InstanceId")
	return instanceId, nil
}

func (self *SRegion) doStartVM(instanceId string) error {
	return self.instanceOperation(instanceId, "StartInstance", nil)
}

func (self *SRegion) doStopVM(instanceId string, isForce bool) error {
	params := make(map[string]string)
	if isForce {
		params["ForceStop"] = "true"
	} else {
		params["ForceStop"] = "false"
	}
	params["StoppedMode"] = "KeepCharging"
	return self.instanceOperation(instanceId, "StopInstance", params)
}

func (self *SRegion) doDeleteVM(instanceId string) error {
	params := make(map[string]string)
	params["TerminateSubscription"] = "true" // terminate expired prepaid instance
	params["Force"] = "true"
	return self.instanceOperation(instanceId, "DeleteInstance", params)
}

/*func (self *SRegion) waitInstanceStatus(instanceId string, target string, interval time.Duration, timeout time.Duration) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		status, err := self.GetInstanceStatus(instanceId)
		if err != nil {
			return err
		}
		if status == target {
			return nil
		}
		time.Sleep(interval)
	}
	return cloudprovider.ErrTimeout
}

func (self *SInstance) waitStatus(target string, interval time.Duration, timeout time.Duration) error {
	return self.host.zone.region.waitInstanceStatus(self.InstanceId, target, interval, timeout)
}*/

func (self *SRegion) StartVM(instanceId string) error {
	status, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status on StartVM: %s", err)
		return err
	}
	if status != InstanceStatusStopped {
		log.Errorf("StartVM: vm status is %s expect %s", status, InstanceStatusStopped)
		return cloudprovider.ErrInvalidStatus
	}
	return self.doStartVM(instanceId)
	// if err != nil {
	//	return err
	// }
	// return self.waitInstanceStatus(instanceId, InstanceStatusRunning, time.Second*5, time.Second*180) // 3 minutes to timeout
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	status, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status on StopVM: %s", err)
		return err
	}
	if status == InstanceStatusStopped {
		return nil
	}
	if status != InstanceStatusRunning {
		log.Errorf("StopVM: vm status is %s expect %s", status, InstanceStatusRunning)
		return cloudprovider.ErrInvalidStatus
	}
	return self.doStopVM(instanceId, isForce)
	// if err != nil {
	//  return err
	// }
	// return self.waitInstanceStatus(instanceId, InstanceStatusStopped, time.Second*10, time.Second*300) // 5 minutes to timeout
}

func (self *SRegion) DeleteVM(instanceId string) error {
	status, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status on DeleteVM: %s", err)
		return err
	}
	log.Debugf("Instance status on delete is %s", status)
	if status != InstanceStatusStopped {
		log.Warningf("DeleteVM: vm status is %s expect %s", status, InstanceStatusStopped)
	}
	return self.doDeleteVM(instanceId)
	// if err != nil {
	// 	return err
	// }
	// err = self.waitInstanceStatus(instanceId, InstanceStatusRunning, time.Second*10, time.Second*300) // 5 minutes to timeout
	// if err == cloudprovider.ErrNotFound {
	// 	return nil
	// } else if err == nil {
	// 	return cloudprovider.ErrTimeout
	// } else {
	// 	return err
	// }
}

func (self *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return err
	}

	// 修改密钥时直接返回
	if deleteKeypair {
		err = self.DetachKeyPair(instanceId, instance.KeyPairName)
		if err != nil {
			return err
		}
	}

	if len(keypairName) > 0 {
		err = self.AttachKeypair(instanceId, keypairName)
		if err != nil {
			return err
		}
	}

	params := make(map[string]string)

	// if resetPassword {
	//	params["Password"] = seclib2.RandomPassword2(12)
	// }
	// 指定密码的情况下，使用指定的密码
	if len(password) > 0 {
		params["Password"] = password
	}

	if len(name) > 0 && instance.InstanceName != name {
		params["InstanceName"] = name
		params["HostName"] = stringutils2.GenerateHostName(name, instance.OSType)
	}

	if len(description) > 0 && instance.Description != description {
		params["Description"] = description
	}

	if len(params) > 0 {
		return self.modifyInstanceAttribute(instanceId, params)
	} else {
		return nil
	}
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	for {
		err := self.host.zone.region.DeleteVM(self.InstanceId)
		if err != nil {
			if isError(err, "IncorrectInstanceStatus.Initializing") {
				log.Infof("The instance is initializing, try later ...")
				time.Sleep(10 * time.Second)
			} else {
				log.Errorf("DeleteVM fail: %s", err)
				return err
			}
		} else {
			break
		}
	}
	return cloudprovider.WaitDeleted(self, 10*time.Second, 300*time.Second) // 5minutes
}

func (self *SRegion) UpdateVM(instanceId string, name, osType string) error {
	/*
			api: ModifyInstanceAttribute
		    https://help.apsara.com/document_detail/25503.html?spm=a2c4g.11186623.4.1.DrgpjW
	*/
	params := make(map[string]string)
	params["HostName"] = stringutils2.GenerateHostName(name, osType)
	params["InstanceName"] = name
	return self.modifyInstanceAttribute(instanceId, params)
}

func (self *SRegion) modifyInstanceAttribute(instanceId string, params map[string]string) error {
	return self.instanceOperation(instanceId, "ModifyInstanceAttribute", params)
}

func (self *SRegion) ReplaceSystemDisk(instanceId string, imageId string, passwd string, keypairName string, sysDiskSizeGB int) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	params["ImageId"] = imageId
	if len(passwd) > 0 {
		params["Password"] = passwd
	} else {
		params["PasswordInherit"] = "True"
	}
	if len(keypairName) > 0 {
		params["KeyPairName"] = keypairName
	}
	if sysDiskSizeGB > 0 {
		params["SystemDisk.Size"] = fmt.Sprintf("%d", sysDiskSizeGB)
	}
	body, err := self.ecsRequest("ReplaceSystemDisk", params)
	if err != nil {
		return "", err
	}
	// log.Debugf("%s", body.String())
	return body.GetString("DiskId")
}

func (self *SRegion) ChangeVMConfig(zoneId string, instanceId string, ncpu int, vmem int, disks []*SDisk) error {
	// todo: support change disk config?
	params := make(map[string]string)
	instanceTypes, e := self.GetMatchInstanceTypes(ncpu, vmem, 0, zoneId)
	if e != nil {
		return e
	}

	for _, instancetype := range instanceTypes {
		params["InstanceType"] = instancetype.InstanceTypeId
		params["ClientToken"] = utils.GenRequestId(20)
		if err := self.instanceOperation(instanceId, "ModifyInstanceSpec", params); err != nil {
			log.Errorf("Failed for %s: %s", instancetype.InstanceTypeId, err)
		} else {
			return nil
		}
	}

	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (self *SRegion) ChangeVMConfig2(zoneId string, instanceId string, instanceType string, disks []*SDisk) error {
	// todo: support change disk config?
	params := make(map[string]string)
	params["InstanceType"] = instanceType
	params["ClientToken"] = utils.GenRequestId(20)
	if err := self.instanceOperation(instanceId, "ModifyInstanceSpec", params); err != nil {
		log.Errorf("Failed for %s: %s", instanceType, err)
		return fmt.Errorf("Failed to change vm config, specification not supported")
	} else {
		return nil
	}
}

func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["DiskId"] = diskId
	log.Infof("Detach instance %s disk %s", instanceId, diskId)
	_, err := self.ecsRequest("DetachDisk", params)
	if err != nil {
		if strings.Contains(err.Error(), "The specified disk has not been attached on the specified instance") {
			return nil
		}
		return errors.Wrap(err, "DetachDisk")
	}

	return nil
}

func (self *SRegion) AttachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["DiskId"] = diskId
	_, err := self.ecsRequest("AttachDisk", params)
	if err != nil {
		log.Errorf("AttachDisk %s to %s fail %s", diskId, instanceId, err)
		return err
	}

	return nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if len(self.EipAddress.IpAddress) > 0 {
		return self.host.zone.region.GetEip(self.EipAddress.AllocationId)
	}
	if len(self.PublicIpAddress.IpAddress) > 0 {
		eip := SEipAddress{}
		eip.region = self.host.zone.region
		eip.IpAddress = self.PublicIpAddress.IpAddress[0]
		eip.InstanceId = self.InstanceId
		eip.InstanceType = EIP_INSTANCE_TYPE_ECS
		eip.Status = EIP_STATUS_INUSE
		eip.AllocationId = self.InstanceId // fixed
		eip.AllocationTime = self.CreationTime
		eip.Bandwidth = self.InternetMaxBandwidthOut
		eip.InternetChargeType = self.InternetChargeType
		return &eip, nil
	}
	return nil, nil
}

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	return self.host.zone.region.AssignSecurityGroup(secgroupId, self.InstanceId)
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return self.host.zone.region.SetSecurityGroups(secgroupIds, self.InstanceId)
}

func (self *SInstance) GetBillingType() string {
	return convertChargeType(self.InstanceChargeType)
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.CreationTime
}

func (self *SInstance) GetExpiredAt() time.Time {
	return convertExpiredAt(self.ExpiredTime)
}

func (self *SInstance) UpdateUserData(userData string) error {
	return self.host.zone.region.updateInstance(self.InstanceId, "", "", "", "", userData)
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return self.host.zone.region.RenewInstance(self.InstanceId, bc)
}

func billingCycle2Params(bc *billing.SBillingCycle, params map[string]string) error {
	if bc.GetMonths() > 0 {
		params["PeriodUnit"] = "Month"
		params["Period"] = fmt.Sprintf("%d", bc.GetMonths())
	} else if bc.GetWeeks() > 0 {
		params["PeriodUnit"] = "Week"
		params["Period"] = fmt.Sprintf("%d", bc.GetWeeks())
	} else {
		return fmt.Errorf("invalid renew time period %s", bc.String())
	}
	return nil
}

func (region *SRegion) RenewInstance(instanceId string, bc billing.SBillingCycle) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	err := billingCycle2Params(&bc, params)
	if err != nil {
		return err
	}
	params["ClientToken"] = utils.GenRequestId(20)
	_, err = region.ecsRequest("RenewInstance", params)
	if err != nil {
		log.Errorf("RenewInstance fail %s", err)
		return err
	}
	return nil
}

func (self *SInstance) GetProjectId() string {
	return self.ResourceGroupId
}

func (self *SInstance) GetError() error {
	return nil
}

func (region *SRegion) ConvertPublicIpToEip(instanceId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["RegionId"] = region.RegionId
	_, err := region.ecsRequest("ConvertNatPublicIpToEip", params)
	return err
}

func (self *SInstance) ConvertPublicIpToEip() error {
	return self.host.zone.region.ConvertPublicIpToEip(self.InstanceId)
}

func (region *SRegion) SetInstanceAutoRenew(instanceId string, autoRenew bool) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["RegionId"] = region.RegionId
	if autoRenew {
		params["RenewalStatus"] = "AutoRenewal"
		params["Duration"] = "1"
	} else {
		params["RenewalStatus"] = "Normal"
	}
	_, err := region.ecsRequest("ModifyInstanceAutoRenewAttribute", params)
	return err
}

type SAutoRenewAttr struct {
	Duration         int
	AutoRenewEnabled bool
	RenewalStatus    string
	PeriodUnit       string
}

func (region *SRegion) GetInstanceAutoRenewAttribute(instanceId string) (*SAutoRenewAttr, error) {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["RegionId"] = region.RegionId
	resp, err := region.ecsRequest("DescribeInstanceAutoRenewAttribute", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeInstanceAutoRenewAttribute")
	}
	attr := []SAutoRenewAttr{}
	err = resp.Unmarshal(&attr, "InstanceRenewAttributes", "InstanceRenewAttribute")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	if len(attr) == 1 {
		return &attr[0], nil
	}
	return nil, fmt.Errorf("get %d auto renew info", len(attr))
}

func (self *SInstance) IsAutoRenew() bool {
	attr, err := self.host.zone.region.GetInstanceAutoRenewAttribute(self.InstanceId)
	if err != nil {
		log.Errorf("failed to get instance %s auto renew info", self.InstanceId)
		return false
	}
	return attr.AutoRenewEnabled
}

func (self *SInstance) SetAutoRenew(autoRenew bool) error {
	return self.host.zone.region.SetInstanceAutoRenew(self.InstanceId, autoRenew)
}

func (self *SInstance) SetMetadata(tags map[string]string, replace bool) error {
	return self.host.zone.region.SetResourceTags("ecs", "instance", []string{self.InstanceId}, tags, replace)
}
