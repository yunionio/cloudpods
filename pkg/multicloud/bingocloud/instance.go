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
	"time"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SInstance struct {
	ReservationId string `json:"reservationId"`
	OwnerId       string
	GroupSet      struct {
		Item struct {
			GroupId   string
			GroupName string
		}
	}
	InstancesSet struct {
		Item struct {
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
			ProductCodesSet    struct {
				Item struct {
					ProductCode string `json:"productCode"`
				} `json:"item"`
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
			BlockDeviceMapping    struct {
				Item struct {
					DeviceName string `json:"deviceName"`
					Ebs        struct {
						AttachTime          time.Time `json:"attachTime"`
						DeleteOnTermination bool      `json:"deleteOnTermination"`
						Status              string    `json:"status"`
						VolumeId            string    `json:"volumeId"`
						Size                int       `json:"size"`
					} `json:"ebs"`
				} `json:"item"`
			} `json:"blockDeviceMapping"`
			EnableLiveScaleup bool   `json:"enableLiveScaleup"`
			ImageBytes        int64  `json:"imageBytes"`
			StatusReason      string `json:"statusReason"`
			Hypervisor        string `json:"hypervisor"`
			Bootloader        string `json:"bootloader"`
			BmMachineId       string `json:"bmMachineId"`
		}
	}
}

func (self *SRegion) DescribeInstances(id string, maxResult int, nextToken string) ([]SInstance, string, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["instanceId"] = id
	}
	if maxResult > 0 {
		params["maxRecords"] = fmt.Sprintf("%d", maxResult)
	}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	resp, err := self.invoke("DescribeInstances", params)
	// resp, err := self.invoke("DescribeInstanceHosts", params)

	if err != nil {
		return nil, "", err
	}
	result := struct {
		NextToken      string
		ReservationSet struct {
			Item []SInstance
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, "", err
	}
	return result.ReservationSet.Item, result.NextToken, nil
}

func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	vm := &SInstance{}
	return vm, cloudprovider.ErrNotImplemented
}

//	IBillingResource
func (self *SInstance) GetBillingType() string {
	return ""
}

func (self *SInstance) GetCreatedAt() time.Time {
	return time.Now()
}

func (self *SInstance) GetExpiredAt() time.Time {
	return time.Now()
}

func (self *SInstance) SetAutoRenew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotFound
}
func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) IsAutoRenew() bool {
	return false
}

//	IVirtualResource
func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetId() string {
	return ""
}

func (self *SInstance) GetName() string {
	return ""
}

func (self *SInstance) GetGlobalId() string {
	return ""
}

func (self *SInstance) GetStatus() string {
	return ""
}

func (self *SInstance) Refresh() error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetSysTags() map[string]string {
	return nil
}

func (self *SInstance) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SInstance) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotFound
}

//	ICloudVM
func (self *SInstance) ConvertPublicIpToEip() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetHostname() string {
	return ""
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return nil
}

func (self *SInstance) GetIHostId() string {
	return ""
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotFound
}

// 目前仅谷歌云windows机器会使用到此接口
func (self *SInstance) GetSerialOutput(port int) (string, error) {
	return "", cloudprovider.ErrNotFound
}

func (self *SInstance) GetVcpuCount() int {
	return 0
}

//MB
func (self *SInstance) GetVmemSizeMB() int {
	return 0
}

func (self *SInstance) GetBootOrder() string {
	return ""
}

func (self *SInstance) GetVga() string {
	return ""
}

func (self *SInstance) GetVdi() string {
	return ""
}

func (self *SInstance) GetOSArch() string {
	return ""
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	return ""
}

func (self *SInstance) GetOSName() string {
	return ""
}

func (self *SInstance) GetBios() string {
	return ""
}

func (self *SInstance) GetMachine() string {
	return ""
}

func (self *SInstance) GetInstanceType() string {
	return ""
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) GetHypervisor() string {
	return ""
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotFound
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	return "", cloudprovider.ErrNotFound
}

func (self *SInstance) MigrateVM(hostid string) error {
	return cloudprovider.ErrNotFound
}
func (self *SInstance) LiveMigrateVM(hostid string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) GetError() error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) CreateInstanceSnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotFound
}
func (self *SInstance) GetInstanceSnapshot(idStr string) (cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotFound
}
func (self *SInstance) GetInstanceSnapshots() ([]cloudprovider.ICloudInstanceSnapshot, error) {
	return nil, cloudprovider.ErrNotFound
}
func (self *SInstance) ResetToInstanceSnapshot(ctx context.Context, idStr string) error {
	return cloudprovider.ErrNotFound
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SInstance) AllocatePublicIpAddress() (string, error) {
	return "", cloudprovider.ErrNotFound
}
