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

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/cloudinit"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/apis"
	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	img        *SImage
	RegionId   string
	ZoneId     string
	InstanceId string
	ImageId    string

	HostName                string
	InstanceName            string
	InstanceType            string
	Cpu                     int
	Memory                  int // MB
	IoOptimized             bool
	KeyPairName             string
	CreationTime            time.Time // LaunchTime
	ExpiredTime             time.Time
	ProductCodes            []string
	PublicDNSName           string
	InnerIpAddress          SIpAddress
	PublicIpAddress         SIpAddress
	RootDeviceName          string
	Status                  string // state
	VlanId                  string // subnet ID ?
	VpcAttributes           SVpcAttributes
	SecurityGroupIds        SSecurityGroupIds
	NetworkInterfaces       []SNetworkInterface
	EipAddress              SEipAddress
	Disks                   []string
	DeviceNames             []string
	OSName                  string
	OSType                  string
	Description             string
	InternetMaxBandwidthOut int
	Throughput              int
	OsArch                  *string

	TagSpec TagSpec

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
	ec2Client, err := self.host.zone.region.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.ModifyInstanceAttribute(input)
	if err != nil {
		return errors.Wrap(err, "ModifyInstanceAttribute")
	}

	return nil
}

func (self *SInstance) GetUserData() (string, error) {
	input := &ec2.DescribeInstanceAttributeInput{}
	input.SetInstanceId(self.GetId())
	input.SetAttribute("userData")
	ec2Client, err := self.host.zone.region.getEc2Client()
	if err != nil {
		return "", errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.DescribeInstanceAttribute(input)
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

func (self *SInstance) GetHostname() string {
	return self.GetName()
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

func (self *SInstance) GetTags() (map[string]string, error) {
	return self.TagSpec.GetTags()
}

func (self *SInstance) GetSysTags() map[string]string {
	return map[string]string{}
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

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetThroughput() int {
	return self.Throughput
}

func (self *SInstance) GetInternetMaxBandwidthOut() int {
	return self.InternetMaxBandwidthOut
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, _, err := self.host.zone.region.GetDisks(self.InstanceId, "", "", nil, 0, 0)
	if err != nil {
		log.Errorf("fetchDisks fail %s", err)
		return nil, errors.Wrap(err, "GetDisks")
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		store, err := self.host.zone.getStorageByCategory(disks[i].Category)
		if err != nil {
			return nil, errors.Wrap(err, "getStorageByCategory")
		}
		disks[i].storage = store
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	var (
		networkInterfaces = self.NetworkInterfaces
		nics              = make([]cloudprovider.ICloudNic, 0)
	)
	for _, networkInterface := range networkInterfaces {
		nic := SInstanceNic{
			instance: self,
			id:       networkInterface.NetworkInterfaceId,
			ipAddr:   networkInterface.PrivateIpAddress,
			macAddr:  networkInterface.MacAddress,
		}
		nics = append(nics, &nic)
	}
	return nics, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if len(self.EipAddress.PublicIp) > 0 {
		return self.host.zone.region.GetEipByIpAddress(self.EipAddress.PublicIp)
	}
	if len(self.PublicIpAddress.IpAddress) > 0 {
		eip := SEipAddress{region: self.host.zone.region}
		eip.region = self.host.zone.region
		eip.PublicIp = self.PublicIpAddress.IpAddress[0]
		eip.InstanceId = self.InstanceId
		eip.AllocationId = self.InstanceId // fixed. AllocationId等于InstanceId即表示为 仿真EIP。
		return &eip, nil
	}
	return nil, nil
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

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(osprofile.NormalizeOSType(self.OSType))
}

func (self *SInstance) GetFullOsName() string {
	return self.OSName
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	img, err := self.GetImage()
	if err != nil {
		log.Errorf("GetImage fail %s", err)
		return cloudprovider.BIOS
	}
	return img.GetBios()
}

func (self *SInstance) GetOsArch() string {
	if self.OsArch != nil && len(*self.OsArch) > 0 {
		switch *self.OsArch {
		case ec2.ArchitectureValuesArm64:
			return apis.OS_ARCH_AARCH64
		case ec2.ArchitectureValuesI386:
			return apis.OS_ARCH_X86
		case ec2.ArchitectureValuesX8664:
			return apis.OS_ARCH_X86_64
		default:
			return apis.OS_ARCH_X86_64
		}
	}
	img, err := self.GetImage()
	if err != nil {
		log.Errorf("GetImage fail %s", err)
		return apis.OS_ARCH_X86_64
	}
	return img.GetOsArch()
}

func (self *SInstance) GetOsDist() string {
	img, err := self.GetImage()
	if err != nil {
		log.Errorf("GetImage fail %s", err)
		return ""
	}
	return img.GetOsDist()
}

func (self *SInstance) GetOsVersion() string {
	img, err := self.GetImage()
	if err != nil {
		log.Errorf("GetImage fail %s", err)
		return ""
	}
	return img.GetOsVersion()
}

func (self *SInstance) GetOsLang() string {
	img, err := self.GetImage()
	if err != nil {
		log.Errorf("GetImage fail %s", err)
		return ""
	}
	return img.GetOsLang()
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

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := self.host.zone.region.StopVM(self.InstanceId, opts.IsForce)
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
	ec2Client, err := self.host.zone.region.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	return ec2Client.WaitUntilInstanceTerminated(params)
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	addTags := map[string]string{}
	addTags["Name"] = name
	Arn := self.GetArn()
	err := self.host.zone.region.TagResources([]string{Arn}, addTags)
	if err != nil {
		return errors.Wrapf(err, "self.host.zone.region.TagResources([]string{%s}, %s)", Arn, jsonutils.Marshal(addTags).String())
	}
	return nil
}

func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	udata, err := self.GetUserData()
	if err != nil {
		return "", err
	}

	// compare sysSizeGB
	image, err := self.host.zone.region.GetImage(desc.ImageId)
	if err != nil {
		return "", err
	} else {
		minSizeGB := image.GetMinOsDiskSizeGb()
		if minSizeGB > desc.SysSizeGB {
			desc.SysSizeGB = minSizeGB
		}
	}

	// upload keypair
	keypairName := self.KeyPairName
	if len(desc.PublicKey) > 0 {
		keypairName, err = self.host.zone.region.syncKeypair(desc.PublicKey)
		if err != nil {
			return "", fmt.Errorf("RebuildRoot.syncKeypair %s", err)
		}
	}

	userdata := ""
	srcOsType := strings.ToLower(string(self.GetOsType()))
	destOsType := strings.ToLower(string(image.GetOsType()))
	winOS := strings.ToLower(osprofile.OS_TYPE_WINDOWS)

	cloudconfig := &cloudinit.SCloudConfig{}
	if srcOsType != winOS && len(udata) > 0 {
		_cloudconfig, err := cloudinit.ParseUserDataBase64(udata)
		if err != nil {
			// 忽略无效的用户数据
			log.Debugf("RebuildRoot invalid instance user data %s", udata)
		} else {
			cloudconfig = _cloudconfig
		}
	}

	if (srcOsType != winOS && destOsType != winOS) || (srcOsType == winOS && destOsType != winOS) {
		// linux/windows to linux
		loginUser := cloudinit.NewUser(api.VM_AWS_DEFAULT_LOGIN_USER)
		loginUser.SudoPolicy(cloudinit.USER_SUDO_NOPASSWD)
		if len(desc.PublicKey) > 0 {
			loginUser.SshKey(desc.PublicKey)
			cloudconfig.MergeUser(loginUser)
		} else if len(desc.Password) > 0 {
			cloudconfig.SshPwauth = cloudinit.SSH_PASSWORD_AUTH_ON
			loginUser.Password(desc.Password)
			cloudconfig.MergeUser(loginUser)
		}

		userdata = cloudconfig.UserDataBase64()
	} else {
		// linux/windows to windows
		data := ""
		if len(desc.Password) > 0 {
			cloudconfig.SshPwauth = cloudinit.SSH_PASSWORD_AUTH_ON
			loginUser := cloudinit.NewUser(api.VM_AWS_DEFAULT_WINDOWS_LOGIN_USER)
			loginUser.SudoPolicy(cloudinit.USER_SUDO_NOPASSWD)
			loginUser.Password(desc.Password)
			cloudconfig.MergeUser(loginUser)
			data = fmt.Sprintf("<powershell>%s</powershell>", cloudconfig.UserDataPowerShell())
		} else {
			if len(udata) > 0 {
				data = fmt.Sprintf("<powershell>%s</powershell>", udata)
			}
		}

		userdata = base64.StdEncoding.EncodeToString([]byte(data))
	}

	diskId, err := self.host.zone.region.ReplaceSystemDisk(ctx, self.InstanceId, image, desc.SysSizeGB, keypairName, userdata)
	if err != nil {
		return "", err
	}

	return diskId, nil
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return self.host.zone.region.DeployVM(self.InstanceId, name, password, publicKey, deleteKeypair, description)
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if len(config.InstanceType) > 0 {
		return self.ChangeConfig2(ctx, config.InstanceType)
	}
	return errors.Wrap(errors.ErrClient, "Instance.ChangeConfig.InstanceTypeIsEmpty")
}

func (self *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	return self.host.zone.region.ChangeVMConfig2(self.ZoneId, self.InstanceId, instanceType, nil)
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) GetImage() (*SImage, error) {
	if self.img != nil {
		return self.img, nil
	}

	img, err := self.host.zone.region.GetImage(self.ImageId)
	if err != nil {
		return nil, errors.Wrap(err, "GetImage")
	}

	self.img = img
	return self.img, nil
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	img, err := self.GetImage()
	if err != nil {
		return errors.Wrap(err, "GetImage")
	}

	// mix in image block device names
	for i := range img.BlockDevicesNames {
		if !utils.IsInStringArray(img.BlockDevicesNames[i], self.DeviceNames) {
			self.DeviceNames = append(self.DeviceNames, img.BlockDevicesNames[i])
		}
	}

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

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, 0, errors.Wrap(err, "getEc2Client")
	}
	res, err := ec2Client.DescribeInstances(params)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
			return nil, 0, errors.Wrap(cloudprovider.ErrNotFound, "DescribeInstances")
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

			networkInterfaces := []SNetworkInterface{}
			eipAddress := SEipAddress{}
			for _, n := range instance.NetworkInterfaces {
				i := SNetworkInterface{
					MacAddress:         *n.MacAddress,
					NetworkInterfaceId: *n.NetworkInterfaceId,
					PrivateIpAddress:   *n.PrivateIpAddress,
				}
				networkInterfaces = append(networkInterfaces, i)

				// todo: 可能有多个EIP的情况。目前只支持一个EIP
				if n.Association != nil && StrVal(n.Association.IpOwnerId) != "amazon" {
					if eipAddress.PublicIp == "" && len(StrVal(n.Association.PublicIp)) > 0 {
						eipAddress.PublicIp = *n.Association.PublicIp
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
			vcpu := int(*instance.CpuOptions.CoreCount) * int(*instance.CpuOptions.ThreadsPerCore)
			sinstance := SInstance{
				RegionId:          self.RegionId,
				host:              host,
				ZoneId:            *instance.Placement.AvailabilityZone,
				InstanceId:        *instance.InstanceId,
				ImageId:           *instance.ImageId,
				InstanceType:      *instance.InstanceType,
				Cpu:               vcpu,
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
				OsArch:            instance.Architecture,

				TagSpec: tagspec,
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
		log.Errorf("GetInstances %s: %s", instanceId, err)
		return nil, errors.Wrap(err, "GetInstances")
	}
	if len(instances) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetInstances")
	}
	return &instances[0], nil
}

func (self *SRegion) GetInstanceIdByImageId(imageId string) (string, error) {
	params := &ec2.DescribeInstancesInput{}
	filters := []*ec2.Filter{}
	filters = AppendSingleValueFilter(filters, "image-id", imageId)
	params.SetFilters(filters)

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return "", errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.DescribeInstances(params)
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

func (self *SRegion) CreateInstance(name string, image *SImage, instanceType string, SubnetId string, securityGroupId string,
	zoneId string, desc string, disks []SDisk, ipAddr string,
	keypair string, userData string, ntags map[string]string, enableMonitorAgent bool,
) (string, error) {
	var count int64 = 1
	// disk
	blockDevices := []*ec2.BlockDeviceMapping{}
	for i := range disks {
		var ebs ec2.EbsBlockDevice
		var deviceName string
		var err error
		disk := disks[i]

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

			if len(image.RootDeviceName) > 0 {
				deviceName = image.RootDeviceName
			} else {
				deviceName = fmt.Sprintf("/dev/sda1")
			}
		} else {
			var size int64
			size = int64(disk.Size)
			ebs = ec2.EbsBlockDevice{
				DeleteOnTermination: &disk.DeleteWithInstance,
				Encrypted:           &disk.Encrypted,
				VolumeSize:          &size,
				VolumeType:          &disk.Category,
			}

			deviceName, err = NextDeviceName(image.BlockDevicesNames)
			if err != nil {
				return "", errors.Wrap(err, "NextDeviceName")
			}
		}

		if iops := GenDiskIops(disk.Category, disk.Size); iops > 0 {
			ebs.SetIops(iops)
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

	if len(ntags) > 0 {
		for k, v := range ntags {
			tags.SetTag(k, v)
		}
	}

	ec2TagSpec, err := tags.GetTagSpecifications()
	if err != nil {
		return "", err
	}

	imgId := image.GetId()
	params := ec2.RunInstancesInput{
		ImageId:             &imgId,
		InstanceType:        &instanceType,
		MaxCount:            &count,
		MinCount:            &count,
		BlockDeviceMappings: blockDevices,
		Placement:           &ec2.Placement{AvailabilityZone: &zoneId},
		TagSpecifications:   []*ec2.TagSpecification{ec2TagSpec},
	}
	params.Monitoring = &ec2.RunInstancesMonitoringEnabled{
		Enabled: &enableMonitorAgent,
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

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return "", errors.Wrap(err, "getEc2Client")
	}
	res, err := ec2Client.RunInstances(&params)
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

func (self *SRegion) StartVM(instanceId string) error {
	params := &ec2.StartInstancesInput{}
	params.SetInstanceIds([]*string{&instanceId})
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.StartInstances(params)
	return errors.Wrap(err, "StartInstances")
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	params := &ec2.StopInstancesInput{}
	params.SetInstanceIds([]*string{&instanceId})
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.StopInstances(params)
	return errors.Wrap(err, "StopInstances")
}

func (self *SRegion) DeleteVM(instanceId string) error {
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
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.TerminateInstances(params)
	return errors.Wrap(err, "TerminateInstances")
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

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	ec2Tag, _ := tagspec.GetTagSpecifications()
	if len(ec2Tag.Tags) > 0 {
		params.SetTags(ec2Tag.Tags)
		_, err := ec2Client.CreateTags(params)
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
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) ReplaceSystemDisk(ctx context.Context, instanceId string, image *SImage, sysDiskSizeGB int, keypair string, userdata string) (string, error) {
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

	subnetId := ""
	if len(instance.VpcAttributes.NetworkId) > 0 {
		subnetId = instance.VpcAttributes.NetworkId
	}

	// create tmp server
	tempName := fmt.Sprintf("__tmp_%s", instance.GetName())
	_id, err := self.CreateInstance(tempName,
		image,
		instance.InstanceType,
		subnetId,
		"",
		instance.ZoneId,
		instance.Description,
		[]SDisk{{Size: sysDiskSizeGB, Category: rootDisk.Category}},
		"",
		keypair,
		userdata,
		nil,
		false,
	)
	if err == nil {
		defer self.DeleteVM(_id)
	} else {
		log.Debugf("ReplaceSystemDisk create temp server failed. %s", err)
		return "", fmt.Errorf("ReplaceSystemDisk create temp server failed.")
	}

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return "", errors.Wrap(err, "getEc2Client")
	}

	ec2Client.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{InstanceIds: []*string{&_id}})
	err = self.StopVM(_id, true)
	if err != nil {
		log.Debugf("ReplaceSystemDisk stop temp server failed %s", err)
		return "", fmt.Errorf("ReplaceSystemDisk stop temp server failed")
	}
	ec2Client.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{InstanceIds: []*string{&_id}})

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
	ec2Client.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{VolumeIds: []*string{&rootDisk.DiskId}})
	ec2Client.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{VolumeIds: []*string{&tempInstance.Disks[0]}})

	err = self.AttachDisk(instance.GetId(), tempInstance.Disks[0], rootDisk.Device)
	if err != nil {
		log.Debugf("ReplaceSystemDisk attach disk %s: %s", tempInstance.Disks[0], err)
		return "", err
	}
	ec2Client.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{InstanceIds: []*string{&instanceId}})
	ec2Client.WaitUntilVolumeInUse(&ec2.DescribeVolumesInput{VolumeIds: []*string{&tempInstance.Disks[0]}})

	userdataText, err := base64.StdEncoding.DecodeString(userdata)
	if err != nil {
		return "", errors.Wrap(err, "SRegion.ReplaceSystemDisk.DecodeString")
	}
	err = instance.UpdateUserData(string(userdataText))
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

func (self *SRegion) ChangeVMConfig2(zoneId string, instanceId string, instanceType string, disks []*SDisk) error {
	params := &ec2.ModifyInstanceAttributeInput{}
	params.SetInstanceId(instanceId)

	t := &ec2.AttributeValue{Value: &instanceType}
	params.SetInstanceType(t)

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.ModifyInstanceAttribute(params)
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

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	_, err = ec2Client.DetachVolume(params)
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
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.AttachVolume(params)
	return errors.Wrap(err, "AttachVolume")
}

func (self *SRegion) deleteProtectStatusVM(instanceId string) (bool, error) {
	p := &ec2.DescribeInstanceAttributeInput{}
	p.SetInstanceId(instanceId)
	p.SetAttribute("disableApiTermination")
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return false, errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.DescribeInstanceAttribute(p)
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
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.ModifyInstanceAttribute(p2)
	return errors.Wrap(err, "ModifyInstanceAttribute")
}

func (self *SRegion) getPasswordData(instanceId string) (string, error) {
	params := &ec2.GetPasswordDataInput{}
	params.SetInstanceId(instanceId)

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return "", errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.GetPasswordData(params)
	if err != nil {
		return "", err
	}

	return *ret.PasswordData, nil
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

func (self *SInstance) SetTags(tags map[string]string, replace bool) error {
	ec2Client, err := self.host.zone.region.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	oldTagsJson, err := FetchTags(ec2Client, self.InstanceId)
	if err != nil {
		return errors.Wrapf(err, "FetchTags(self.host.zone.region.ec2Client, %s)", self.InstanceId)
	}
	oldTags := map[string]string{}
	err = oldTagsJson.Unmarshal(oldTags)
	if err != nil {
		return errors.Wrapf(err, "(%s).Unmarshal(oldTags)", oldTagsJson.String())
	}
	addTags := map[string]string{}
	for k, v := range tags {
		if strings.HasPrefix(k, "aws:") {
			return errors.Wrap(cloudprovider.ErrNotSupported, "The aws: prefix is reserved for AWS use")
		}
		if _, ok := oldTags[k]; !ok {
			addTags[k] = v
		} else {
			if oldTags[k] != v {
				addTags[k] = v
			}
		}
	}
	delTags := []string{}
	if replace {
		for k := range oldTags {
			if _, ok := tags[k]; !ok {
				if !strings.HasPrefix(k, "aws:") && k != "Name" {
					delTags = append(delTags, k)
				}
			}
		}
	}
	Arn := self.GetArn()
	err = self.host.zone.region.UntagResources([]string{Arn}, delTags)
	if err != nil {
		return errors.Wrapf(err, "self.host.zone.region.UntagResources([]string{%s}, %s)", Arn, jsonutils.Marshal(delTags).String())
	}
	delete(addTags, "Name")
	err = self.host.zone.region.TagResources([]string{Arn}, addTags)
	if err != nil {
		return errors.Wrapf(err, "self.host.zone.region.TagResources([]string{%s}, %s)", Arn, jsonutils.Marshal(addTags).String())
	}
	return nil
}

func (self *SInstance) GetAccountId() string {
	identity, err := self.host.zone.region.client.GetCallerIdentity()
	if err != nil {
		log.Errorf(err.Error() + "self.region.client.GetCallerIdentity()")
		return ""
	}
	return identity.Account
}

func (self *SInstance) GetArn() string {
	partition := ""
	switch self.host.zone.region.client.GetAccessEnv() {
	case api.CLOUD_ACCESS_ENV_AWS_GLOBAL:
		partition = "aws"
	case api.CLOUD_ACCESS_ENV_AWS_CHINA:
		partition = "aws-cn"
	default:
		partition = "aws"
	}
	return fmt.Sprintf("arn:%s:ec2:%s:%s:instance/%s", partition, self.host.zone.region.GetId(), self.GetAccountId(), self.InstanceId)
}

func (self *SRegion) SaveImage(instanceId string, opts *cloudprovider.SaveImageOptions) (*SImage, error) {
	params := map[string]string{
		"Description": opts.Notes,
		"InstanceId":  instanceId,
		"Name":        opts.Name,
	}
	ret := struct {
		ImageId string `xml:"imageId"`
	}{}
	err := self.ec2Request("CreateImage", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateImage")
	}
	err = cloudprovider.Wait(time.Second*10, time.Minute*5, func() (bool, error) {
		_, err := self.GetImage(ret.ImageId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return false, nil
			}
			return false, errors.Wrapf(err, "GetImage(%s)", ret.ImageId)
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "wait for image created")
	}
	image, err := self.GetImage(ret.ImageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage(%s)", ret.ImageId)
	}
	image.storageCache = self.getStoragecache()
	return image, nil
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	image, err := self.host.zone.region.SaveImage(self.InstanceId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveImage")
	}
	return image, nil
}
