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
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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

type SSecurityGroupIds struct {
	SecurityGroupId []string
}

type SVpcAttributes struct {
	PrivateIpAddress SIpAddress
	NetworkId        string // subnet id
	VpcId            string
}

type GroupSet struct {
	GroupId   string `xml:"groupId"`
	GroupName string `xml:"groupName"`
}

type Association struct {
	CarrierIp       string `xml:"carrierIp"`
	CustomerOwnedIp string `xml:"customerOwnedIp"`
	IpOwnerId       string `xml:"ipOwnerId"`
	PublicDnsName   string `xml:"publicDnsName"`
	PublicIp        string `xml:"publicIp"`
}

type BlockDeviceMapping struct {
	DeviceName string `xml:"deviceName"`
	Ebs        struct {
		AttachTime          time.Time `xml:"attachTime"`
		DeleteOnTermination bool      `xml:"deleteOnTermination"`
		Status              string    `xml:"status"`
		VolumeId            string    `xml:"volumeId"`
		VolumeSize          int       `xml:"volumeSize"`
	} `xml:"ebs"`
}

type ProductCode struct {
	productCode string `xml:"productCode"`
	Type        string `xml:"type"`
}

type StateReason struct {
	Code    string `xml:"code"`
	Message string `xml:"message"`
}

