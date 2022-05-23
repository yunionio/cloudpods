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

package bingocloud

import (
	"context"
	"fmt"
	"strings"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
)

const (
	InstanceStatusTerminated = "terminated"
)

type SInstance struct {
	multicloud.BingoTags
	multicloud.SInstanceBase
	node *SNode

	ReservationId string `json:"reservationId"`
	OwnerId       string
	GroupSet      []struct {
		GroupId   string
		GroupName string
	}
	InstancesSet struct {
		InstanceId    string `json:"instanceId"`
		InstanceName  string `json:"instanceName"`
		HostName      string `json:"hostName"`
		ImageId       string `json:"imageId"`
		InstanceState struct {
			Code            int    `json:"code"`
			Name            string `json:"name"`
			PendingProgress string `json:"pendingProgress"`
		} `json:"instanceState"`
		PrivateDNSName     string `json:"privateDnsName"`
		DNSName            string `json:"dnsName"`
		PrivateIPAddress   string `json:"privateIpAddress"`
		PrivateIPAddresses string `json:"privateIpAddresses"`
		IPAddress          string `json:"ipAddress"`
		NifInfo            string `json:"nifInfo"`
		KeyName            string `json:"keyName"`
		AmiLaunchIndex     int    `json:"amiLaunchIndex"`
		ProductCodesSet    []struct {
			ProductCode string `json:"productCode"`
		} `json:"productCodesSet"`
		InstanceType   string    `json:"instanceType"`
		VmtypeCPU      int       `json:"vmtype_cpu"`
		VmtypeMem      int       `json:"vmtype_mem"`
		VmtypeDisk     int       `json:"vmtype_disk"`
		VmtypeGpu      int       `json:"vmtype_gpu"`
		VmtypeSsd      int       `json:"vmtype_ssd"`
		VmtypeHdd      int       `json:"vmtype_hdd"`
		VmtypeHba      int       `json:"vmtype_hba"`
		VmtypeSriov    int       `json:"vmtype_sriov"`
		LaunchTime     time.Time `json:"launchTime"`
		RootDeviceType string    `json:"rootDeviceType"`
		HostAddress    string    `json:"hostAddress"`
		Platform       string    `json:"platform"`
		UseCompactMode bool      `json:"useCompactMode"`
		ExtendDisk     bool      `json:"extendDisk"`
		Placement      struct {
			AvailabilityZone string `json:"availabilityZone"`
		} `json:"placement"`
		Namespace    string `json:"namespace"`
		KernelId     string `json:"kernelId"`
		RamdiskId    string `json:"ramdiskId"`
		OperName     string `json:"operName"`
		OperProgress int    `json:"operProgress"`
		Features     string `json:"features"`
		Monitoring   struct {
			State string `json:"state"`
		} `json:"monitoring"`
		SubnetId              string    `json:"subnetId"`
		VpcId                 string    `json:"vpcId"`
		StorageId             string    `json:"storageId"`
		DisableAPITermination bool      `json:"disableApiTermination"`
		Vncdisabled           bool      `json:"vncdisabled"`
		StartTime             time.Time `json:"startTime"`
		CustomStatus          string    `json:"customStatus"`
		SystemStatus          int       `json:"systemStatus"`
		NetworkStatus         int       `json:"networkStatus"`
		ScheduleTags          string    `json:"scheduleTags"`
		StorageScheduleTags   string    `json:"storageScheduleTags"`
		IsEncrypt             bool      `json:"isEncrypt"`
		IsImported            bool      `json:"isImported"`
		Ec2Version            string    `json:"ec2Version"`
		Passphrase            string    `json:"passphrase"`
		DrsEnabled            bool      `json:"drs_enabled"`
		LaunchPriority        int       `json:"launchPriority"`
		CPUPriority           int       `json:"cpuPriority"`
		MemPriority           int       `json:"memPriority"`
		CPUQuota              int       `json:"cpuQuota"`
		AutoMigrate           bool      `json:"autoMigrate"`
		DrMirrorId            string    `json:"drMirrorId"`
		BlockDeviceMapping    []struct {
			DeviceName string `json:"deviceName"`
			Ebs        struct {
				AttachTime          time.Time `json:"attachTime"`
				DeleteOnTermination bool      `json:"deleteOnTermination"`
				Status              string    `json:"status"`
				VolumeId            string    `json:"volumeId"`
				Size                int       `json:"size"`
			} `json:"ebs"`
		} `json:"blockDeviceMapping"`
		EnableLiveScaleup bool   `json:"enableLiveScaleup"`
		ImageBytes        int64  `json:"imageBytes"`
		StatusReason      string `json:"statusReason"`
		Hypervisor        string `json:"hypervisor"`
		Bootloader        string `json:"bootloader"`
		BmMachineId       string `json:"bmMachineId"`
	}
}

