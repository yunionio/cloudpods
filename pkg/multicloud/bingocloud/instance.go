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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
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

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
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
	result := struct {
		NextToken      string
		ReservationSet []SInstance
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, "", err
	}
	return result.ReservationSet, result.NextToken, nil
}
