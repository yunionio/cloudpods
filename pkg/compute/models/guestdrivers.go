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

package models

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type IGuestScheduleDriver interface {
	DoScheduleSKUFilter() bool
	DoScheduleCPUFilter() bool
	DoScheduleMemoryFilter() bool
	DoScheduleStorageFilter() bool
}

type IGuestDriver interface {
	IGuestScheduleDriver

	GetHypervisor() string
	GetProvider() string
	GetQuotaPlatformID() []string

	GetMaxVCpuCount() int
	GetMaxVMemSizeGB() int
	GetMaxSecurityGroupCount() int

	GetDefaultSysDiskBackend() string
	GetMinimalSysDiskSizeGb() int

	IsSupportedBillingCycle(bc billing.SBillingCycle) bool

	RequestRenewInstance(guest *SGuest, bc billing.SBillingCycle) (time.Time, error)

	GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, host *SHost) jsonutils.JSONObject

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *api.ServerCreateInput) (*api.ServerCreateInput, error)

	ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)

	ValidateCreateDataOnHost(ctx context.Context, userCred mcclient.TokenCredential, bmName string, host *SHost, input *api.ServerCreateInput) (*api.ServerCreateInput, error)

	PrepareDiskRaidConfig(userCred mcclient.TokenCredential, host *SHost, params []*api.BaremetalDiskConfig) error

	GetNamedNetworkConfiguration(guest *SGuest, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig) (*SNetwork, []SNicConfig, api.IPAllocationDirection)

	Attach2RandomNetwork(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]SGuestnetwork, error)
	GetRandomNetworkTypes() []string

	GetStorageTypes() []string
	ChooseHostStorage(host *SHost, backend string, storageIds []string) *SStorage

	StartGuestCreateTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, pendingUsage quotas.IQuota, parentTaskId string) error

	RequestGuestCreateAllDisks(ctx context.Context, guest *SGuest, task taskman.ITask) error

	OnGuestCreateTaskComplete(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestGuestCreateInsertIso(ctx context.Context, imageId string, guest *SGuest, task taskman.ITask) error

	StartGuestStopTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error
	StartGuestResetTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, isHard bool, parentTaskId string) error
	StartGuestRestartTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error

	RequestSoftReset(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestDeployGuestOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error
	RemoteDeployGuestForCreate(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, host *SHost, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error)
	RemoteDeployGuestForDeploy(ctx context.Context, guest *SGuest, ihost cloudprovider.ICloudHost, task taskman.ITask, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error)
	RemoteDeployGuestForRebuildRoot(ctx context.Context, guest *SGuest, ihost cloudprovider.ICloudHost, task taskman.ITask, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error)
	GetGuestInitialStateAfterCreate() string
	GetGuestInitialStateAfterRebuild() string
	GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string

	OnGuestDeployTaskDataReceived(ctx context.Context, guest *SGuest, task taskman.ITask, data jsonutils.JSONObject) error

	StartGuestSyncstatusTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error

	RequestSyncConfigOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error
	RequestSyncSecgroupsOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error
	GetGuestSecgroupVpcid(guest *SGuest) (string, error)

	RequestSyncstatusOnHost(ctx context.Context, guest *SGuest, host *SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error)

	RequestStartOnHost(ctx context.Context, guest *SGuest, host *SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error)

	RequestStopOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	StartDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	StartGuestSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	RequestStopGuestForDelete(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestUndeployGuestOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	OnDeleteGuestFinalCleanup(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential) error

	PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, data *jsonutils.JSONDict) error

	CheckDiskTemplateOnStorage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, storageId string, task taskman.ITask) error

	GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, host *SHost) (*jsonutils.JSONDict, error)

	RequestAttachDisk(ctx context.Context, guest *SGuest, disk *SDisk, task taskman.ITask) error
	RequestDetachDisk(ctx context.Context, guest *SGuest, disk *SDisk, task taskman.ITask) error
	GetDetachDiskStatus() ([]string, error)
	GetAttachDiskStatus() ([]string, error)
	GetRebuildRootStatus() ([]string, error)
	GetChangeConfigStatus() ([]string, error)
	GetDeployStatus() ([]string, error)
	ValidateResizeDisk(guest *SGuest, disk *SDisk, storage *SStorage) error
	CanKeepDetachDisk() bool
	IsNeedRestartForResetLoginInfo() bool
	IsRebuildRootSupportChangeImage() bool

	RequestDeleteDetachedDisk(ctx context.Context, disk *SDisk, task taskman.ITask, isPurge bool) error
	StartGuestDetachdiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error
	StartGuestAttachDiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error
	RqeuestSuspendOnHost(ctx context.Context, guest *SGuest, task taskman.ITask) error

	AllowReconfigGuest() bool
	DoGuestCreateDisksTask(ctx context.Context, guest *SGuest, task taskman.ITask) error
	RequestChangeVmConfig(ctx context.Context, guest *SGuest, task taskman.ITask, instanceType string, vcpuCount, vmemSize int64) error

	RequestGuestHotAddIso(ctx context.Context, guest *SGuest, path string, task taskman.ITask) error
	RequestRebuildRootDisk(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestDiskSnapshot(ctx context.Context, guest *SGuest, task taskman.ITask, snapshotId, diskId string) error
	RequestDeleteSnapshot(ctx context.Context, guest *SGuest, task taskman.ITask, params *jsonutils.JSONDict) error
	RequestReloadDiskSnapshot(ctx context.Context, guest *SGuest, task taskman.ITask, params *jsonutils.JSONDict) error
	RequestSyncToBackup(ctx context.Context, guest *SGuest, task taskman.ITask) error

	IsSupportEip() bool
	ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error

	NeedStopForChangeSpec(guest *SGuest) bool

	OnGuestChangeCpuMemFailed(ctx context.Context, guest *SGuest, data *jsonutils.JSONDict, task taskman.ITask) error
	IsSupportGuestClone() bool

	ValidateChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, cpuChanged bool, memChanged bool, newDisks []*api.DiskConfig) error
	ValidateDetachDisk(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, disk *SDisk) error

	IsNeedInjectPasswordByCloudInit(desc *cloudprovider.SManagedVMCreateConfig) bool
	GetUserDataType() string
	CancelExpireTime(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error
}

var guestDrivers map[string]IGuestDriver

func init() {
	guestDrivers = make(map[string]IGuestDriver)
}

func RegisterGuestDriver(driver IGuestDriver) {
	guestDrivers[driver.GetHypervisor()] = driver
}

func GetDriver(hypervisor string) IGuestDriver {
	driver, ok := guestDrivers[hypervisor]
	if ok {
		return driver
	} else {
		panic(fmt.Sprintf("Unsupported hypervisor %q", hypervisor))
	}
}