func (self *SInstance) GetId() string {
	return self.InstancesSet.InstanceId
}

func (self *SInstance) GetGlobalId() string {
	return self.GetId()
}

func (self *SInstance) GetName() string {
	if len(self.InstancesSet.InstanceName) > 0 {
		return self.InstancesSet.InstanceName
	}
	return self.GetId()
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	for _, sec := range self.GroupSet {
		ret = append(ret, sec.GroupId)
	}
	return ret, nil
}

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return nil
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return self.node.cluster.region.AttachDisk(self.GetId(), diskId)
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return self.node.cluster.region.DetachDisk(self.GetId(), diskId)
}

//调整配置
func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	instanceTypes := []string{}

	if len(config.InstanceType) > 0 {
		instanceTypes = []string{config.InstanceType}
	} else {
		specs, err := self.node.cluster.region.GetMatchInstanceTypes(config.Cpu, config.MemoryMB, 0, self.node.ClusterId)
		if err != nil {
			return err
		}
		for _, spec := range specs {
			instanceTypes = append(instanceTypes, spec.InstanceType)
		}
	}

	var err error
	for _, instanceType := range instanceTypes {
		err = self.node.cluster.region.ChangeVMConfig(self.GetId(), instanceType)
		if err != nil {
			log.Warningf("ChangeVMConfig %s for %s error: %v", self.GetId(), instanceType, err)
		} else {
			return cloudprovider.WaitStatusWithDelay(self, api.VM_READY, 15*time.Second, 15*time.Second, 180*time.Second)
		}
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("Failed to change vm config, specification not supported")
}

//todo:修改实例属性
func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	var keypairName string
	return self.node.cluster.region.DeployVM(self.GetId(), name, password, keypairName, deleteKeypair, description)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	err := self.node.cluster.region.DeleteVM(self.GetId())
	if err != nil {
		return err
	}
	return cloudprovider.WaitDeleted(self, 10*time.Second, 300*time.Second) // 5minutes
}

func (self *SInstance) GetVcpuCount() int {
	return self.InstancesSet.VmtypeCPU
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.InstancesSet.VmtypeMem
}

func (self *SInstance) GetBootOrder() string {
	return "cdn"
}

func (self *SInstance) GetVga() string {
	return ""
}

func (self *SInstance) GetVdi() string {
	return ""
}

func (self *SInstance) GetOSArch() string {
	return "x86"
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	if self.InstancesSet.Platform == "linux" {
		return cloudprovider.OsTypeLinux
	}
	return cloudprovider.OsTypeWindows
}

func (self *SInstance) GetOSName() string {
	return ""
}

func (self *SInstance) GetBios() string {
	return strings.ToUpper(self.InstancesSet.Bootloader)
}

func (self *SInstance) GetMachine() string {
	return ""
}

func (self *SInstance) GetInstanceType() string {
	return self.InstancesSet.InstanceType
}

func (self *SInstance) GetError() error {
	return nil
}

func (self *SInstance) GetHostname() string {
	return self.InstancesSet.HostName
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.node
}

