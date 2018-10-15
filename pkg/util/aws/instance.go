package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coredns/coredns/plugin/pkg/log"
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

// {"NatIpAddress":"","PrivateIpAddress":{"IpAddress":["192.168.220.214"]},"VSwitchId":"vsw-2ze9cqwza4upoyujq1thd","VpcId":"vpc-2zer4jy8ix3i8f0coc5uw"}

type SVpcAttributes struct {
	NatIpAddress     string
	PrivateIpAddress SIpAddress
	VSwitchId        string
	VpcId            string
}

type SInstance struct {
	host *SHost

	// idisks []cloudprovider.ICloudDisk

	AutoReleaseTime         string
	ClusterId               string
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
	ImageId                 string
	InnerIpAddress          SIpAddress
	InstanceChargeType      InstanceChargeType
	InstanceId              string
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

func (self *SRegion) CreateInstance(name string, imageId string, instanceType string, securityGroupId string,
	zoneId string, desc string, passwd string, disks []SDisk, vSwitchId string, ipAddr string,
	keypair string) (string, error) {
		return "", nil
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