package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coredns/coredns/plugin/pkg/log"
	"fmt"
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

type InstanceChargeType string

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

type SNetworkInterface struct {
	MacAddress         string
	NetworkInterfaceId string
	PrimaryIpAddress   string
}

type SOperationLocks struct {
	LockReason []string
}

type SSecurityGroupIds struct {
	SecurityGroupId []string
}

type SVpcAttributes struct {
	NatIpAddress     string
	PrivateIpAddress SIpAddress
	VSwitchId        string
	VpcId            string
}

type SInstance struct {
	host *SHost

	RegionId                string
	ZoneId                  string
	InstanceId              string
	ImageId                 string
	SecurityGroupIds        SSecurityGroupIds

	AutoReleaseTime         string
	Cpu                     int8
	CreationTime            time.Time
	DedicatedHostAttribute  SDedicatedHostAttribute
	Description             string
	DeviceAvailable         bool
	EipAddress              SEipAddress
	ExpiredTime             time.Time
	GPUAmount               int
	GPUSpec                 string
	HostName                string
	InnerIpAddress          SIpAddress
	InstanceChargeType      InstanceChargeType
	InstanceName            string
	InstanceNetworkType     string
	InstanceType            string
	InstanceTypeFamily      string
	InternetChargeType      string
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
	RootDeviceName          string
	SerialNumber            string
	SpotPriceLimit          string
	SpotStrategy            string
	StartTime               time.Time
	Status                  string
	StoppedMode             string
	VlanId                  string
}

func (self *SInstance) GetId() string {
	panic("implement me")
}

func (self *SInstance) GetName() string {
	panic("implement me")
}

func (self *SInstance) GetGlobalId() string {
	panic("implement me")
}

func (self *SInstance) GetStatus() string {
	panic("implement me")
}

func (self *SInstance) Refresh() error {
	panic("implement me")
}

func (self *SInstance) IsEmulated() bool {
	panic("implement me")
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SInstance) GetBillingType() string {
	panic("implement me")
}

func (self *SInstance) GetExpiredAt() time.Time {
	panic("implement me")
}

func (self *SInstance) GetCreateTime() time.Time {
	panic("implement me")
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	panic("implement me")
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	panic("implement me")
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	panic("implement me")
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	panic("implement me")
}

func (self *SInstance) GetVcpuCount() int8 {
	panic("implement me")
}

func (self *SInstance) GetVmemSizeMB() int {
	panic("implement me")
}

func (self *SInstance) GetBootOrder() string {
	panic("implement me")
}

func (self *SInstance) GetVga() string {
	panic("implement me")
}

func (self *SInstance) GetVdi() string {
	panic("implement me")
}

func (self *SInstance) GetOSType() string {
	panic("implement me")
}

func (self *SInstance) GetOSName() string {
	panic("implement me")
}

func (self *SInstance) GetBios() string {
	panic("implement me")
}

func (self *SInstance) GetMachine() string {
	panic("implement me")
}

func (self *SInstance) SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) error {
	panic("implement me")
}

func (self *SInstance) GetHypervisor() string {
	panic("implement me")
}

func (self *SInstance) StartVM() error {
	panic("implement me")
}

func (self *SInstance) StopVM(isForce bool) error {
	panic("implement me")
}

func (self *SInstance) DeleteVM() error {
	panic("implement me")
}

func (self *SInstance) UpdateVM(name string) error {
	panic("implement me")
}

func (self *SInstance) RebuildRoot(imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	panic("implement me")
}

func (self *SInstance) DeployVM(name string, password string, publicKey string, deleteKeypair bool, description string) error {
	panic("implement me")
}

func (self *SInstance) ChangeConfig(instanceId string, ncpu int, vmem int) error {
	panic("implement me")
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *SInstance) AttachDisk(diskId string) error {
	panic("implement me")
}

func (self *SInstance) DetachDisk(diskId string) error {
	panic("implement me")
}