func (self *SInstance) GetIHostId() string {
	return self.InstancesSet.HostAddress
	//info := strings.Split(self.InstancesSet.HostAddress, "@")
	//if len(info) == 2 {
	//	return info[1]
	//}
	//return ""
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_BINGO_CLOUD
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	storages, err := self.node.cluster.region.getStorages()
	if err != nil {
		return nil, err
	}
	storageMaps := map[string]SStorage{}
	for i := range storages {
		storageMaps[storages[i].StorageId] = storages[i]
	}
	ret := []cloudprovider.ICloudDisk{}
	for _, _disk := range self.InstancesSet.BlockDeviceMapping {
		disk, err := self.node.cluster.region.GetDisk(_disk.Ebs.VolumeId)
		if err != nil {
			return nil, err
		}
		storage, ok := storageMaps[disk.StorageId]
		if ok {
			storage.cluster = self.node.cluster
			disk.storage = &storage
			ret = append(ret, disk)
		}
	}
	return ret, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics, err := self.node.cluster.region.GetInstanceNics(self.InstancesSet.InstanceId)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNic{}
	for i := range nics {
		ret = append(ret, &nics[i])
	}
	return ret, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eips, _, err := self.node.cluster.region.GetEips("", self.InstancesSet.InstanceId, "")
	if err != nil {
		return nil, err
	}
	for i := range eips {
		eips[i].region = self.node.cluster.region
		return &eips[i], nil
	}
	return nil, nil
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetStatus() string {
	switch self.InstancesSet.InstanceState.Name {
	case "stopped":
		return api.VM_READY
	default:
		return self.InstancesSet.InstanceState.Name
	}
}

func (self *SInstance) Refresh() error {
	newNode, err := self.node.cluster.region.GetInstance(self.GetId())
	if err != nil {
		return err
	}
	if newNode.InstancesSet.InstanceState.Name == InstanceStatusTerminated {
		return cloudprovider.ErrNotFound
	}
	newNode.node = self.node
	err = jsonutils.Update(self, newNode)
	if err != nil {
		return err
	}
	return nil
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return self.node.cluster.region.RebuildRoot(self.GetId(), self.GetInstanceType(), config.ImageId, config.Password, config.PublicKey, config.SysSizeGB)
	//return "", cloudprovider.ErrNotImplemented
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
			err := self.node.cluster.region.StartVM(self.GetId())
			if err != nil {
				return err
			}
		}
		time.Sleep(interval)
	}
	return cloudprovider.ErrTimeout
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := self.node.cluster.region.StopVM(self.GetId(), opts.IsForce, opts.StopCharging)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) UpdateUserData(userData string) error {
	return self.node.cluster.region.updateUserData(self.GetId(), userData)
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return self.node.cluster.region.updateVM(self.GetId(), name)
}

func (self *SRegion) GetInstances(id, nodeId string, maxResult int, nextToken string) ([]SInstance, string, error) {
	params := map[string]string{}
	if maxResult > 0 {
		params["maxRecords"] = fmt.Sprintf("%d", maxResult)
	}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}

	idx := 1
	if len(nodeId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "node-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = nodeId
		idx++
	}

	if len(id) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "instance-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = id
		idx++
	}

	resp, err := self.invoke("DescribeInstances", params)
	if err != nil {
		return nil, "", err
	}
	if !resp.Contains("reservationSet") {
		return nil, "", nil
	}
	if item, _ := resp.Get("reservationSet"); item.IsZero() {
		return nil, "", nil
	}
	var result struct {
		NextToken      string
		ReservationSet []SInstance
	}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, "", err
	}
	return result.ReservationSet, result.NextToken, nil
}

func (self *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, _, err := self.GetInstances(instanceId, "", 1, "")
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &instances[0], nil
}

func (self *SRegion) StartVM(instanceId string) error {
	status, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status on StartVM: %s", err)
		return err
	}
	if status != "stopped" {
		log.Errorf("StartVM: vm status is %s expect %s", status, "stopped")
		return cloudprovider.ErrInvalidStatus
	}
	return self.doStartVM(instanceId)
}

func (self *SRegion) doStartVM(id string) error {
	params := map[string]string{}
	if len(id) > 0 {
		params["InstanceId.1"] = id
	}
	_, err := self.invoke("StartInstances", params)
	return err
}

func (self *SRegion) StopVM(instanceId string, isForce, stopCharging bool) error {
	status, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status on StopVM: %s", err)
		return err
	}
	if status == "stopped" {
		return nil
	}
	if status != "running" {
		log.Errorf("StopVM: vm status is %s expect %s", status, "running")
		return cloudprovider.ErrInvalidStatus
	}
	return self.doStopVM(instanceId, isForce)
}

func (self *SRegion) doStopVM(id string, isForce bool) error {
	params := map[string]string{}
	if len(id) > 0 {
		params["InstanceId.1"] = id
	}
	if isForce {
		params["Force"] = "true"
	} else {
		params["Force"] = "false"
	}
	_, err := self.invoke("StopInstances", params)
	return err
}

func (self *SRegion) DeleteVM(id string) error {
	param := make(map[string]string)
	param["InstanceId.1"] = id
	_, err := self.invoke("TerminateInstances", param)
	return err
}

func (self *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return err
	}

	params := make(map[string]string)

	// if resetPassword {
	//	params["Password"] = seclib2.RandomPassword2(12)
	// }
	if len(name) > 0 && instance.GetName() != name {
		params["InstanceName"] = name
	}

	if len(params) > 0 {
		return self.modifyInstanceAttribute(instanceId, params)
	} else {
		return nil
	}
}

func (self *SRegion) modifyInstanceAttribute(instanceId string, extra map[string]string) error {
	params := map[string]string{
		"InstanceId": instanceId,
	}
	if extra != nil && len(extra) > 0 {
		for k, v := range extra {
			params["Attribute"] = k
			params["Value"] = v
		}
	}
	_, err := self.invoke("ModifyInstanceAttribute", params)
	return err
}

