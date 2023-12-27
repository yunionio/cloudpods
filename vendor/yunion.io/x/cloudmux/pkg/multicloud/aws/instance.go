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

type EbsInstanceBlockDevice struct {
	AttachTime          time.Time `xml:"attachTime"`
	DeleteOnTermination bool      `xml:"deleteOnTermination"`
	Status              string    `xml:"status"`
	VolumeId            string    `xml:"volumeId"`
}

type InstanceBlockDeviceMapping struct {
	DeviceName *string                `xml:"deviceName"`
	Ebs        EbsInstanceBlockDevice `xml:"ebs"`
}

type SInstance struct {
	multicloud.SInstanceBase
	AwsTags

	host *SHost
	img  *SImage

	AmiLaunchIndex        int64                        `xml:"amiLaunchIndex"`
	Architecture          string                       `xml:"architecture"`
	BlockDeviceMappings   []InstanceBlockDeviceMapping `xml:"blockDeviceMapping>item"`
	BootMode              string                       `xml:"bootMode"`
	CapacityReservationId string                       `xml:"capacityReservationId"`
	ClientToken           string                       `xml:"clientToken"`
	CpuOptions            struct {
		CoreCount      int `xml:"coreCount"`
		ThreadsPerCore int `xml:"threadsPerCore"`
	} `xml:"cpuOptions"`
	EbsOptimized           bool `xml:"ebsOptimized"`
	ElasticGpuAssociations []struct {
		ElasticGpuAssociationId    string `xml:"elasticGpuAssociationId"`
		ElasticGpuAssociationState string `xml:"elasticGpuAssociationState"`
		ElasticGpuAssociationTime  string `xml:"elasticGpuAssociationTime"`
		ElasticGpuId               string `xml:"elasticGpuId"`
	} `xml:"elasticGpuAssociationSet>item"`
	EnaSupport     bool `xml:"enaSupport"`
	EnclaveOptions struct {
		Enabled bool `xml:"enabled"`
	} `xml:"enclaveOptions"`
	HibernationOptions struct {
		Configured bool `xml:"configured"`
	} `xml:"hibernationOptions"`
	Hypervisor         string `xml:"hypervisor"`
	IamInstanceProfile struct {
		Id  string `xml:"id"`
		Arn string `xml:"arn"`
	} `xml:"iamInstanceProfile"`
	ImageId           string    `xml:"imageId"`
	InstanceId        string    `xml:"instanceId"`
	InstanceLifecycle string    `xml:"instanceLifecycle"`
	InstanceType      string    `xml:"instanceType"`
	KernelId          string    `xml:"kernelId"`
	KeyName           string    `xml:"keyName"`
	LaunchTime        time.Time `xml:"launchTime"`
	Licenses          []struct {
		LicenseConfigurationArn string `xml:"licenseConfigurationArn"`
	} `xml:"licenseSet>item"`
	MetadataOptions struct {
		HttpEndpoint            string `xml:"httpEndpoint"`
		HttpPutResponseHopLimit int64  `xml:"httpPutResponseHopLimit"`
		HttpTokens              string `xml:"httpTokens"`
		State                   string `xml:"state"`
	} `xml:"metadataOptions"`
	Monitoring struct {
		State string `xml:"state"`
	} `xml:"monitoring"`
	NetworkInterfaces []SNetworkInterface `xml:"networkInterfaceSet>item"`
	OutpostArn        string              `xml:"outpostArn"`
	Placement         struct {
		Affinity             string `xml:"affinity"`
		AvailabilityZone     string `xml:"availabilityZone"`
		GroupName            string `xml:"groupName"`
		HostId               string `xml:"hostId"`
		HostResourceGroupArn string `xml:"hostResourceGroupArn"`
		PartitionNumber      int64  `xml:"partitionNumber"`
		SpreadDomain         string `xml:"spreadDomain"`
		Tenancy              string `xml:"tenancy"`
	} `xml:"placement"`
	Platform         string `xml:"platform"`
	PrivateDnsName   string `xml:"privateDnsName"`
	PrivateIpAddress string `xml:"privateIpAddress"`
	ProductCodes     []struct {
		ProductCodeId   string `xml:"productCode"`
		ProductCodeType string `xml:"type"`
	} `xml:"productCodes>item"`
	PublicDnsName   string `xml:"dnsName"`
	PublicIpAddress string `xml:"ipAddress"`
	RamdiskId       string `xml:"ramdiskId"`
	RootDeviceName  string `xml:"rootDeviceName"`
	RootDeviceType  string `xml:"rootDeviceType"`
	SecurityGroups  []struct {
		GroupId   string `xml:"groupId"`
		GroupName string `xml:"groupName"`
	} `xml:"groupSet>item"`
	SourceDestCheck       bool   `xml:"sourceDestCheck"`
	SpotInstanceRequestId string `xml:"spotInstanceRequestId"`
	SriovNetSupport       string `xml:"sriovNetSupport"`
	State                 struct {
		Code int64  `xml:"code"`
		Name string `xml:"name"`
	} `xml:"instanceState"`
	StateReason struct {
		Code    string `xml:"code"`
		Message string `xml:"message"`
	} `xml:"stateReason"`
	StateTransitionReason string `xml:"reason"`
	SubnetId              string `xml:"subnetId"`
	VirtualizationType    string `xml:"virtualizationType"`
	VpcId                 string `xml:"vpcId"`
}

