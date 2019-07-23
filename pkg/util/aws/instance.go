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

package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/onecloud/pkg/multicloud"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/osprofile"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/cloudinit"
)

const (
	InstanceStatusPending    = "pending"
	InstanceStatusRunning    = "running"
	InstanceStatusShutting   = "shutting-down"
	InstanceStatusTerminated = "terminated"
	InstanceStatusStopping   = "stopping"
	InstanceStatusStopped    = "stopped"
)

type InstanceChargeType string

type SIpAddress struct {
	IpAddress []string
}

type SNetworkInterfaces struct {
	NetworkInterface []SNetworkInterface
}

type SNetworkInterface struct {
	MacAddress         string
	NetworkInterfaceId string
	PrimaryIpAddress   string // PrivateIpAddress
}

type SSecurityGroupIds struct {
	SecurityGroupId []string
}

type SVpcAttributes struct {
	PrivateIpAddress SIpAddress
	NetworkId        string // subnet id
	VpcId            string
}

type SInstance struct {
	multicloud.SInstanceBase

	host       *SHost
	RegionId   string
	ZoneId     string
	InstanceId string
	ImageId    string

	HostName          string
	InstanceName      string
	InstanceType      string
	Cpu               int
	Memory            int // MB
	IoOptimized       bool
	KeyPairName       string
	CreationTime      time.Time // LaunchTime
	ExpiredTime       time.Time
	ProductCodes      []string
	PublicDNSName     string
	InnerIpAddress    SIpAddress
	PublicIpAddress   SIpAddress
	RootDeviceName    string
	Status            string // state
	VlanId            string // subnet ID ?
	VpcAttributes     SVpcAttributes
	SecurityGroupIds  SSecurityGroupIds
	NetworkInterfaces SNetworkInterfaces
	EipAddress        SEipAddress
	Disks             []string
	DeviceNames       []string
	OSName            string
	OSType            string
	Description       string

	// 这些貌似都没啥用
	// AutoReleaseTime         string
	// DeviceAvailable         bool
	// GPUAmount               int
	// GPUSpec                 string
	// InstanceChargeType      InstanceChargeType
	// InstanceNetworkType     string
	// InstanceTypeFamily      string
	// InternetChargeType      string
	// InternetMaxBandwidthIn  int
	// InternetMaxBandwidthOut int
	// OperationLocks          SOperationLocks
	// Recyclable              bool
	// SerialNumber            string
	// SpotPriceLimit          string
	// SpotStrategy            string
	// StartTime               time.Time
	// StoppedMode             string
}

func (self *SInstance) UpdateUserData(userData string) error {
	udata := &ec2.BlobAttributeValue{}
	udata.SetValue([]byte(userData))

	input := &ec2.ModifyInstanceAttributeInput{}
	input.SetUserData(udata)
	input.SetInstanceId(self.GetId())
	_, err := self.host.zone.region.ec2Client.ModifyInstanceAttribute(input)
	if err != nil {
		return err
	}

	return nil
}

func (self *SInstance) GetUserData() (string, error) {
	input := &ec2.DescribeInstanceAttributeInput{}
	input.SetInstanceId(self.GetId())
	input.SetAttribute("userData")
	ret, err := self.host.zone.region.ec2Client.DescribeInstanceAttribute(input)
	if err != nil {
		return "", err
	}

	d := StrVal(ret.UserData.Value)
	udata, err := base64.StdEncoding.DecodeString(d)
	if err != nil {
		return "", fmt.Errorf("GetUserData decode user data %s", err)
	}

	return string(udata), nil
}

func (self *SInstance) GetId() string {
	return self.InstanceId
}

func (self *SInstance) GetName() string {
	if len(self.InstanceName) > 0 {
		return self.InstanceName
	}

	return self.GetId()
}

func (self *SInstance) GetGlobalId() string {
	return self.InstanceId
}

