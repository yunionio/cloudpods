package aws

import (
	"fmt"
	"strings"
	"time"

	"context"
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/osprofile"
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
	host       *SHost
	RegionId   string
	ZoneId     string
	InstanceId string
	ImageId    string

	HostName          string
	InstanceName      string
	InstanceType      string
	Cpu               int8
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
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetId() string {
	return self.InstanceId
}

func (self *SInstance) GetName() string {
	return self.InstanceName
}

func (self *SInstance) GetGlobalId() string {
	return self.InstanceId
}

func (self *SInstance) GetStatus() string {
	switch self.Status {
	case InstanceStatusRunning:
		return models.VM_RUNNING
	case InstanceStatusPending: // todo: pending ?
		return models.VM_STARTING
	case InstanceStatusStopping:
		return models.VM_STOPPING
	case InstanceStatusStopped:
		return models.VM_READY
	default:
		return models.VM_UNKNOWN
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
		log.Errorf(err.Error())
	}
	data.Update(tags)

	data.Add(jsonutils.NewString(self.host.zone.GetGlobalId()), "zone_ext_id")
	if len(self.ImageId) > 0 {
		if image, err := self.host.zone.region.GetImage(self.ImageId); err != nil {
			log.Errorf("Failed to find image %s for instance %s zone %s", self.ImageId, self.GetId(), self.ZoneId)
		} else if meta := image.GetMetadata(); meta != nil {
			data.Update(meta)
		}
	}
	for _, secgroupId := range self.SecurityGroupIds.SecurityGroupId {
		if len(secgroupId) > 0 {
			data.Add(jsonutils.NewString(secgroupId), "secgroupId")
			break
		}
	}

	return data
}

func (self *SInstance) GetBillingType() string {
	// todo: implement me
	return models.BILLING_TYPE_POSTPAID
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
	// todo: implement me
	if len(self.PublicIpAddress.IpAddress) > 0 {
		eip := SEipAddress{}
		eip.region = self.host.zone.region
		eip.IpAddress = self.PublicIpAddress.IpAddress[0]
		eip.InstanceId = self.InstanceId
		eip.AllocationId = self.InstanceId // fixed
		eip.Bandwidth = 10000
		return &eip, nil
	} else if len(self.EipAddress.IpAddress) > 0 {
		return self.host.zone.region.GetEip(self.EipAddress.AllocationId)
	} else {
		return nil, nil
	}
}

func (self *SInstance) GetVcpuCount() int8 {
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
	return self.host.zone.region.assignSecurityGroup(secgroupId, self.InstanceId)
}

func (self *SInstance) GetHypervisor() string {
	return models.HYPERVISOR_AWS
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

		if self.GetStatus() == models.VM_RUNNING {
			return nil
		} else if self.GetStatus() == models.VM_READY {
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
	return cloudprovider.WaitStatus(self, models.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	for {
		err := self.host.zone.region.DeleteVM(self.InstanceId)
		if err != nil {
			return err
		} else {
			break
		}
	}

	return self.host.zone.region.ec2Client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{InstanceIds: ConvertedList([]string{self.InstanceId})})
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return self.host.zone.region.UpdateVM(self.InstanceId, name)
}

func (self *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	if len(publicKey) > 0 || len(passwd) > 0 {
		return "", fmt.Errorf("aws rebuild root not support specific publickey/password")
	}

	diskId, err := self.host.zone.region.ReplaceSystemDisk(ctx, self.InstanceId, imageId, sysSizeGB)
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
	panic("implement me")
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

	log.Debugf("GetInstances with params: %s", params.String())
	res, err := self.ec2Client.DescribeInstances(params)
	if err != nil {
		log.Errorf("GetInstances fail %s", err)
		return nil, 0, err
	}

	instances := []SInstance{}
	for _, reservation := range res.Reservations {
		for _, instance := range reservation.Instances {
			if err := FillZero(instance); err != nil {
				return nil, 0, err
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
			for _, n := range instance.NetworkInterfaces {
				i := SNetworkInterface{
					MacAddress:         *n.MacAddress,
					NetworkInterfaceId: *n.NetworkInterfaceId,
					PrimaryIpAddress:   *n.PrivateIpAddress,
				}
				networkInterfaces.NetworkInterface = append(networkInterfaces.NetworkInterface, i)
			}

			var vpcattr SVpcAttributes
			vpcattr.VpcId = *instance.VpcId
			vpcattr.PrivateIpAddress = SIpAddress{[]string{*instance.PrivateIpAddress}}
			vpcattr.NetworkId = *instance.SubnetId

			var productCodes []string
			for _, p := range instance.ProductCodes {
				productCodes = append(productCodes, *p.ProductCodeId)
			}

			szone, err := self.getZoneById(*instance.Placement.AvailabilityZone)
			if err != nil {
				return nil, 0, err
			}

			image, err := self.GetImage(*instance.ImageId)
			if err != nil {
				return nil, 0, err
			}

			host := szone.getHost()

			sinstance := SInstance{
				RegionId:          self.RegionId,
				host:              host,
				ZoneId:            *instance.Placement.AvailabilityZone,
				InstanceId:        *instance.InstanceId,
				ImageId:           *instance.ImageId,
				InstanceType:      *instance.InstanceType,
				Cpu:               int8(*instance.CpuOptions.CoreCount),
				IoOptimized:       *instance.EbsOptimized,
				KeyPairName:       *instance.KeyName,
				CreationTime:      *instance.LaunchTime,
				PublicDNSName:     *instance.PublicDnsName,
				RootDeviceName:    *instance.RootDeviceName,
				Status:            *instance.State.Name,
				InnerIpAddress:    SIpAddress{[]string{*instance.PrivateIpAddress}},
				PublicIpAddress:   SIpAddress{[]string{*instance.PublicIpAddress}},
				InstanceName:      tagspec.GetNameTag(),
				Description:       tagspec.GetDescTag(),
				Disks:             disks,
				DeviceNames:       devicenames,
				SecurityGroupIds:  secgroups,
				NetworkInterfaces: networkInterfaces,
				VpcAttributes:     vpcattr,
				ProductCodes:      productCodes,
				OSName:            image.OSName, // todo: 这里在model层回写OSName信息
				OSType:            image.OSType,
				// ExpiredTime:
				// EipAddress:
				// VlanId:
				// OSType:
			}

			instances = append(instances, sinstance)
		}
	}

	return instances, len(instances), nil
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
		if i == 0 {
			var size int64
			deleteOnTermination := true
			size = int64(disk.Size)
			ebs := &ec2.EbsBlockDevice{
				DeleteOnTermination: &deleteOnTermination,
				// The st1 volume type cannot be used for boot volumes. Please use a supported boot volume type: standard,io1,gp2.
				// the encrypted flag cannot be specified since device /dev/sda1 has a snapshot specified.
				// Encrypted:           &disk.Encrypted,
				VolumeSize: &size,
				VolumeType: &disk.Category,
			}

			divceName := fmt.Sprintf("/dev/sda1")
			blockDevice := &ec2.BlockDeviceMapping{
				DeviceName: &divceName,
				Ebs:        ebs,
			}

			blockDevices = append(blockDevices, blockDevice)
		} else {
			var size int64
			size = int64(disk.Size)
			ebs := &ec2.EbsBlockDevice{
				DeleteOnTermination: &disk.DeleteWithInstance,
				Encrypted:           &disk.Encrypted,
				VolumeSize:          &size,
				VolumeType:          &disk.Category,
			}
			// todo: generator device name
			// todo: 这里还需要测试预置硬盘的实例。deviceName是否会冲突。
			deviceName := fmt.Sprintf("/dev/sd%s", string(98+i))
			blockDevice := &ec2.BlockDeviceMapping{
				DeviceName: &deviceName,
				Ebs:        ebs,
			}

			blockDevices = append(blockDevices, blockDevice)
		}
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
		SubnetId:            &SubnetId,
		PrivateIpAddress:    &ipAddr,
		BlockDeviceMappings: blockDevices,
		Placement:           &ec2.Placement{AvailabilityZone: &zoneId},
		SecurityGroupIds:    []*string{&securityGroupId},
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

	res, err := self.ec2Client.RunInstances(&params)
	if err != nil {
		log.Errorf("CreateInstance fail %s", err)
		return "", err
	}

	if len(res.Instances) == 1 {
		return *res.Instances[0].InstanceId, nil
	} else {
		msg := fmt.Sprintf("CreateInstance fail: %s instance created. ", len(res.Instances))
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

	params := &ec2.TerminateInstancesInput{}
	params.SetInstanceIds([]*string{&instanceId})
	_, err := self.ec2Client.TerminateInstances(params)
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

func (self *SRegion) ReplaceSystemDisk(ctx context.Context, instanceId string, imageId string, sysDiskSizeGB int) (string, error) {
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
		if disk.Type == models.DISK_TYPE_SYS {
			rootDisk = &disk
			break
		}
	}

	if rootDisk == nil {
		return "", fmt.Errorf("can not find root disk of instance %s", instanceId)
	}
	log.Debugf("ReplaceSystemDisk replace root disk %s", rootDisk)

	image, err := self.GetImage(imageId)
	if err != nil {
		return "", err
	}

	diskId, err := self.CreateDisk(instance.ZoneId, rootDisk.Category, rootDisk.GetName(), sysDiskSizeGB, image.RootDevice.SnapshotId, "")
	if err != nil {
		return "", err
	}

	self.ec2Client.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{VolumeIds: []*string{&diskId}})
	self.ec2Client.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{InstanceIds: []*string{&instanceId}})

	err = self.DetachDisk(instance.GetId(), rootDisk.DiskId)
	if err != nil {
		log.Debugf("ReplaceSystemDisk detach disk %s: %s", rootDisk.DiskId, err)
		return "", err
	}
	self.ec2Client.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{VolumeIds: []*string{&rootDisk.DiskId}})

	err = self.AttachDisk(instance.GetId(), diskId, rootDisk.Device)
	if err != nil {
		log.Debugf("ReplaceSystemDisk attach disk %s: %s", diskId, err)
		return "", err
	}
	self.ec2Client.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{InstanceIds: []*string{&instanceId}})
	self.ec2Client.WaitUntilVolumeInUse(&ec2.DescribeVolumesInput{VolumeIds: []*string{&diskId}})

	err = self.DeleteDisk(rootDisk.DiskId)
	if err != nil {
		log.Debugf("ReplaceSystemDisk delete old disk %s", rootDisk.DiskId)
	}
	return diskId, nil
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
	return err
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

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotSupported
}