func (self *SRegion) GetInstances(zoneId string, ids []string, offset int, limit int) ([]SInstance, int, error) {
	params := &ec2.DescribeInstancesInput{}
	filters := make([]*ec2.Filter, 0)
	if len(zoneId) > 0 {
		name := "availability-zone"
		filters = append(filters, &ec2.Filter{Name: &name, Values: []*string{&zoneId}})
	}

	if len(ids) > 0 {
		_ids := make([]*string, len(ids))
		for _, id := range ids {
			_ids = append(_ids, &id)
		}
		params = params.SetInstanceIds(_ids)
	}

	params = params.SetFilters(filters)
	res, err := self.ec2Client.DescribeInstances(params)
	if err != nil {
		log.Errorf("GetInstances fail %s", err)
		return nil, 0, err
	}

	instances := make([]SInstance, 0)
	for _, reservation := range res.Reservations {
		for _, instance := range reservation.Instances {
			// todo :implement me later
			instances = append(instances, SInstance{
				InstanceId: *instance.InstanceId,
				ImageId: *instance.ImageId,
				InnerIpAddress: SIpAddress{[]string{*instance.PrivateIpAddress}},
				PublicIpAddress: SIpAddress{[]string{*instance.PublicIpAddress}},
			})
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

func (self *SRegion) CreateInstance(name string, imageId string, instanceType string, SubnetId string, securityGroupId string,
	zoneId string, desc string, passwd string, disks []SDisk, ipAddr string,
	keypair string) (string, error) {
		var count int64 = 1
		// disk
		blockDevices := make([]*ec2.BlockDeviceMapping, len(disks))
		for i, disk := range disks{
			if i == 0 {
				var size int64
				size = int64(disk.Size)
				ebs := &ec2.EbsBlockDevice{
					DeleteOnTermination: &disk.DeleteWithInstance,
					SnapshotId: &imageId, // todo: 这里是snapshotid
					Encrypted: &disk.Encrypted,
					VolumeSize: &size,
					VolumeType: &disk.Type,
				}

				// todo: 这里是镜像绑定的deviceName
				divceName := fmt.Sprintf("/dev/sda1")
				blockDevice := &ec2.BlockDeviceMapping{
					DeviceName: &divceName,
					Ebs: ebs,
				}

				blockDevices = append(blockDevices, blockDevice)
			} else {
				var size int64
				size = int64(disk.Size)
				ebs := &ec2.EbsBlockDevice{
					DeleteOnTermination: &disk.DeleteWithInstance,
					Encrypted: &disk.Encrypted,
					VolumeSize: &size,
					VolumeType: &disk.Type,
				}
				// todo: generator device name
				divceName := fmt.Sprintf("/dev/sd%s", string(98+i))
				blockDevice := &ec2.BlockDeviceMapping{
					DeviceName: &divceName,
					Ebs: ebs,
				}

				blockDevices = append(blockDevices, blockDevice)
			}
		}
		// tags
		tags := make([]*ec2.TagSpecification, 2)
		instanceTag := &ec2.TagSpecification{}
		resourceType := "instance"
		instanceName := "Name"
		instanceDesc := "Description"
		instanceTag.ResourceType = &resourceType
		instanceTag.Tags = []*ec2.Tag{
			&ec2.Tag{Key:&instanceName, Value: &name},
			&ec2.Tag{Key:&instanceDesc, Value: &desc},
		}

		params := ec2.RunInstancesInput{
			ImageId: &imageId,
			InstanceType: &instanceType,
			MaxCount: &count,
			MinCount: &count,
			SubnetId: &SubnetId,
			PrivateIpAddress: &ipAddr,
			BlockDeviceMappings: blockDevices,
			KeyName: &keypair,
			Placement: &ec2.Placement{AvailabilityZone: &zoneId},
			SecurityGroupIds: []*string{&securityGroupId},
			TagSpecifications: tags,
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

func (self *SRegion) StartVM(instanceId string) error {
	return nil
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	return nil
}

func (self *SRegion) DeleteVM(instanceId string) error {
	return nil
}

func (self *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	return nil
}

func (self *SRegion) UpdateVM(instanceId string, hostname string) error {
	return nil
}

func (self *SRegion) ReplaceSystemDisk(instanceId string, imageId string, passwd string, keypairName string, sysDiskSizeGB int) (string, error) {
	return "", nil
}

func (self *SRegion) ChangeVMConfig(zoneId string, instanceId string, ncpu int, vmem int, disks []*SDisk) error {
	return nil
}

func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	return nil
}

func (self *SRegion) AttachDisk(instanceId string, diskId string) error {
	return nil
}