func (self *SInstance) UpdateUserData(userData string) error {
	return self.host.zone.region.ModifyInstanceAttribute(self.InstanceId, &SInstanceAttr{UserData: userData})
}

func (self *SInstance) GetUserData() (string, error) {
	ret, err := self.host.zone.region.DescribeInstanceAttribute(self.InstanceId, &InstanceAttributeInput{UserData: true})
	if err != nil {
		return "", errors.Wrapf(err, "DescribeInstanceAttribute")
	}
	udata, err := base64.StdEncoding.DecodeString(ret.UserData.Value)
	return string(udata), err
}

type InstanceAttribute struct {
	UserData struct {
		Value string `xml:"value"`
	} `xml:"userData"`
}

type InstanceAttributeInput struct {
	UserData bool
}

func (self *SRegion) DescribeInstanceAttribute(id string, opts *InstanceAttributeInput) (*InstanceAttribute, error) {
	params := map[string]string{
		"InstanceId": id,
	}
	if opts.UserData {
		params["Attribute"] = "userData"
	}

	ret := &InstanceAttribute{}
	err := self.ec2Request("DescribeInstanceAttribute", params, ret)
	return ret, err

}

func (self *SInstance) GetId() string {
	return self.InstanceId
}

func (self *SInstance) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.InstanceId
}

func (self *SInstance) GetHostname() string {
	return self.GetName()
}

func (self *SInstance) GetGlobalId() string {
	return self.InstanceId
}