func (self *SRegion) AttachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["VolumeId"] = diskId
	params["Device"] = "/dev/vda"
	_, err := self.invoke("AttachVolume", params)
	if err != nil {
		log.Errorf("AttachDisk %s to %s fail %s", diskId, instanceId, err)
		return err
	}

	return nil
}

func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	params["VolumeId"] = diskId
	log.Infof("Detach instance %s disk %s", instanceId, diskId)
	_, err := self.invoke("DetachVolume", params)
	return err
}

func (self *SRegion) ChangeVMConfig(instanceId string, instanceType string) error {
	param := make(map[string]string)
	param["instanceType"] = instanceType
	return self.modifyInstanceAttribute(instanceId, param)
}

func (self *SRegion) updateUserData(instanceId string, userData string) error {
	param := make(map[string]string)
	if len(userData) > 0 {
		param["userData"] = userData
	}
	return self.modifyInstanceAttribute(instanceId, param)
}

func (self *SRegion) updateVM(instanceId string, name string) error {
	params := make(map[string]string)
	if len(name) > 0 {
		params["InstanceName"] = name
	}
	return self.modifyInstanceAttribute(instanceId, params)
}

// CreateInstance
//{
//    "Tag.1.Key":"测试",
//    "InstanceName":"我的 Demo 实例",
//    "VpcId":"vpc-D37DF705",
//    "OwnerId":"zhouqifeng",
//    "Version":"2011-11-01",
//    "Tag.1.Value":"test",
//    "ImageId":"ami-03F1FC93",
//    "InstanceType":"m1.small",
//    "SecurityGroup.1":"default"
//}
func (self *SRegion) CreateInstance(name string, nodeId string, img *SImage, instanceType string, SubnetId string,
	securityGroupId string, vpcId string, zoneId string, desc string, disks []SDisk, ipAddr string,
	keypair string, publicKey string, passwd string, userData string, bc *billing.SBillingCycle, projectId string, tags map[string]string) (string, error) {
	params := make(map[string]string)
	params["ImageId"] = img.ImageId
	params["MinCount"] = "1"
	params["MaxCount"] = "1"
	params["SecurityGroup.1"] = securityGroupId
	params["InstanceType"] = instanceType
	params["VpcId"] = vpcId
	params["SubnetId"] = SubnetId
	params["RootDevicePersistent"] = "true"
	params["InstanceName"] = name
	if len(nodeId) > 0 {
		params["AllowNodes.1"] = nodeId
	}
	if len(keypair) > 0 {
		params["KeyName"] = keypair
	}
	if len(passwd) > 0 {
		params["Password"] = passwd
	}
	if len(userData) > 0 {
		params["UserData"] = userData
	}
	for i := range disks {
		var deviceName string
		if i == 0 {
			params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.DeleteOnTermination", i+1)] = "true"
			deviceName = fmt.Sprintf("/dev/vda")
			params["StorageId"] = disks[i].StorageId
		} else {
			deviceName = "/dev/vdb"
		}
		params[fmt.Sprintf("BlockDeviceMapping.%d.DeviceName", i+1)] = deviceName
		params[fmt.Sprintf("BlockDeviceMapping.%d.Ebs.VolumeSize", i+1)] = fmt.Sprintf("%d", disks[i].Size)
	}

	ret, err := self.invoke("RunInstances", params)
	if err != nil {
		return "", err
	}
	id, err := ret.Get("instancesSet", "instanceId")
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (self *SRegion) RebuildRoot(instanceId string, instanceType string, imageId string, passwd string, keypairName string, sysDiskSizeGB int) (string, error) {
	params := make(map[string]string)
	params["instanceId"] = instanceId
	params["instanceType"] = instanceType
	params["ImageId"] = imageId
	if len(keypairName) > 0 {
		params["KeyName"] = keypairName
	}
	_, err := self.invoke("ReinstallInstance", params)
	if err != nil {
		return "", err
	}
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	if len(instance.InstancesSet.BlockDeviceMapping) <= 0 {
		return "", nil
	}
	diskSizeMB := instance.InstancesSet.BlockDeviceMapping[0].Ebs.Size
	diskId := instance.InstancesSet.BlockDeviceMapping[0].Ebs.VolumeId
	if sysDiskSizeGB*1024 > diskSizeMB {
		return diskId, self.resizeDisk(diskId, int64(sysDiskSizeGB)*1024)
	}
	return diskId, nil
}
