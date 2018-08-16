package aliyun

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
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

type SEipAddress struct {
	AllocationId       string
	InternetChargeType string
	IpAddress          string
}

func (self *SEipAddress) GetIP() string {
	return self.IpAddress
}

func (self *SEipAddress) GetAllocationId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetChargeType() string {
	return self.GetChargeType()
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

	idisks []cloudprovider.ICloudDisk

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
	params := make(map[string]string)
	params["RegionId"] = self.RegionId

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

func (self *SInstance) GetCreateTime() time.Time {
	return self.CreationTime
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
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

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) getVpc() (*SVpc, error) {
	return self.host.zone.region.getVpc(self.VpcAttributes.VpcId)
}

func (self *SInstance) fetchDisks() error {
	disks, total, err := self.host.zone.region.GetDisks(self.InstanceId, "", "", nil, 0, 50)
	if err != nil {
		log.Errorf("fetchDisks fail %s", err)
		return err
	}
	if total > len(disks) {
		disks, _, err = self.host.zone.region.GetDisks(self.InstanceId, "", "", nil, 0, total)
	}
	self.idisks = make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		store, err := self.host.zone.getStorageByCategory(disks[i].Category)
		if err != nil {
			return err
		}
		disks[i].storage = store
		self.idisks[i] = &disks[i]
	}
	return nil
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	if self.idisks == nil {
		err := self.fetchDisks()
		if err != nil {
			return nil, err
		}
	}
	return self.idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := make([]cloudprovider.ICloudNic, 0)
	for _, ip := range self.VpcAttributes.PrivateIpAddress.IpAddress {
		nic := SInstanceNic{instance: self, ipAddr: ip}
		nics = append(nics, &nic)
	}
	return nics, nil
}

func (self *SInstance) GetEIP() cloudprovider.ICloudEIP {
	return &self.EipAddress
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

func (self *SInstance) GetStatus() string {
	// Running：运行中
	//Starting：启动中
	//Stopping：停止中
	//Stopped：已停止
	switch self.Status {
	case InstanceStatusRunning:
		return models.VM_RUNNING
	case InstanceStatusStarting:
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
	return models.HYPERVISOR_ALIYUN
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
	params["HostName"] = name
	if len(passwd) > 0 {
		params["Password"] = passwd
	} else {
		params["PasswordInherit"] = "True"
	}
	params["IoOptimized"] = "optimized"
	for i, d := range disks {
		if i == 0 {
			params["SystemDisk.Category"] = d.Category
			params["SystemDisk.Size"] = fmt.Sprintf("%d", d.Size)
			params["SystemDisk.DiskName"] = d.GetName()
			params["SystemDisk.Description"] = d.Description
		} else {
			params[fmt.Sprintf("DataDisk.%d.Size", i)] = fmt.Sprintf("%d", d.Size)
			params[fmt.Sprintf("DataDisk.%d.Category", i)] = d.Category
			params[fmt.Sprintf("DataDisk.%d.DiskName", i)] = d.GetName()
			params[fmt.Sprintf("DataDisk.%d.Description", i)] = d.Description
			params[fmt.Sprintf("DataDisk.%d.Encrypted", i)] = "false"
		}
	}
	params["VSwitchId"] = vSwitchId
	params["PrivateIpAddress"] = ipAddr
	params["InstanceChargeType"] = "PostPaid"
	params["SpotStrategy"] = "NoSpot"
	if len(keypair) > 0 {
		params["KeyPairName"] = keypair
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
	return self.instanceOperation(instanceId, "DeleteInstance", nil)
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
	status, _ := self.GetInstanceStatus(instanceId)
	if status != InstanceStatusStopped {
		return cloudprovider.ErrInvalidStatus
	}
	return self.doStartVM(instanceId)
	// if err != nil {
	//	return err
	// }
	// return self.waitInstanceStatus(instanceId, InstanceStatusRunning, time.Second*5, time.Second*180) // 3 minutes to timeout
}

func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	status, _ := self.GetInstanceStatus(instanceId)
	if status != InstanceStatusRunning {
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
	if status == InstanceStatusRunning {
		err = self.StopVM(instanceId, true)
		if err != nil {
			return err
		}
	} else if status != InstanceStatusStopped {
		return cloudprovider.ErrInvalidStatus
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

func (self *SInstance) StartVM() error {
	err := self.host.zone.region.StartVM(self.InstanceId)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_RUNNING, 5*time.Second, 180*time.Second) // 3minutes
}

func (self *SInstance) StopVM(isForce bool) error {
	err := self.host.zone.region.StopVM(self.InstanceId, isForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) DeleteVM() error {
	err := self.host.zone.region.DeleteVM(self.InstanceId)
	if err != nil {
		return err
	}
	return cloudprovider.WaitDeleted(self, 10*time.Second, 300*time.Second) // 5minutes
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
	ret.Add(jsonutils.NewString("aliyun"), "protocol")
	ret.Add(jsonutils.NewString(self.InstanceId), "instance_id")
	return ret, nil
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
	} else if secgrpId, err := vpc.syncSecurityGroup(secgroupId, name, rules); err != nil {
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

func (self *SInstance) UpdateVM(name string) error {
	return fmt.Errorf("not implement")
}

func (self *SInstance) RebuildRoot(imageId string) error {
	return fmt.Errorf("not implement")
}

func (self *SInstance) DeployVM(resetPassword bool, keypair string, deleteKeypair bool) error {
	return fmt.Errorf("not implement")
}

func (self *SInstance) ChangeConfig(instanceId string,ncpu int, vmem int) error {
	return fmt.Errorf("not implement")
}

func (self *SInstance) AttachDisk(diskId string) error {
	return fmt.Errorf("not implement")
}