type SInstance struct {
	multicloud.SInstanceBase
	multicloud.AwsTags

	host *SHost
	img  *SImage

	Architecture       string               `xml:"architecture"`
	BlockDeviceMapping []BlockDeviceMapping `xml:"blockDeviceMapping"`

	BootMode              string `xml:"bootMode"`
	CapacityReservationId string `xml:"capacityReservationId"`

	CapacityReservationSpecification struct {
		CapacityReservationPreference string `xml:"capacityReservationPreference"`
		CapacityReservationTarget     struct {
			CapacityReservationId               string `xml:"capacityReservationId"`
			CapacityReservationResourceGroupArn string `xml:"capacityReservationResourceGroupArn"`
		} `xml:"capacityReservationTarget"`
	} `xml:"capacityReservationSpecification"`

	ClientToken string `xml:"clientToken"`
	CpuOptions  struct {
		CoreCount      int `xml:"coreCount"`
		ThreadsPerCore int `xml:"threadsPerCore"`
	} `xml:"cpuOptions"`
	DnsName                  string `xml:"dnsName"`
	EbsOptimized             bool   `xml:"ebsOptimized"`
	ElasticGpuAssociationSet []struct {
		ElasticGpuAssociationId    string `xml:"elasticGpuAssociationId"`
		ElasticGpuAssociationState string `xml:"elasticGpuAssociationState"`
		ElasticGpuAssociationTime  string `xml:"elasticGpuAssociationTime"`
		ElasticGpuId               string `xml:"elasticGpuId"`
	} `xml:"elasticGpuAssociationSet>item"`
	ElasticInferenceAcceleratorAssociationSet []struct {
		ElasticInferenceAcceleratorArn              string    `xml:"elasticInferenceAcceleratorArn"`
		ElasticInferenceAcceleratorAssociationId    string    `xml:"elasticInferenceAcceleratorAssociationId"`
		ElasticInferenceAcceleratorAssociationState string    `xml:"elasticInferenceAcceleratorAssociationState"`
		ElasticInferenceAcceleratorAssociationTime  time.Time `xml:"elasticInferenceAcceleratorAssociationTime"`
	} `xml:"elasticInferenceAcceleratorAssociationSet>item"`
	EnaSupport     bool `xml:"enaSupport"`
	EnclaveOptions struct {
		Enabled bool `xml:"enabled"`
	} `xml:"enclaveOptions"`
	GroupSet           []GroupSet `xml:"groupSet"`
	HibernationOptions struct {
		Configured bool `xml:"configured"`
	} `xml:"hibernationOptions"`
	Hypervisor         string `xml:"hypervisor"`
	IamInstanceProfile struct {
		Arn string `xml:"arn"`
		Id  string `xml:"id"`
	} `xml:"iamInstanceProfile"`
	ImageId           string `xml:"imageId"`
	InstanceId        string `xml:"instanceId"`
	InstanceLifecycle string `xml:"instanceLifecycle"`
	InstanceState     struct {
		Code int    `xml:"code"`
		Name string `xml:"name"`
	} `xml:"instanceState"`
	InstanceType string    `xml:"instanceType"`
	IpAddress    string    `xml:"ipAddress"`
	KernelId     string    `xml:"kernelId"`
	KeyName      string    `xml:"keyName"`
	LaunchTime   time.Time `xml:"launchTime"`
	LicenseSet   []struct {
		LicenseConfigurationArn string `xml:"licenseConfigurationArn"`
	} `xml:"licenseSet>item"`
	MetadataOptions struct {
		HttpEndpoint            string `xml:"httpEndpoint"`
		HttpProtocolIpv6        string `xml:"httpProtocolIpv6"`
		HttpPutResponseHopLimit int    `xml:"httpPutResponseHopLimit"`
		HttpTokens              string `xml:"httpTokens"`
		State                   string `xml:"state"`
	} `xml:"metadataOptions"`
	Monitoring struct {
		State string `xml:"state"`
	} `xml:"monitoring"`
	NetworkInterfaceSet []struct {
		Association Association `xml:"association"`
		Attachment  struct {
			AttachmentId        string    `xml:"attachmentId"`
			AttachTime          time.Time `xml:"attachTime"`
			DeleteOnTermination bool      `xml:"deleteOnTermination"`
			DeviceIndex         int       `xml:"deviceIndex"`
			NetworkCardIndex    int       `xml:"networkCardIndex"`
			Status              string    `xml:"status"`
		} `xml:"attachment"`
		Description   string     `xml:"description"`
		GroupSet      []GroupSet `xml:"groupSet"`
		InterfaceType string     `xml:"interfaceType"`
		Ipv4PrefixSet []struct {
			Ipv4Prefix string `xml:"ipv4Prefix>item"`
		} `xml:"ipv4PrefixSet"`
		Ipv6AddressesSet []struct {
			Ipv6Address string `xml:"ipv6Address"`
		} `xml:"ipv6AddressesSet>item"`
		MacAddress            string `xml:"macAddress"`
		NetworkInterfaceId    string `xml:"networkInterfaceId"`
		OwnerId               string `xml:"ownerId"`
		PrivateDnsName        string `xml:"privateDnsName"`
		PrivateIpAddress      string `xml:"privateIpAddress"`
		PrivateIpAddressesSet []struct {
			Association      Association `xml:"Association"`
			Primary          bool        `xml:"primary"`
			PrivateDnsName   string      `xml:"privateDnsName"`
			PrivateIpAddress string      `xml:"privateIpAddress"`
		} `xml:"privateIpAddressesSet>item"`
		SourceDestCheck bool   `xml:"sourceDestCheck"`
		Status          string `xml:"status"`
		SubnetId        string `xml:"subnetId"`
		VpcId           string `xml:"vpcId"`
	} `xml:"networkInterfaceSet>item"`
	OutpostArn string `xml:"outpostArn"`
	Placement  struct {
		Affinity             string `xml:"affinity"`
		AvailabilityZone     string `xml:"availabilityZone"`
		GroupName            string `xml:"groupName"`
		HostId               string `xml:"hostId"`
		HostResourceGroupArn string `xml:"hostResourceGroupArn"`
		PartitionNumber      int    `xml:"partitionNumber"`
		SpreadDomain         string `xml:"spreadDomain"`
		Tenancy              string `xml:"tenancy"`
	} `xml:"placement"`
	Platform                 string        `xml:"platform"`
	PlatformDetails          string        `xml:"platformDetails"`
	PrivateDnsName           string        `xml:"privateDnsName"`
	PrivateIpAddress         string        `xml:"privateIpAddress"`
	ProductCodes             []ProductCode `xml:"productCodes>item"`
	RamdiskId                string        `xml:"ramdiskId"`
	Reason                   string        `xml:"reason"`
	RootDeviceName           string        `xml:"rootDeviceName"`
	RootDeviceType           string        `xml:"rootDeviceType"`
	SourceDestCheck          bool          `xml:"sourceDestCheck"`
	SpotInstanceRequestId    string        `xml:"spotInstanceRequestId"`
	SriovNetSupport          string        `xml:"sriovNetSupport"`
	StateReason              StateReason   `xml:"stateReason"`
	SubnetId                 string        `xml:"subnetId"`
	UsageOperation           string        `xml:"usageOperation"`
	UsageOperationUpdateTime time.Time     `xml:"usageOperationUpdateTime"`
	VirtualizationType       string        `xml:"virtualizationType"`
	VpcId                    string        `xml:"vpcId"`
}

