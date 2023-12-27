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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SInstance struct {
	BingoTags
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
	var ret []string
	for _, sec := range self.GroupSet {
		ret = append(ret, sec.GroupId)
	}
	return ret, nil
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return self.node.cluster.region.modifyInstanceAttribute(self.InstancesSet.InstanceId, map[string]string{"GroupId": secgroupIds[0]})
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return nil
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return nil
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	attrs := make(map[string]string)
	if opts.Password != "" {
		attrs["InstanceAction"] = "ResetPassword"
	}
	return self.node.cluster.region.modifyInstanceAttribute(self.InstancesSet.InstanceId, attrs)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	params := map[string]string{}
	params["InstanceId.1"] = self.InstancesSet.InstanceId

	_, err := self.node.cluster.region.invoke("TerminateInstances", params)
	return err
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

func (self *SInstance) GetOsArch() string {
	return apis.OS_ARCH_X86_64
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	if self.InstancesSet.Platform == "linux" {
		return cloudprovider.OsTypeLinux
	}
	return cloudprovider.OsTypeWindows
}

func (self *SInstance) GetFullOsName() string {
	return ""
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(self.InstancesSet.Bootloader)
}

func (i *SInstance) GetOsDist() string {
	return ""
}

func (i *SInstance) GetOsVersion() string {
	return ""
}

func (i *SInstance) GetOsLang() string {
	return ""
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
	info := strings.Split(self.InstancesSet.HostAddress, "@")
	if len(info) == 2 {
		return info[1]
	}
	return ""
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
	var ret []cloudprovider.ICloudDisk
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
	var ret []cloudprovider.ICloudNic
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

func (self *SInstance) Refresh() error {
	newInstances, _, err := self.node.cluster.region.GetInstances(self.InstancesSet.InstanceId, self.node.NodeId, MAX_RESULT, "")
	if err != nil {
		return err
	}
	if len(newInstances) == 1 {
		return jsonutils.Update(self, &newInstances[0])
	}
	return cloudprovider.ErrNotFound
}

func (self *SInstance) GetStatus() string {
	switch self.InstancesSet.InstanceState.Name {
	case "stopped":
		return api.VM_READY
	default:
		return self.InstancesSet.InstanceState.Name
	}
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	params := map[string]string{}
	params["InstanceId"] = self.InstancesSet.InstanceId

	resp, err := self.node.cluster.region.invoke("GetVncInfo", params)
	if err != nil {
		return nil, err
	}

	result := struct {
		GetVncInfoResult *cloudprovider.ServerVncOutput `json:"getVncInfoResult"`
	}{}
	_ = resp.Unmarshal(&result)

	result.GetVncInfoResult.InstanceId = self.InstancesSet.InstanceId
	result.GetVncInfoResult.Hypervisor = self.GetHypervisor()

	return result.GetVncInfoResult, nil
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	params := map[string]string{}
	params["InstanceId"] = self.InstancesSet.InstanceId
	params["ImageId"] = config.ImageId
	params["InstanceType"] = self.InstancesSet.InstanceType
	if config.PublicKey != "" {
		params["KeyName"] = config.PublicKey
	}

	isOk := "false"
	result, err := self.node.cluster.region.invoke("ReinstallInstance", params)
	if err != nil {
		return "", err
	}
	_ = result.Unmarshal(&isOk, "return")
	if isOk != "true" {
		return "", errors.Wrap(cloudprovider.ErrUnknown, "RebuildRoot")
	}

	iDisks, err := self.GetIDisks()
	if err != nil {
		return "", err
	}
	if len(iDisks) > 0 {
		return iDisks[0].GetGlobalId(), nil
	}

	return "", errors.Wrap(cloudprovider.ErrUnknown, "RebuildRoot")
}

func (self *SInstance) StartVM(ctx context.Context) error {
	params := map[string]string{}
	params["InstanceId.1"] = self.InstancesSet.InstanceId
	_, err := self.node.cluster.region.invoke("StartInstances", params)
	return err
}

func (self *SInstance) SuspendVM(ctx context.Context) error {
	params := map[string]string{}
	params["InstanceId.1"] = self.InstancesSet.InstanceId
	_, err := self.node.cluster.region.invoke("SuspendInstances", params)
	return err
}

func (self *SInstance) ResumeVM(ctx context.Context) error {
	params := map[string]string{}
	params["InstanceId.1"] = self.InstancesSet.InstanceId
	_, err := self.node.cluster.region.invoke("ResumeInstances", params)
	return err
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	params := map[string]string{}
	params["InstanceId.1"] = self.InstancesSet.InstanceId
	_, err := self.node.cluster.region.invoke("StopInstances", params)
	return err
}

func (self *SInstance) UpdateUserData(userData string) error {
	return self.node.cluster.region.modifyInstanceAttribute(self.InstancesSet.InstanceId, map[string]string{"UserData": userData})
}

func (self *SInstance) UpdateInstanceType(instanceType string) error {
	return self.node.cluster.region.modifyInstanceAttribute(self.InstancesSet.InstanceId, map[string]string{"InstanceType": instanceType})
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return self.node.cluster.region.modifyInstanceAttribute(self.InstancesSet.InstanceId, map[string]string{"InstanceName": input.NAME})
}

func (self *SInstance) CreateInstanceSnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudInstanceSnapshot, error) {
	newId, err := self.node.cluster.region.createInstanceSnapshot(self.InstancesSet.InstanceId, name, desc)
	if err != nil {
		return nil, err
	}
	return self.GetInstanceSnapshot(newId)
}

func (self *SInstance) GetInstanceSnapshot(id string) (cloudprovider.ICloudInstanceSnapshot, error) {
	snapshots, err := self.node.cluster.region.getInstanceSnapshots(self.InstancesSet.InstanceId, id)
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		if snapshots[i].GetGlobalId() == id {
			snapshots[i].region = self.node.cluster.region
			return &snapshots[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SInstance) GetInstanceSnapshots() ([]cloudprovider.ICloudInstanceSnapshot, error) {
	snapshots, err := self.node.cluster.region.getInstanceSnapshots(self.InstancesSet.InstanceId, "")
	if err != nil {
		return nil, err
	}
	var ret []cloudprovider.ICloudInstanceSnapshot
	for i := range snapshots {
		snapshots[i].region = self.node.cluster.region
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (self *SInstance) ResetToInstanceSnapshot(ctx context.Context, idStr string) error {
	return self.node.cluster.region.revertInstanceSnapshot(idStr)
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vms, _, err := self.GetInstances(id, "", 1, "")
	if err != nil {
		return nil, err
	}
	for i := range vms {
		if vms[i].GetGlobalId() == id {
			return &vms[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetInstances(id, nodeId string, maxResult int, nextToken string) ([]SInstance, string, error) {
	params := map[string]string{}
	if maxResult > 0 {
		params["MaxRecords"] = fmt.Sprintf("%d", maxResult)
	}
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
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
	result := struct {
		NextToken      string
		ReservationSet []SInstance
	}{}
	_ = resp.Unmarshal(&result)

	return result.ReservationSet, result.NextToken, nil
}

func (self *SRegion) modifyInstanceAttribute(instanceId string, attrs map[string]string) error {
	params := map[string]string{}
	params["InstanceId"] = instanceId
	for key, value := range attrs {
		params["Attribute"] = key
		params["Value"] = value
		_, err := self.client.invoke("ModifyInstanceAttribute", params)
		if err != nil {
			return err
		}
	}
	return nil
}
