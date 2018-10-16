package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coredns/coredns/plugin/pkg/log"
	"fmt"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/osprofile"
)

const (
	InstanceStatusPending = "pending"
	InstanceStatusRunning = "running"
	InstanceStatusShutting = "shutting-down"
	InstanceStatusTerminated = "terminated"
	InstanceStatusStopping = "stopping"
	InstanceStatusStopped = "stopped"
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
	host *SHost
	RegionId                string
	ZoneId                  string
	InstanceId              string
	ImageId                 string

	HostName                string
	InstanceName            string
	InstanceType            string
	Cpu                     int8
	Memory                  int
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
	NetworkInterfaces       SNetworkInterfaces
	EipAddress              SEipAddress
	Disks                   []string
	OSName                  string
	OSType                  string
	Description             string

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

func (self *SInstance) GetId() string {
	return self.InstanceId
}

func (self *SInstance) GetName() string {
	return self.HostName
}

func (self *SInstance) GetGlobalId() string {
	return self.InstanceId
}

func (self *SInstance) GetStatus() string {
	// todo : implement me
	switch self.Status {
	case InstanceStatusRunning:
		return models.VM_RUNNING
	case InstanceStatusPending:   // todo: pending ?
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

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	// todo: add price_key here

	if len(self.ImageId) > 0 {
		if image, err := self.host.zone.region.GetImage(self.ImageId); err != nil {
			log.Errorf("Failed to find image %s for instance %s", self.ImageId, self.GetName())
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
	panic("implement me")
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
	// todo: implement me
	return nil, nil
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

func (self *SInstance) SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) error {
	if vpc, err := self.getVpc(); err != nil {
		return err
	} else if len(secgroupId) == 0 {
		for index, secgrpId := range self.SecurityGroupIds.SecurityGroupId {
			if err := vpc.revokeSecurityGroup(secgrpId, self.InstanceId, index == 0); err != nil {
				return err
			}
		}
	} else if secgrpId, err := vpc.SyncSecurityGroup(secgroupId, name, rules); err != nil {
		return err
	} else if err := vpc.assignSecurityGroup(secgrpId, self.InstanceId); err != nil {
		return err
	} else {
		for _, secgroupId := range self.SecurityGroupIds.SecurityGroupId {
			if secgroupId != secgrpId {
				if err := vpc.revokeSecurityGroup(secgroupId, self.InstanceId, false); err != nil {
					return err
				}
			}
		}
		self.SecurityGroupIds.SecurityGroupId = []string{secgrpId}
	}
	return nil
}

func (self *SInstance) GetHypervisor() string {
	return models.HYPERVISOR_AWS
}

func (self *SInstance) StartVM() error {
	timeout := 300 * time.Second
	interval := 15 * time.Second

	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := self.Refresh()
		if err != nil {
			return err
		}
		log.Debugf("status %s expect %s", self.GetStatus(), models.VM_RUNNING)
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

func (self *SInstance) StopVM(isForce bool) error {
	err := self.host.zone.region.StopVM(self.InstanceId, isForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) DeleteVM() error {
	for {
		err := self.host.zone.region.DeleteVM(self.InstanceId)
		if err != nil {
			// todo: implement me
		} else {
			break
		}
	}
	return cloudprovider.WaitDeleted(self, 10*time.Second, 300*time.Second) // 5minutes

}

func (self *SInstance) UpdateVM(name string) error {
	// todo :implement me
	return nil
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
	return self.host.zone.region.AttachDisk(self.InstanceId, diskId)
}

func (self *SInstance) DetachDisk(diskId string) error {
	return self.host.zone.region.DetachDisk(self.InstanceId, diskId)
}

func (self *SInstance) getVpc() (*SVpc, error) {
	return self.host.zone.region.getVpc(self.VpcAttributes.VpcId)
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