func (self *SInstance) GetStatus() string {
	switch self.Status {
	case InstanceStatusRunning:
		return api.VM_RUNNING
	case InstanceStatusPending: // todo: pending ?
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

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetInstanceType() string {
	return self.InstanceType
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return self.SecurityGroupIds.SecurityGroupId, nil
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	// todo: add price_key here
	// 格式 ：regionId::instanceType::osName::os_license::preInstall::tenancy::usageType
	// 举例 ： cn-northwest-1::c3.2xlarge::linux::NA::NA::shared::boxusage
	// 注意：除了空用大写NA.其他一律用小写格式
	priceKey := fmt.Sprintf("%s::%s::%s::NA::NA::shared::boxusage", self.RegionId, self.InstanceType, strings.ToLower(self.OSType))
	data.Add(jsonutils.NewString(priceKey), "price_key")
	tags, err := FetchTags(self.host.zone.region.ec2Client, self.InstanceId)
	if err != nil {
		log.Errorln(err)
	} else {
		data.Update(tags)
	}

	data.Add(jsonutils.NewString(self.host.zone.GetGlobalId()), "zone_ext_id")

	// no need to sync image metadata
	/*
		if len(self.ImageId) > 0 {
			image, err := self.host.zone.region.GetImage(self.ImageId)
			if err != nil {
				log.Errorf("Failed to find image %s for instance %s zone %s", self.ImageId, self.GetId(), self.ZoneId)
			} else {
				meta := image.GetMetadata()
				if meta != nil {
					data.Update(meta)
				}
			}
		}
	*/
	return data
}

func (self *SInstance) GetBillingType() string {
	// todo: implement me
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.CreationTime
}

func (self *SInstance) GetExpiredAt() time.Time {
	return self.ExpiredTime
}

func (self *SInstance) GetCreateTime() time.Time {
	return self.CreationTime
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, _, err := self.host.zone.region.GetDisks(self.InstanceId, "", "", nil, 0, 0)
	if err != nil {
		log.Errorf("fetchDisks fail %s", err)
		return nil, err
	}

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
	nics := make([]cloudprovider.ICloudNic, 0)
	for _, ip := range self.VpcAttributes.PrivateIpAddress.IpAddress {
		nic := SInstanceNic{instance: self, ipAddr: ip}
		nics = append(nics, &nic)
	}
	return nics, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if len(self.EipAddress.IpAddress) > 0 {
		return self.host.zone.region.GetEipByIpAddress(self.EipAddress.IpAddress)
	} else if len(self.PublicIpAddress.IpAddress) > 0 {
		eip := SEipAddress{}
		eip.region = self.host.zone.region
		eip.IpAddress = self.PublicIpAddress.IpAddress[0]
		eip.InstanceId = self.InstanceId
		eip.AllocationId = self.InstanceId // fixed. AllocationId等于InstanceId即表示为 仿真EIP。
		eip.Bandwidth = 10000
		return &eip, nil
	} else {
		return nil, nil
	}
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

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	return self.SetSecurityGroups([]string{secgroupId})
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	ids := []*string{}
	for i := 0; i < len(secgroupIds); i++ {
		ids = append(ids, &secgroupIds[i])
	}
	return self.host.zone.region.assignSecurityGroups(ids, self.InstanceId)
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_AWS
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

func (self *SInstance) StopVM(ctx context.Context, isForce bool) error {
	err := self.host.zone.region.StopVM(self.InstanceId, isForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	for {
		err := self.host.zone.region.DeleteVM(self.InstanceId)
		if err != nil && self.Status != InstanceStatusTerminated {
			return err
		} else {
			break
		}
	}

	params := &ec2.DescribeInstancesInput{InstanceIds: []*string{&self.InstanceId}}
	return self.host.zone.region.ec2Client.WaitUntilInstanceTerminated(params)
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return self.host.zone.region.UpdateVM(self.InstanceId, name)
}

func (self *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	udata, err := self.GetUserData()
	if err != nil {
		return "", err
	}

	var cloudconfig *cloudinit.SCloudConfig
	if len(udata) == 0 {
		cloudconfig = &cloudinit.SCloudConfig{}
	} else {
		cloudconfig, err = cloudinit.ParseUserDataBase64(udata)
		if err != nil {
			log.Debugf("RebuildRoot invalid instance user data %s", udata)
			return "", fmt.Errorf("RebuildRoot invalid instance user data %s", err)
		}
	}

	loginUser := cloudinit.NewUser(api.VM_AWS_DEFAULT_LOGIN_USER)
	loginUser.SudoPolicy(cloudinit.USER_SUDO_NOPASSWD)
	if len(publicKey) > 0 {
		loginUser.SshKey(publicKey)
		cloudconfig.MergeUser(loginUser)
	} else if len(passwd) > 0 {
		loginUser.Password(passwd)
		cloudconfig.MergeUser(loginUser)
	}

	diskId, err := self.host.zone.region.ReplaceSystemDisk(ctx, self.InstanceId, imageId, sysSizeGB, cloudconfig.UserDataBase64())
	if err != nil {
		return "", err
	}

	return diskId, nil
}

func (self *SInstance) DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return self.host.zone.region.DeployVM(self.InstanceId, name, password, publicKey, deleteKeypair, description)
}

func (self *SInstance) ChangeConfig(ctx context.Context, ncpu int, vmem int) error {
	return self.host.zone.region.ChangeVMConfig(self.ZoneId, self.InstanceId, ncpu, vmem, nil)
}

func (self *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	return self.host.zone.region.ChangeVMConfig2(self.ZoneId, self.InstanceId, instanceType, nil)
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	name, err := NextDeviceName(self.DeviceNames)
	if err != nil {
		return err
	}

	err = self.host.zone.region.AttachDisk(self.InstanceId, diskId, name)
	if err != nil {
		return err
	}

	self.DeviceNames = append(self.DeviceNames, name)
	return nil
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.DetachDisk(self.InstanceId, diskId)
}

func (self *SInstance) getVpc() (*SVpc, error) {
	return self.host.zone.region.getVpc(self.VpcAttributes.VpcId)
}

func (self *SRegion) GetInstances(zoneId string, ids []string, offset int, limit int) ([]SInstance, int, error) {
	params := &ec2.DescribeInstancesInput{}
	filters := make([]*ec2.Filter, 0)
	if len(zoneId) > 0 {
		filters = AppendSingleValueFilter(filters, "availability-zone", zoneId)
	}

	if len(ids) > 0 {
		params = params.SetInstanceIds(ConvertedList(ids))
	}

	if len(filters) > 0 {
		params = params.SetFilters(filters)
	}

	res, err := self.ec2Client.DescribeInstances(params)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
			return nil, 0, cloudprovider.ErrNotFound
		} else {
			return nil, 0, err
		}
	}

	instances := []SInstance{}
	for _, reservation := range res.Reservations {
		for _, instance := range reservation.Instances {
			if err := FillZero(instance); err != nil {
				return nil, 0, err
			}

			// 不同步已经terminated的主机
			if *instance.State.Name == ec2.InstanceStateNameTerminated {
				continue
			}

			tagspec := TagSpec{}
			tagspec.LoadingEc2Tags(instance.Tags)

			disks := []string{}
			devicenames := []string{}
			for _, d := range instance.BlockDeviceMappings {
				if d.Ebs != nil && d.Ebs.VolumeId != nil {
					disks = append(disks, *d.Ebs.VolumeId)
					devicenames = append(devicenames, *d.DeviceName)
				}
			}

			var secgroups SSecurityGroupIds
			for _, s := range instance.SecurityGroups {
				if s.GroupId != nil {
					secgroups.SecurityGroupId = append(secgroups.SecurityGroupId, *s.GroupId)
				}
			}

			var networkInterfaces SNetworkInterfaces
			eipAddress := SEipAddress{}
			for _, n := range instance.NetworkInterfaces {
				i := SNetworkInterface{
					MacAddress:         *n.MacAddress,
					NetworkInterfaceId: *n.NetworkInterfaceId,
					PrimaryIpAddress:   *n.PrivateIpAddress,
				}
				networkInterfaces.NetworkInterface = append(networkInterfaces.NetworkInterface, i)

				// todo: 可能有多个EIP的情况。目前只支持一个EIP
				if n.Association != nil && StrVal(n.Association.IpOwnerId) != "amazon" {
					if eipAddress.IpAddress == "" && len(StrVal(n.Association.PublicIp)) > 0 {
						eipAddress.IpAddress = *n.Association.PublicIp
					}
				}
			}

			var vpcattr SVpcAttributes
			vpcattr.VpcId = *instance.VpcId
			vpcattr.PrivateIpAddress = SIpAddress{[]string{*instance.PrivateIpAddress}}
			vpcattr.NetworkId = *instance.SubnetId

			var productCodes []string
			for _, p := range instance.ProductCodes {
				productCodes = append(productCodes, *p.ProductCodeId)
			}

			publicIpAddress := SIpAddress{}
			if len(*instance.PublicIpAddress) > 0 {
				publicIpAddress.IpAddress = []string{*instance.PublicIpAddress}
			}

			innerIpAddress := SIpAddress{}
			if len(*instance.PrivateIpAddress) > 0 {
				innerIpAddress.IpAddress = []string{*instance.PrivateIpAddress}
			}

			szone, err := self.getZoneById(*instance.Placement.AvailabilityZone)
			if err != nil {
				log.Errorf("getZoneById %s fail %s", *instance.Placement.AvailabilityZone, err)
				return nil, 0, err
			}

			osType := "Linux"
			if instance.Platform != nil && len(*instance.Platform) > 0 {
				osType = *instance.Platform
			}

			host := szone.getHost()

			sinstance := SInstance{
				RegionId:          self.RegionId,
				host:              host,
				ZoneId:            *instance.Placement.AvailabilityZone,
				InstanceId:        *instance.InstanceId,
				ImageId:           *instance.ImageId,
				InstanceType:      *instance.InstanceType,
				Cpu:               int(*instance.CpuOptions.CoreCount),
				IoOptimized:       *instance.EbsOptimized,
				KeyPairName:       *instance.KeyName,
				CreationTime:      *instance.LaunchTime,
				PublicDNSName:     *instance.PublicDnsName,
				RootDeviceName:    *instance.RootDeviceName,
				Status:            *instance.State.Name,
				InnerIpAddress:    innerIpAddress,
				PublicIpAddress:   publicIpAddress,
				EipAddress:        eipAddress,
				InstanceName:      tagspec.GetNameTag(),
				Description:       tagspec.GetDescTag(),
				Disks:             disks,
				DeviceNames:       devicenames,
				SecurityGroupIds:  secgroups,
				NetworkInterfaces: networkInterfaces,
				VpcAttributes:     vpcattr,
				ProductCodes:      productCodes,
				OSName:            osType, // todo: 这里在model层回写OSName信息
				OSType:            osType,
				// ExpiredTime:
				// VlanId:
				// OSType:
			}

			instances = append(instances, sinstance)
		}
	}

	return instances, len(instances), nil
}

func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	if len(instanceId) == 0 {
		return nil, fmt.Errorf("GetInstance instanceId should not be empty.")
	}

	instances, _, err := self.GetInstances("", []string{instanceId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &instances[0], nil
}

func (self *SRegion) GetInstanceIdByImageId(imageId string) (string, error) {
	params := &ec2.DescribeInstancesInput{}
	filters := []*ec2.Filter{}
	filters = AppendSingleValueFilter(filters, "image-id", imageId)
	params.SetFilters(filters)
	ret, err := self.ec2Client.DescribeInstances(params)
	if err != nil {
		return "", err
	}

	for _, item := range ret.Reservations {
		for _, instance := range item.Instances {
			return *instance.InstanceId, nil
		}
	}
	return "", fmt.Errorf("instance launch with image %s not found", imageId)
}

func (self *SRegion) CreateInstance(name string, imageId string, instanceType string, SubnetId string, securityGroupId string,
	zoneId string, desc string, disks []SDisk, ipAddr string,
	keypair string, userData string) (string, error) {
	var count int64 = 1
	// disk
	blockDevices := []*ec2.BlockDeviceMapping{}
	for i, disk := range disks {
		var ebs ec2.EbsBlockDevice
		var deviceName string

		if i == 0 {
			var size int64
			deleteOnTermination := true
			size = int64(disk.Size)
			ebs = ec2.EbsBlockDevice{
				DeleteOnTermination: &deleteOnTermination,
				// The st1 volume type cannot be used for boot volumes. Please use a supported boot volume type: standard,io1,gp2.
				// the encrypted flag cannot be specified since device /dev/sda1 has a snapshot specified.
				// Encrypted:           &disk.Encrypted,
				VolumeSize: &size,
				VolumeType: &disk.Category,
			}

			deviceName = fmt.Sprintf("/dev/sda1")
		} else {
			var size int64
			size = int64(disk.Size)
			ebs = ec2.EbsBlockDevice{
				DeleteOnTermination: &disk.DeleteWithInstance,
				Encrypted:           &disk.Encrypted,
				VolumeSize:          &size,
				VolumeType:          &disk.Category,
			}
			// todo: generator device name
			// todo: 这里还需要测试预置硬盘的实例。deviceName是否会冲突。
			deviceName = fmt.Sprintf("/dev/sd%s", string(98+i))
		}

		// io1类型的卷需要指定IOPS参数。这里根据aws网站的建议值进行设置
		// 卷每增加1G。IOPS增加50。最大不超过32000
		if disk.Category == api.STORAGE_IO1_SSD {
			iops := int64(disk.Size * 50)
			if iops < 32000 {
				ebs.SetIops(iops)
			} else {
				ebs.SetIops(32000)
			}
		}

		blockDevice := &ec2.BlockDeviceMapping{
			DeviceName: &deviceName,
			Ebs:        &ebs,
		}

		blockDevices = append(blockDevices, blockDevice)
	}

	// tags
	tags := TagSpec{ResourceType: "instance"}
	tags.SetNameTag(name)
	tags.SetDescTag(desc)
	ec2TagSpec, err := tags.GetTagSpecifications()
	if err != nil {
		return "", err
	}

	params := ec2.RunInstancesInput{
		ImageId:             &imageId,
		InstanceType:        &instanceType,
		MaxCount:            &count,
		MinCount:            &count,
		BlockDeviceMappings: blockDevices,
		Placement:           &ec2.Placement{AvailabilityZone: &zoneId},
		TagSpecifications:   []*ec2.TagSpecification{ec2TagSpec},
	}

	// keypair
	if len(keypair) > 0 {
		params.SetKeyName(keypair)
	}

	// user data
	if len(userData) > 0 {
		params.SetUserData(userData)
	}

	// ip address
	if len(ipAddr) > 0 {
		params.SetPrivateIpAddress(ipAddr)
	}

	// subnet id
	if len(SubnetId) > 0 {
		params.SetSubnetId(SubnetId)
	}

	// security group
	if len(securityGroupId) > 0 {
		params.SetSecurityGroupIds([]*string{&securityGroupId})
	}

	res, err := self.ec2Client.RunInstances(&params)
	if err != nil {
		log.Errorf("CreateInstance fail %s", err)
		return "", err
	}

	if len(res.Instances) == 1 {
		return *res.Instances[0].InstanceId, nil
	} else {
		msg := fmt.Sprintf("CreateInstance fail: %d instance created. ", len(res.Instances))
		log.Errorf(msg)
		return "", fmt.Errorf(msg)
	}
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.Status, nil
}

func (self *SRegion) instanceStatusChecking(instanceId, status string) error {
	remoteStatus, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status: %s", err)
		return err
	}
	if status != remoteStatus {
		log.Errorf("instanceStatusChecking: vm status is %s expect %s", remoteStatus, status)
		return cloudprovider.ErrInvalidStatus
	}

	return nil
}

func (self *SRegion) StartVM(instanceId string) error {
	if err := self.instanceStatusChecking(instanceId, InstanceStatusStopped); err != nil {
		return err
	}

	params := &ec2.StartInstancesInput{}
	params.SetInstanceIds([]*string{&instanceId})
	_, err := self.ec2Client.StartInstances(params)
	return err
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	if err := self.instanceStatusChecking(instanceId, InstanceStatusRunning); err != nil {
		return err
	}

	params := &ec2.StopInstancesInput{}
	params.SetInstanceIds([]*string{&instanceId})
	_, err := self.ec2Client.StopInstances(params)
	return err
}

func (self *SRegion) DeleteVM(instanceId string) error {
	if err := self.instanceStatusChecking(instanceId, InstanceStatusStopped); err != nil {
		return err
	}

	// 检查删除保护状态.如果已开启则先关闭删除保护再进行删除操作
	protect, err := self.deleteProtectStatusVM(instanceId)
	if err != nil {
		return err
	}

	if protect {
		log.Warningf("DeleteVM instance %s which termination protect is in open status", instanceId)
		err = self.deleteProtectVM(instanceId, false)
		if err != nil {
			return err
		}
	}

	params := &ec2.TerminateInstancesInput{}
	params.SetInstanceIds([]*string{&instanceId})
	_, err = self.ec2Client.TerminateInstances(params)
	return err
}

func (self *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	params := &ec2.CreateTagsInput{}
	params.SetResources([]*string{&instanceId})
	tagspec := TagSpec{ResourceType: "instance"}

	if len(keypairName) > 0 {
		return fmt.Errorf("aws not support reset publickey")
	}

	if len(password) > 0 {
		return fmt.Errorf("aws not support set password, use publickey instead")
	}

	if deleteKeypair {
		return fmt.Errorf("aws not support delete publickey")
	}

	if len(name) > 0 {
		tagspec.SetNameTag(name)
	}

	if len(description) > 0 {
		tagspec.SetDescTag(description)
	}

	ec2Tag, _ := tagspec.GetTagSpecifications()
	if len(ec2Tag.Tags) > 0 {
		params.SetTags(ec2Tag.Tags)
		_, err := self.ec2Client.CreateTags(params)
		if err != nil {
			return err
		}
	} else {
		log.Debugf("no changes")
	}

	return nil
}

func (self *SRegion) UpdateVM(instanceId string, hostname string) error {
	// https://docs.aws.amazon.com/zh_cn/AWSEC2/latest/UserGuide/set-hostname.html
	return fmt.Errorf("aws not support change hostname.")
}

func (self *SRegion) ReplaceSystemDisk(ctx context.Context, instanceId string, imageId string, sysDiskSizeGB int, userdata string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	disks, _, err := self.GetDisks(instanceId, instance.ZoneId, "", nil, 0, 0)
	if err != nil {
		return "", err
	}

	var rootDisk *SDisk
	for _, disk := range disks {
		if disk.Type == api.DISK_TYPE_SYS {
			rootDisk = &disk
			break
		}
	}

	if rootDisk == nil {
		return "", fmt.Errorf("can not find root disk of instance %s", instanceId)
	}
	log.Debugf("ReplaceSystemDisk replace root disk %s", rootDisk.DiskId)

	// create tmp server
	tempName := fmt.Sprintf("__tmp_%s", instance.GetName())
	_id, err := self.CreateInstance(tempName,
		imageId,
		instance.InstanceType,
		"",
		"",
		instance.ZoneId,
		instance.Description,
		[]SDisk{{Size: sysDiskSizeGB, Category: rootDisk.Category}},
		"",
		"",
		userdata)
	if err == nil {
		defer self.DeleteVM(_id)
	} else {
		log.Debugf("ReplaceSystemDisk create temp server failed. %s", err)
		return "", fmt.Errorf("ReplaceSystemDisk create temp server failed.")
	}

	self.ec2Client.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{InstanceIds: []*string{&_id}})
	err = self.StopVM(_id, true)
	if err != nil {
		log.Debugf("ReplaceSystemDisk stop temp server failed %s", err)
		return "", fmt.Errorf("ReplaceSystemDisk stop temp server failed")
	}
	self.ec2Client.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{InstanceIds: []*string{&_id}})

	// detach disks
	tempInstance, err := self.GetInstance(_id)
	if err != nil {
		log.Debugf("ReplaceSystemDisk get temp server failed %s", err)
		return "", fmt.Errorf("ReplaceSystemDisk get temp server failed")
	}

	err = self.DetachDisk(instance.GetId(), rootDisk.DiskId)
	if err != nil {
		log.Debugf("ReplaceSystemDisk detach disk %s: %s", rootDisk.DiskId, err)
		return "", err
	}

	err = self.DetachDisk(tempInstance.GetId(), tempInstance.Disks[0])
	if err != nil {
		log.Debugf("ReplaceSystemDisk detach disk %s: %s", tempInstance.Disks[0], err)
		return "", err
	}
	self.ec2Client.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{VolumeIds: []*string{&rootDisk.DiskId}})
	self.ec2Client.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{VolumeIds: []*string{&tempInstance.Disks[0]}})

	err = self.AttachDisk(instance.GetId(), tempInstance.Disks[0], rootDisk.Device)
	if err != nil {
		log.Debugf("ReplaceSystemDisk attach disk %s: %s", tempInstance.Disks[0], err)
		return "", err
	}
	self.ec2Client.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{InstanceIds: []*string{&instanceId}})
	self.ec2Client.WaitUntilVolumeInUse(&ec2.DescribeVolumesInput{VolumeIds: []*string{&tempInstance.Disks[0]}})

	err = instance.UpdateUserData(userdata)
	if err != nil {
		log.Debugf("ReplaceSystemDisk update user data %s", err)
		return "", fmt.Errorf("ReplaceSystemDisk update user data failed")
	}

	err = self.DeleteDisk(rootDisk.DiskId)
	if err != nil {
		log.Debugf("ReplaceSystemDisk delete old disk %s: %s", rootDisk.DiskId, err)
	}
	return tempInstance.Disks[0], nil
}