func (self *SInstance) UpdateUserData(userData string) error {
	return self.host.zone.region.UpdateUserData(self.InstanceId, userData)
}

func (self *SRegion) UpdateUserData(instanceId, userData string) error {
	params := map[string]string{
		"InstanceId":     instanceId,
		"UserData.Value": base64.StdEncoding.EncodeToString([]byte(userData)),
	}
	return self.ec2Request("ModifyInstanceAttribute", params, nil)
}

func (self *SInstance) GetUserData() (string, error) {
	return self.host.zone.region.GetUserData(self.InstanceId)
}

func (self *SRegion) GetUserData(instanceId string) (string, error) {
	params := map[string]string{
		"InstanceId": instanceId,
		"Attribute":  "userData",
	}
	ret := struct {
		UserData struct {
			Value string `xml:"value"`
		} `xml:"userData"`
	}{}
	err := self.ec2Request("DescribeInstanceAttribute", params, &ret)
	if err != nil {
		return "", errors.Wrapf(err, "DescribeInstanceAttribute")
	}
	data, err := base64.StdEncoding.DecodeString(ret.UserData.Value)
	if err != nil {
		return "", errors.Wrapf(err, "DecodeString")
	}
	return string(data), nil
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
	switch self.InstanceState.Name {
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
	return nil, nil
	//return self.SecurityGroupIds.SecurityGroupId, nil
}

func (self *SInstance) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.LaunchTime
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
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

func (self *SRegion) GetNetworkInterfaces(instanceId string) ([]SInstanceNic, error) {
	params := map[string]string{}
	if len(instanceId) > 0 {
		params["Filter.1.attachment.instance-id"] = instanceId
	}

	ret := []SInstanceNic{}
	for {
		result := struct {
			Nics      []SInstanceNic `xml:"networkInterfaceSet>item"`
			NextToken string         `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeNetworkInterfaces", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeNetworkInterfaces")
		}
		ret = append(ret, result.Nics...)
		if len(result.NextToken) == 0 || len(result.Nics) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics, err := self.host.zone.region.GetNetworkInterfaces(self.InstanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetworkInterfaces")
	}
	ret := []cloudprovider.ICloudNic{}
	for i := range nics {
		nics[i].region = self.host.zone.region
		ret = append(ret, &nics[i])
	}
	return ret, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eips, err := self.host.zone.region.GetEips(self.InstanceId, "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetEips")
	}
	for i := range eips {
		return &eips[i], nil
	}
	return nil, nil
}

func (self *SInstance) GetVcpuCount() int {
	return self.CpuOptions.CoreCount * self.CpuOptions.ThreadsPerCore
}

func (self *SInstance) GetVmemSizeMB() int {
	return 0 // self.Memory
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
	if strings.Contains(self.PlatformDetails, "Linux") {
		return cloudprovider.OsTypeLinux
	}
	return cloudprovider.OsTypeLinux
}

func (self *SInstance) GetOSName() string {
	img, _ := self.GetImage()
	if img != nil {
		return img.GetName()
	}
	return self.PlatformDetails
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
	return self.host.zone.region.StartVM(self.InstanceId)
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return self.host.zone.region.StopVM(self.InstanceId)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.DeleteVM(self.InstanceId)
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
	keypairName := self.KeyName
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

func (self *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	return self.host.zone.region.ChangeVMConfig(self.InstanceId, opts.InstanceType)
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
	deviceNames := []string{}
	for _, device := range img.BlockDeviceMapping {
		deviceNames = append(deviceNames, device.DeviceName)
	}
	for _, blockDevice := range self.BlockDeviceMapping {
		if !utils.IsInStringArray(blockDevice.DeviceName, deviceNames) {
			deviceNames = append(deviceNames, blockDevice.DeviceName)
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

func (self *SRegion) GetInstances(zoneId string, ids []string) ([]SInstance, error) {
	params := map[string]string{}
	idx := 1
	// skip terminated instance
	params[fmt.Sprintf("Filter.%d.Name", idx)] = "instance-state-name"
	for i, state := range []string{"pending", "running", "shutting-down", "stopping", "stopped"} {
		params[fmt.Sprintf("Filter.%d.Value.%d", idx, i+1)] = state
	}
	idx++
	if len(zoneId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "availability-zone"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = zoneId
		idx++
	}
	for i, id := range ids {
		params[fmt.Sprintf("InstanceId.%d", i)] = id
	}
	ret := []SInstance{}
	for {
		result := struct {
			ReservationSet []struct {
				ReservationId string      `xml:"reservationId"`
				OwnerId       string      `xml:"ownerId"`
				InstancesSet  []SInstance `xml:"instancesSet>item"`
			} `xml:"reservationSet>item"`
			NextToken string `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeInstances", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeInstances")
		}
		for _, instances := range result.ReservationSet {
			ret = append(ret, instances.InstancesSet...)
		}
		if len(result.NextToken) == 0 || len(result.ReservationSet) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, err := self.GetInstances("", []string{instanceId})
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}
	for i := range instances {
		if instances[i].InstanceId == instanceId {
			return &instances[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetInstances")
}

func (self *SRegion) CreateInstance(opts *cloudprovider.SManagedVMCreateConfig) (*SInstance, error) {
	img, err := self.GetImage(opts.ExternalImageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage")
	}

	if opts.SysDisk.SizeGB > 0 && opts.SysDisk.SizeGB < img.getRootDiskSizeGb() {
		opts.SysDisk.SizeGB = img.getRootDiskSizeGb()
	}

	rootDeviceName := "/dev/sda1"
	if len(img.RootDeviceName) > 0 {
		rootDeviceName = img.RootDeviceName
	}

	deviceNames := []string{rootDeviceName}

	params := map[string]string{
		"TagSpecification.1.ResourceType": "instance",
		"TagSpecification.1.Tags.1.Key":   "Name",
		"TagSpecification.1.Tags.1.Value": opts.Name,
		"TagSpecification.1.Tags.2.Key":   "Description",
		"TagSpecification.1.Tags.2.Value": opts.Description,

		"BlockDeviceMapping.1.DeviceName":              rootDeviceName,
		"BlockDeviceMapping.1.Ebs.DeleteOnTermination": "true",
		"BlockDeviceMapping.1.Ebs.VolumeSize":          fmt.Sprintf("%d", opts.SysDisk.SizeGB),
		"BlockDeviceMapping.1.Ebs.VolumeType":          opts.SysDisk.StorageType,
		"BlockDeviceMapping.1.Ebs.Iops":                fmt.Sprintf("%d", genDiskIops(opts.SysDisk.StorageType, opts.SysDisk.SizeGB)),

		"ImageId":                    opts.ExternalImageId,
		"InstanceType":               opts.InstanceType,
		"MaxCount":                   "1",
		"MinCount":                   "1",
		"Placement.AvailabilityZone": opts.ZoneId,
	}

	idx := 3
	for k, v := range opts.Tags {
		params[fmt.Sprintf("TagSpecification.1.Tags.%d.Key", idx)] = k
		params[fmt.Sprintf("TagSpecification.1.Tags.%d.Value", idx)] = v
		idx++
	}

	for i, disk := range opts.DataDisks {
		params[fmt.Sprintf("BlockDeviceMapping.%d.DeviceName", i+2)], err = NextDeviceName(deviceNames)
		if err != nil {
			return nil, errors.Wrapf(err, "NextDeviceName")
		}
		deviceNames = append(deviceNames, params[fmt.Sprintf("BlockDeviceMapping.%d.DeviceName", i+2)])
		params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.DeleteOnTermination", i+2)] = "true"
		params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.VolumeSize", i+2)] = fmt.Sprintf("%d", disk.SizeGB)
		params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.VolumeType", i+2)] = disk.StorageType
		params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.Iops", i+2)] = fmt.Sprintf("%d", genDiskIops(disk.StorageType, disk.SizeGB))
	}

	// keypair
	if len(opts.PublicKey) > 0 {
		keyName, err := self.syncKeypair(opts.PublicKey)
		if err != nil {
			return nil, err
		}
		params["KeyName"] = keyName
	}

	// user data
	if len(opts.UserData) > 0 {
		params["UserData"] = base64.StdEncoding.EncodeToString([]byte(opts.UserData))
	}

	// ip address
	if len(opts.IpAddr) > 0 {
		params["PrivateIpAddress"] = opts.IpAddr
	}

	// subnet id
	if len(opts.ExternalNetworkId) > 0 {
		params["SubnetId"] = opts.ExternalNetworkId
	}

	// security group
	for i, id := range opts.ExternalSecgroupIds {
		params[fmt.Sprintf("SecurityGroupId.%d", i+1)] = id
	}

	ret := struct {
		Instances []SInstance `xml:"instancesSet>item"`
	}{}

	err = self.ec2Request("RunInstances", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "RunInstances")
	}

	for i := range ret.Instances {
		return &ret.Instances[i], nil
	}

	return nil, fmt.Errorf("no instance return after instance create")
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.InstanceState.Name, nil
}

func (self *SRegion) StartVM(instanceId string) error {
	params := map[string]string{
		"InstanceId.1": instanceId,
	}
	return self.ec2Request("StartInstances", params, nil)
}

func (self *SRegion) StopVM(instanceId string) error {
	params := map[string]string{
		"InstanceId.1": instanceId,
	}
	return self.ec2Request("StopInstances", params, nil)
}

func (self *SRegion) DeleteVM(instanceId string) error {
	self.DisableVMDetete(instanceId, false)
	params := map[string]string{
		"InstanceId.1": instanceId,
	}
	return self.ec2Request("TerminateInstances", params, nil)
}

func (self *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, desc string) error {
	params := map[string]string{
		"ResourceId.1": instanceId,
	}
	idx := 1
	if len(name) > 0 {
		params[fmt.Sprintf("Tag.%d.Key", idx)] = "Name"
		params[fmt.Sprintf("Tag.%d.Value", idx)] = name
		idx++
	}
	if len(desc) > 0 {
		params[fmt.Sprintf("Tag.%d.Key", idx)] = "Description"
		params[fmt.Sprintf("Tag.%d.Value", idx)] = desc
		idx++
	}

	return self.ec2Request("CreateTags", params, nil)
}

func (self *SRegion) UpdateVM(instanceId string, hostname string) error {
	// https://docs.aws.amazon.com/zh_cn/AWSEC2/latest/UserGuide/set-hostname.html
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) ReplaceSystemDisk(ctx context.Context, instanceId string, image *SImage, sysDiskSizeGB int, keypair string, userdata string) (string, error) {
	/*
		instance, err := self.GetInstance(instanceId)
		if err != nil {
			return "", err
		}
		disks, _, err := self.GetDisks(instanceId, instance.Placement.AvailabilityZone, "", nil, 0, 0)
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
		if len(instance.SubnetId) > 0 {
			subnetId = instance.SubnetId
		}

		// create tmp server
		tempName := fmt.Sprintf("__tmp_%s", instance.GetName())
		_id, err := self.CreateInstance(tempName,
			image,
			instance.InstanceType,
			subnetId,
			"",
			instance.Placement.AvailabilityZone,
			instance.GetDesc(),
			[]SDisk{{Size: sysDiskSizeGB, Category: rootDisk.Category}},
			"",
			keypair,
			userdata,
			nil,
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
	*/
	return "", nil
}

func (self *SRegion) ChangeVMConfig(instanceId string, instanceType string) error {
	params := map[string]string{
		"InstanceId":         instanceId,
		"InstanceType.Value": instanceType,
	}
	return self.ec2Request("ModifyInstanceAttribute", params, nil)
}

func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	params := map[string]string{
		"InstanceId": instanceId,
		"VolumeId":   diskId,
	}
	err := self.ec2Request("DetachVolume", params, nil)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return err
	}
	return nil
}

func (self *SRegion) AttachDisk(instanceId string, diskId string, deviceName string) error {
	params := map[string]string{
		"InstanceId": instanceId,
		"VolumeId":   diskId,
		"Device":     deviceName,
	}
	return self.ec2Request("AttachVolume", params, nil)
}

func (self *SRegion) DisableVMDetete(instanceId string, disableDelete bool) error {
	params := map[string]string{
		"InstanceId": instanceId,
		"Attribute":  "disableApiTermination",
		"Value":      fmt.Sprintf("%v", disableDelete),
	}
	return self.ec2Request("ModifyInstanceAttribute", params, nil)
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) SetTags(tags map[string]string, replace bool) error {
	oldTags, err := self.host.zone.region.GetResourceTags(self.InstanceId)
	if err != nil {
		return errors.Wrapf(err, "GetResourceTags")
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
		return errors.Wrapf(err, "UntagResources")
	}
	delete(addTags, "Name")
	err = self.host.zone.region.TagResources([]string{Arn}, addTags)
	if err != nil {
		return errors.Wrapf(err, "TagResources")
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