func (self *SInstance) GetStatus() string {
	switch self.State.Name {
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

func (self *SInstance) GetInstanceType() string {
	return self.InstanceType
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	for _, group := range self.SecurityGroups {
		ret = append(ret, group.GroupId)
	}
	return ret, nil
}

func (self *SInstance) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.LaunchTime
}

func (self *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetThroughput() int {
	return 0
}

func (self *SInstance) GetInternetMaxBandwidthOut() int {
	return 0
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.host.zone.region.GetDisks(self.InstanceId, "", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetDisks")
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		store, err := self.host.zone.getStorageByCategory(disks[i].VolumeType)
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
	if len(self.PublicIpAddress) > 0 {
		eip, err := self.host.zone.region.GetEipByIpAddress(self.PublicIpAddress)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				eip := SEipAddress{region: self.host.zone.region}
				eip.region = self.host.zone.region
				eip.PublicIp = self.PublicIpAddress
				eip.InstanceId = self.InstanceId
				eip.AllocationId = self.InstanceId // fixed. AllocationId等于InstanceId即表示为 仿真EIP。
				return &eip, nil
			}
			return nil, err
		}
		return eip, nil
	}
	for _, nic := range self.NetworkInterfaces {
		if len(nic.Association.PublicIp) > 0 {
			eip := SEipAddress{region: self.host.zone.region}
			eip.region = self.host.zone.region
			eip.PublicIp = nic.Association.PublicIp
			eip.InstanceId = self.InstanceId
			eip.AllocationId = self.InstanceId // fixed. AllocationId等于InstanceId即表示为 仿真EIP。
			return &eip, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty eip")
}

func (self *SInstance) GetVcpuCount() int {
	return self.CpuOptions.CoreCount * self.CpuOptions.ThreadsPerCore
}

func (self *SInstance) GetVmemSizeMB() int {
	instanceType, _ := self.host.zone.region.GetInstanceType(self.InstanceType)
	if instanceType != nil {
		return instanceType.MemoryInfo.SizeInMiB
	}
	return 0
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
	if len(self.Platform) > 0 {
		return cloudprovider.TOsType(osprofile.NormalizeOSType(self.Platform))
	}
	return cloudprovider.OsTypeLinux
}

func (self *SInstance) GetFullOsName() string {
	img, err := self.GetImage()
	if err != nil {
		return ""
	}
	return img.ImageName
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
	if len(self.Architecture) > 0 {
		switch self.Architecture {
		case "arm64":
			return apis.OS_ARCH_AARCH64
		case "i386":
			return apis.OS_ARCH_X86
		case "x86_64":
			return apis.OS_ARCH_X86_64
		default:
			return apis.OS_ARCH_X86_64
		}
	}
	img, err := self.GetImage()
	if err != nil {
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

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return self.host.zone.region.assignSecurityGroups(secgroupIds, self.InstanceId)
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
	err := cloudprovider.Wait(time.Second*4, time.Minute*10, func() (bool, error) {
		if utils.IsInStringArray(self.State.Name, []string{"running", "pending", "stopping", "stopped"}) {
			return true, nil
		}
		err := self.Refresh()
		if err != nil {
			return false, err
		}
		return false, nil
	})
	if err != nil {
		log.Errorf("wait instance status stoped, current status: %s error: %v", self.State.Name, err)
	}
	return self.host.zone.region.StopVM(self.InstanceId, opts.IsForce)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	err := cloudprovider.Wait(time.Second*4, time.Minute*10, func() (bool, error) {
		if utils.IsInStringArray(self.State.Name, []string{"running", "pending", "stopping", "stopped"}) {
			return true, nil
		}
		err := self.Refresh()
		if err != nil {
			return false, err
		}
		return false, nil
	})
	if err != nil {
		log.Errorf("wait instance status stoped to delete, current status: %s error: %v", self.State.Name, err)
	}
	return self.host.zone.region.DeleteVM(self.InstanceId)
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return self.host.zone.region.UpdateVM(self.InstanceId, input)
}

func (self *SRegion) UpdateVM(instanceId string, input cloudprovider.SInstanceUpdateOptions) error {
	return self.setTags("instance", instanceId, map[string]string{"Name": input.NAME, "Description": input.Description}, false)
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
	keypairName := self.KeyName
	if len(desc.PublicKey) > 0 {
		keypairName, err = self.host.zone.region.SyncKeypair(desc.PublicKey)
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

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if len(config.InstanceType) > 0 {
		return self.ChangeConfig2(ctx, config.InstanceType)
	}
	return errors.Wrap(errors.ErrClient, "Instance.ChangeConfig.InstanceTypeIsEmpty")
}

func (self *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	return self.host.zone.region.ChangeVMConfig2(self.InstanceId, instanceType)
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

	deviceNames := []string{}
	// mix in image block device names
	for i := range img.BlockDeviceMapping {
		if !utils.IsInStringArray(img.BlockDeviceMapping[i].DeviceName, deviceNames) {
			deviceNames = append(deviceNames, img.BlockDeviceMapping[i].DeviceName)
		}
	}

	name, err := NextDeviceName(deviceNames)
	if err != nil {
		return err
	}

	err = self.host.zone.region.AttachDisk(self.InstanceId, diskId, name)
	if err != nil {
		return err
	}
	return nil
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.DetachDisk(self.InstanceId, diskId)
}

func (self *SInstance) getVpc() (*SVpc, error) {
	return self.host.zone.region.getVpc(self.VpcId)
}

func (self *SRegion) GetInstances(zoneId, imageId string, ids []string) ([]SInstance, error) {
	params := map[string]string{}
	idx := 1
	if len(zoneId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "availability-zone"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = zoneId
		idx++
	}
	if len(imageId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "image-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = imageId
		idx++
	}
	// skip terminated instance
	params[fmt.Sprintf("Filter.%d.Name", idx)] = "instance-state-name"
	for i, state := range []string{"pending", "running", "shutting-down", "stopping", "stopped"} {
		params[fmt.Sprintf("Filter.%d.Value.%d", idx, i+1)] = state
	}
	idx++

	for i, id := range ids {
		params[fmt.Sprintf("InstanceId.%d", i+1)] = id
	}
	ret := []SInstance{}
	for {
		part := struct {
			NextToken      string `xml:"nextToken"`
			ReservationSet []struct {
				InstancesSet []SInstance `xml:"instancesSet>item"`
			} `xml:"reservationSet>item"`
		}{}
		err := self.ec2Request("DescribeInstances", params, &part)
		if err != nil {
			return nil, err
		}
		for _, res := range part.ReservationSet {
			ret = append(ret, res.InstancesSet...)
		}
		if len(part.ReservationSet) == 0 || len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, err := self.GetInstances("", "", []string{instanceId})
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}
	for i := range instances {
		if instances[i].InstanceId == instanceId {
			return &instances[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, instanceId)
}

func (self *SRegion) GetInstanceIdByImageId(imageId string) (string, error) {
	instances, err := self.GetInstances("", imageId, nil)
	if err != nil {
		return "", err
	}
	for i := range instances {
		return instances[i].InstanceId, nil
	}
	return "", fmt.Errorf("instance launch with image %s not found", imageId)
}

func (self *SRegion) CreateInstance(name string, image *SImage, instanceType string, subnetId string, secgroupIds []string,
	zoneId string, desc string, disks []cloudprovider.SDiskInfo, ipAddr string,
	keypair string, userData string, tags map[string]string, enableMonitorAgent bool,
) (*SInstance, error) {
	params := map[string]string{}
	for i, disk := range disks {
		deviceName := image.RootDeviceName
		if i == 0 && len(deviceName) == 0 {
			deviceName = "/dev/sda1"
		}
		if i > 0 {
			var err error
			deviceName, err = NextDeviceName(image.GetBlockDeviceNames())
			if err != nil {
				return nil, errors.Wrapf(err, "NextDeviceName")
			}
		}
		params[fmt.Sprintf("BlockDeviceMapping.%d.DeviceName", i+1)] = deviceName
		params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.DeleteOnTermination", i+1)] = "true"
		params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.VolumeSize", i+1)] = fmt.Sprintf("%d", disk.SizeGB)
		params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.VolumeType", i+1)] = disk.StorageType
		iops := disk.Iops
		if iops == 0 {
			iops = int(GenDiskIops(disk.StorageType, disk.SizeGB))
		}
		if utils.IsInStringArray(disk.StorageType, []string{
			api.STORAGE_IO1_SSD,
			api.STORAGE_IO2_SSD,
			api.STORAGE_GP3_SSD,
		}) {
			params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.Iops", i+1)] = fmt.Sprintf("%d", iops)
		}
		if disk.Throughput >= 125 && disk.Throughput <= 1000 && disk.StorageType == api.STORAGE_GP3_SSD {
			params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.Throughput", i+1)] = fmt.Sprintf("%d", disk.Throughput)
		}
	}

	tagIdx := 1
	params[fmt.Sprintf("TagSpecification.1.ResourceType")] = "instance"
	params[fmt.Sprintf("TagSpecification.1.Tag.%d.Key", tagIdx)] = "Name"
	params[fmt.Sprintf("TagSpecification.1.Tag.%d.Value", tagIdx)] = name
	tagIdx++
	if len(desc) > 0 {
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Key", tagIdx)] = "Description"
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Value", tagIdx)] = desc
		tagIdx++
	}
	for k, v := range tags {
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Key", tagIdx)] = k
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Value", tagIdx)] = v
		tagIdx++
	}
	params["ImageId"] = image.ImageId
	params["InstanceType"] = instanceType
	params["MaxCount"] = "1"
	params["MinCount"] = "1"
	params["Placement.AvailabilityZone"] = zoneId
	params["Monitoring.Enabled"] = fmt.Sprintf("%v", enableMonitorAgent)
	// keypair
	if len(keypair) > 0 {
		params["KeyName"] = keypair
	}

	// user data
	if len(userData) > 0 {
		params["UserData"] = userData
	}

	// ip address
	if len(ipAddr) > 0 {
		params["PrivateIpAddress"] = ipAddr
	}

	// subnet id
	if len(subnetId) > 0 {
		params["SubnetId"] = subnetId
	}

	// security group
	for i, id := range secgroupIds {
		params[fmt.Sprintf("SecurityGroupId.%d", i+1)] = id
	}

	ret := struct {
		InstancesSet []SInstance `xml:"instancesSet>item"`
	}{}
	err := self.ec2Request("RunInstances", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "RunInstances")
	}

	for i := range ret.InstancesSet {
		return &ret.InstancesSet[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (self *SRegion) StartVM(instanceId string) error {
	params := map[string]string{
		"InstanceId.1": instanceId,
	}
	ret := struct{}{}
	return self.ec2Request("StartInstances", params, &ret)
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	params := map[string]string{
		"InstanceId.1": instanceId,
	}
	if isForce {
		params["Force"] = "true"
	}
	ret := struct{}{}
	return self.ec2Request("StopInstances", params, &ret)
}

func (self *SRegion) DeleteVM(instanceId string) error {
	disableApiTermination := false
	err := self.ModifyInstanceAttribute(instanceId, &SInstanceAttr{
		DisableApiTermination: &disableApiTermination,
	})
	if err != nil {
		return err
	}
	params := map[string]string{
		"InstanceId.1": instanceId,
	}
	ret := struct{}{}
	return self.ec2Request("TerminateInstances", params, &ret)
}

func (self *SRegion) ReplaceSystemDisk(ctx context.Context, instanceId string, image *SImage, sysDiskSizeGB int, keypair string, userdata string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	disks, err := self.GetDisks(instanceId, instance.Placement.AvailabilityZone, "", nil)
	if err != nil {
		return "", err
	}

	var rootDisk *SDisk
	for _, disk := range disks {
		if disk.GetDiskType() == api.DISK_TYPE_SYS {
			rootDisk = &disk
			break
		}
	}

	if rootDisk == nil {
		return "", fmt.Errorf("can not find root disk of instance %s", instanceId)
	}
	log.Debugf("ReplaceSystemDisk replace root disk %s", rootDisk.VolumeId)

	subnetId := instance.SubnetId

	// create tmp server
	tempName := fmt.Sprintf("__tmp_%s", instance.GetName())
	vm, err := self.CreateInstance(tempName,
		image,
		instance.InstanceType,
		subnetId,
		[]string{},
		instance.Placement.AvailabilityZone,
		instance.GetDescription(),
		[]cloudprovider.SDiskInfo{{SizeGB: sysDiskSizeGB, StorageType: rootDisk.VolumeType}},
		"",
		keypair,
		userdata,
		nil,
		false,
	)
	if err == nil {
		defer self.DeleteVM(vm.InstanceId)
	} else {
		return "", fmt.Errorf("ReplaceSystemDisk create temp server failed.")
	}

	cloudprovider.Wait(time.Second*2, time.Minute*3, func() (bool, error) {
		instance, err := self.GetInstance(vm.InstanceId)
		if err != nil {
			return false, errors.Wrapf(err, "GetInstance")
		}
		if instance.GetStatus() == api.VM_RUNNING {
			return true, nil
		}
		return false, nil
	})

	err = self.StopVM(vm.InstanceId, true)
	if err != nil {
		return "", errors.Wrapf(err, "StopVM")
	}

	cloudprovider.Wait(time.Second*2, time.Minute*3, func() (bool, error) {
		instance, err := self.GetInstance(vm.InstanceId)
		if err != nil {
			return false, errors.Wrapf(err, "GetInstance")
		}
		if instance.GetStatus() == api.VM_READY {
			return true, nil
		}
		return false, nil
	})

	// detach disks
	tempInstance, err := self.GetInstance(vm.InstanceId)
	if err != nil {
		return "", errors.Wrapf(err, "GetInstance")
	}

	err = self.DetachDisk(instance.GetId(), rootDisk.VolumeId)
	if err != nil {
		return "", errors.Wrapf(err, "DetachDisk")
	}

	err = self.DetachDisk(tempInstance.GetId(), tempInstance.BlockDeviceMappings[0].Ebs.VolumeId)
	if err != nil {
		return "", errors.Wrapf(err, "DetachDisk")
	}

	err = self.AttachDisk(instance.GetId(), tempInstance.BlockDeviceMappings[0].Ebs.VolumeId, rootDisk.getDevice())
	if err != nil {
		return "", errors.Wrapf(err, "ttachDisk")
	}

	err = self.ModifyInstanceAttribute(instance.InstanceId, &SInstanceAttr{UserData: userdata})
	if err != nil {
		return "", errors.Wrapf(err, "ModifyInstanceAttribute")
	}

	err = self.DeleteDisk(rootDisk.VolumeId)
	if err != nil {
		log.Errorf("DeleteDisk %s", rootDisk.VolumeId)
	}
	return tempInstance.BlockDeviceMappings[0].Ebs.VolumeId, nil
}

func (self *SRegion) ChangeVMConfig2(instanceId string, instanceType string) error {
	return self.ModifyInstanceAttribute(instanceId, &SInstanceAttr{InstanceType: instanceType})
}

func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	params := map[string]string{
		"InstanceId": instanceId,
		"VolumeId":   diskId,
	}

	ret := struct{}{}
	err := self.ec2Request("DetachVolume", params, &ret)
	if err != nil {
		if strings.Contains(err.Error(), "in the 'available' state") {
			return nil
		}
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "DetachVolume")
	}

	return nil
}

func (self *SRegion) AttachDisk(instanceId string, diskId string, deviceName string) error {
	params := map[string]string{
		"InstanceId": instanceId,
		"VolumeId":   diskId,
		"Device":     deviceName,
	}

	ret := struct{}{}
	return self.ec2Request("AttachVolume", params, &ret)
}

type SInstanceAttr struct {
	DisableApiTermination *bool
	InstanceType          string
	UserData              string
}

func (self *SRegion) ModifyInstanceAttribute(instanceId string, opts *SInstanceAttr) error {
	params := map[string]string{
		"InstanceId": instanceId,
	}
	if len(opts.InstanceType) > 0 {
		params["InstanceType.Value"] = opts.InstanceType
	}
	if len(opts.UserData) > 0 {
		params["UserData.Value"] = opts.UserData
	}
	if opts.DisableApiTermination != nil {
		params["DisableApiTermination.Value"] = fmt.Sprintf("%v", opts.DisableApiTermination)
	}
	ret := struct{}{}
	return self.ec2Request("ModifyInstanceAttribute", params, &ret)
}

func (self *SRegion) GetPasswordData(instanceId string) (string, error) {
	params := map[string]string{
		"InstanceId": instanceId,
	}
	ret := struct {
		PasswordData string `xml:"passwordData"`
	}{}
	err := self.ec2Request("GetPasswordData", params, &ret)
	if err != nil {
		return "", errors.Wrapf(err, "GetPasswordData")
	}
	return ret.PasswordData, nil
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
	return self.host.zone.region.setTags("instance", self.InstanceId, tags, replace)
}

func (self *SInstance) GetAccountId() string {
	identity, err := self.host.zone.region.client.GetCallerIdentity()
	if err != nil {
		log.Errorf(err.Error() + "self.region.client.GetCallerIdentity()")
		return ""
	}
	return identity.Account
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
	image.storageCache = self.getStorageCache()
	return image, nil
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	image, err := self.host.zone.region.SaveImage(self.InstanceId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveImage")
	}
	return image, nil
}

func (ins *SInstance) GetDescription() string {
	return ins.AwsTags.GetDescription()
}