func (self *SRegion) ChangeVMConfig(zoneId string, instanceId string, ncpu int, vmem int, disks []*SDisk) error {
	params := &ec2.ModifyInstanceAttributeInput{}
	params.SetInstanceId(instanceId)
	instanceTypes, err := self.GetMatchInstanceTypes(ncpu, vmem, 0, zoneId)
	if err != nil {
		return err
	}

	for _, instancetype := range instanceTypes {
		t := &ec2.AttributeValue{Value: &instancetype.InstanceTypeId}
		params.SetInstanceType(t)

		_, err := self.ec2Client.ModifyInstanceAttribute(params)
		if err != nil {
			log.Errorf("Failed for %s: %s", instancetype.InstanceTypeId, err)
		} else {
			return nil
		}
	}

	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (self *SRegion) ChangeVMConfig2(zoneId string, instanceId string, instanceType string, disks []*SDisk) error {
	params := &ec2.ModifyInstanceAttributeInput{}
	params.SetInstanceId(instanceId)

	t := &ec2.AttributeValue{Value: &instanceType}
	params.SetInstanceType(t)

	_, err := self.ec2Client.ModifyInstanceAttribute(params)
	if err != nil {
		return fmt.Errorf("Failed to change vm config, specification not supported. %s", err.Error())
	} else {
		return nil
	}
}

func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	params := &ec2.DetachVolumeInput{}
	params.SetInstanceId(instanceId)
	params.SetVolumeId(diskId)
	log.Debugf("DetachDisk %s", params.String())
	_, err := self.ec2Client.DetachVolume(params)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("'%s'is in the 'available' state", diskId)) {
			return nil
		}
		//InvalidVolume.NotFound: The volume 'vol-0a9eeda0a70a8d7fe' does not exist
		if strings.Contains(err.Error(), "InvalidVolume.NotFound") {
			return nil
		}
		return errors.Wrap(err, "ec2Client.DetachVolume")
	}
	return nil
}

func (self *SRegion) AttachDisk(instanceId string, diskId string, deviceName string) error {
	params := &ec2.AttachVolumeInput{}
	params.SetInstanceId(instanceId)
	params.SetVolumeId(diskId)
	params.SetDevice(deviceName)
	log.Debugf("AttachDisk %s", params.String())
	_, err := self.ec2Client.AttachVolume(params)
	return err
}

func (self *SRegion) deleteProtectStatusVM(instanceId string) (bool, error) {
	p := &ec2.DescribeInstanceAttributeInput{}
	p.SetInstanceId(instanceId)
	p.SetAttribute("disableApiTermination")
	ret, err := self.ec2Client.DescribeInstanceAttribute(p)
	if err != nil {
		return false, err
	}

	return *ret.DisableApiTermination.Value, nil
}

func (self *SRegion) deleteProtectVM(instanceId string, disableDelete bool) error {
	p2 := &ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{Value: &disableDelete},
		InstanceId:            &instanceId,
	}
	_, err := self.ec2Client.ModifyInstanceAttribute(p2)
	return err
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetError() error {
	return nil
}
