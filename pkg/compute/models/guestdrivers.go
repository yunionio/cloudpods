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
	"net/http"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	guestdriver_types "yunion.io/x/onecloud/pkg/compute/guestdrivers/types"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IGuestScheduleDriver interface {
	DoScheduleSKUFilter() bool
	DoScheduleCPUFilter() bool
	DoScheduleMemoryFilter() bool
	DoScheduleStorageFilter() bool
	DoScheduleCloudproviderTagFilter() bool
}

type IGuestDriver interface {
	IGuestScheduleDriver

	GetHypervisor() string
	GetProvider() string
	GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) SComputeResourceKeys

	GetMaxVCpuCount() int
	GetMaxVMemSizeGB() int
	GetMaxSecurityGroupCount() int

	GetDefaultSysDiskBackend() string
	GetMinimalSysDiskSizeGb() int

	IsSupportedBillingCycle(bc billing.SBillingCycle) bool
	IsSupportPostpaidExpire() bool
	IsSupportShutdownMode() bool

	RequestRenewInstance(ctx context.Context, guest *SGuest, bc billing.SBillingCycle) (time.Time, error)

	GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, host *SHost, params *jsonutils.JSONDict) (jsonutils.JSONObject, error)

	ValidateImage(ctx context.Context, image *cloudprovider.SImage) error
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *api.ServerCreateInput) (*api.ServerCreateInput, error)

	ValidateCreateDataOnHost(ctx context.Context, userCred mcclient.TokenCredential, bmName string, host *SHost, input *api.ServerCreateInput) (*api.ServerCreateInput, error)

	PrepareDiskRaidConfig(userCred mcclient.TokenCredential, host *SHost, params []*api.BaremetalDiskConfig, disks []*api.DiskConfig) ([]*api.DiskConfig, error)

	GetNamedNetworkConfiguration(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig) (*SNetwork, []SNicConfig, api.IPAllocationDirection, bool, error)

	Attach2RandomNetwork(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]SGuestnetwork, error)
	GetRandomNetworkTypes() []string

	GetStorageTypes() []string
	ChooseHostStorage(host *SHost, guest *SGuest, diskConfig *api.DiskConfig, storageIds []string) (*SStorage, error)

	StartGuestCreateTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, pendingUsage quotas.IQuota, parentTaskId string) error

	RequestGuestCreateAllDisks(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestGuestCreateInsertIso(ctx context.Context, imageId string, bootIndex *int8, task taskman.ITask, guest *SGuest) error

	StartGuestStopTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error
	StartGuestResetTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, isHard bool, parentTaskId string) error
	StartGuestRestartTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error

	RequestSoftReset(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestDeployGuestOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error
	RemoteDeployGuestForCreate(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, host *SHost, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error)
	RemoteDeployGuestSyncHost(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, host *SHost, iVM cloudprovider.ICloudVM) (cloudprovider.ICloudHost, error)
	RemoteActionAfterGuestCreated(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, host *SHost, iVM cloudprovider.ICloudVM, desc *cloudprovider.SManagedVMCreateConfig)
	RemoteDeployGuestForDeploy(ctx context.Context, guest *SGuest, ihost cloudprovider.ICloudHost, task taskman.ITask, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error)
	RemoteDeployGuestForRebuildRoot(ctx context.Context, guest *SGuest, ihost cloudprovider.ICloudHost, task taskman.ITask, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error)
	GetGuestInitialStateAfterCreate() string
	GetGuestInitialStateAfterRebuild() string
	GetDefaultAccount(osType, osDist, imageType string) string
	GetInstanceCapability() cloudprovider.SInstanceCapability

	OnGuestDeployTaskDataReceived(ctx context.Context, guest *SGuest, task taskman.ITask, data jsonutils.JSONObject) error

	StartGuestSyncstatusTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error

	RequestSyncConfigOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error
	RequestSyncSecgroupsOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	RequestSyncstatusOnHost(ctx context.Context, guest *SGuest, host *SHost, userCred mcclient.TokenCredential, task taskman.ITask) error

	RequestStartOnHost(ctx context.Context, guest *SGuest, host *SHost, userCred mcclient.TokenCredential, task taskman.ITask) error

	RequestStopOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask, syncStatus bool) error

	StartDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	StartGuestSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error
	RequestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, task taskman.ITask) error

	RequestStopGuestForDelete(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestUndeployGuestOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	OnDeleteGuestFinalCleanup(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential) error

	PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, data *jsonutils.JSONDict) error

	CheckDiskTemplateOnStorage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, storageId string, task taskman.ITask) error

	GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, host *SHost, input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error)

	RequestAttachDisk(ctx context.Context, guest *SGuest, disk *SDisk, task taskman.ITask) error
	RequestDetachDisk(ctx context.Context, guest *SGuest, disk *SDisk, task taskman.ITask) error
	GetDetachDiskStatus() ([]string, error)
	GetAttachDiskStatus() ([]string, error)
	GetRebuildRootStatus() ([]string, error)
	IsAllowSaveImageOnRunning() bool
	GetChangeConfigStatus(guest *SGuest) ([]string, error)
	GetDeployStatus() ([]string, error)
	ValidateResizeDisk(guest *SGuest, disk *SDisk, storage *SStorage) error
	CanKeepDetachDisk() bool
	IsNeedRestartForResetLoginInfo() bool
	IsRebuildRootSupportChangeImage() bool
	IsRebuildRootSupportChangeUEFI() bool

	ValidateRebuildRoot(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, input *api.ServerRebuildRootInput) (*api.ServerRebuildRootInput, error)
	ValidateDetachNetwork(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error

	IsSupportdDcryptPasswordFromSecretKey() bool

	RequestDeleteDetachedDisk(ctx context.Context, disk *SDisk, task taskman.ITask, isPurge bool) error
	StartGuestDetachdiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error
	StartGuestAttachDiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error
	RequestSuspendOnHost(ctx context.Context, guest *SGuest, task taskman.ITask) error

	StartResumeTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error
	RequestResumeOnHost(ctx context.Context, guest *SGuest, task taskman.ITask) error

	AllowReconfigGuest() bool
	DoGuestCreateDisksTask(ctx context.Context, guest *SGuest, task taskman.ITask) error
	RequestChangeVmConfig(ctx context.Context, guest *SGuest, task taskman.ITask, instanceType string, vcpuCount, cpuSockets, vmemSize int64) error

	NeedRequestGuestHotAddIso(ctx context.Context, guest *SGuest) bool
	RequestGuestHotAddIso(ctx context.Context, guest *SGuest, path string, boot bool, task taskman.ITask) error
	RequestGuestHotRemoveIso(ctx context.Context, guest *SGuest, task taskman.ITask) error
	RequestRebuildRootDisk(ctx context.Context, guest *SGuest, task taskman.ITask) error

	NeedRequestGuestHotAddVfd(ctx context.Context, guest *SGuest) bool
	RequestGuestHotAddVfd(ctx context.Context, guest *SGuest, path string, boot bool, task taskman.ITask) error
	RequestGuestHotRemoveVfd(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestDiskSnapshot(ctx context.Context, guest *SGuest, task taskman.ITask, snapshotId, diskId string) error
	RequestDeleteSnapshot(ctx context.Context, guest *SGuest, task taskman.ITask, params *jsonutils.JSONDict) error
	RequestReloadDiskSnapshot(ctx context.Context, guest *SGuest, task taskman.ITask, params *jsonutils.JSONDict) error
	RequestSyncToBackup(ctx context.Context, guest *SGuest, task taskman.ITask) error
	RequestSlaveBlockStreamDisks(ctx context.Context, guest *SGuest, task taskman.ITask) error

	IsSupportEip() bool
	IsSupportPublicIp() bool
	ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error

	NeedStopForChangeSpec(ctx context.Context, guest *SGuest, addCpu int, addMemMb int) bool

	OnGuestChangeCpuMemFailed(ctx context.Context, guest *SGuest, data *jsonutils.JSONDict, task taskman.ITask) error
	IsSupportGuestClone() bool

	ValidateChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, cpuChanged bool, memChanged bool, newDisks []*api.DiskConfig) error
	ValidateDetachDisk(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, disk *SDisk) error

	IsNeedInjectPasswordByCloudInit() bool
	GetUserDataType() string
	GetWindowsUserDataType() string
	IsWindowsUserDataTypeNeedEncode() bool
	CancelExpireTime(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error

	IsSupportCdrom(guest *SGuest) (bool, error)
	IsSupportFloppy(guest *SGuest) (bool, error)
	IsSupportPublicipToEip() bool
	RequestConvertPublicipToEip(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, task taskman.ITask) error

	IsSupportSetAutoRenew() bool
	RequestSetAutoRenewInstance(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, input api.GuestAutoRenewInput, task taskman.ITask) error
	IsSupportMigrate() bool
	IsSupportLiveMigrate() bool
	CheckMigrate(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential, input api.GuestMigrateInput) error
	CheckLiveMigrate(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput) error
	RequestMigrate(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential, input api.GuestMigrateInput, task taskman.ITask) error
	RequestLiveMigrate(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput, task taskman.ITask) error
	RequestCancelLiveMigrate(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential) error

	ValidateUpdateData(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential, input api.ServerUpdateInput) (api.ServerUpdateInput, error)
	RequestRemoteUpdate(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential, replaceTags bool) error

	RequestOpenForward(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, req *guestdriver_types.OpenForwardRequest) (*guestdriver_types.OpenForwardResponse, error)
	RequestListForward(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, req *guestdriver_types.ListForwardRequest) (*guestdriver_types.ListForwardResponse, error)
	RequestCloseForward(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, req *guestdriver_types.CloseForwardRequest) (*guestdriver_types.CloseForwardResponse, error)

	ValidateChangeDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, targetStorageId string) error
	StartChangeDiskStorageTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *api.ServerChangeDiskStorageInternalInput, parentTaskId string) error
	RequestChangeDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, input *api.ServerChangeDiskStorageInternalInput, task taskman.ITask) error
	RequestSwitchToTargetStorageDisk(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, input *api.ServerChangeDiskStorageInternalInput, task taskman.ITask) error

	RequestSyncIsolatedDevice(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestCPUSet(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, guest *SGuest, input *api.ServerCPUSetInput) (*api.ServerCPUSetResp, error)
	RequestCPUSetRemove(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, guest *SGuest, input *api.ServerCPUSetRemoveInput) error

	QgaRequestGuestPing(ctx context.Context, header http.Header, host *SHost, guest *SGuest, async bool, input *api.ServerQgaTimeoutInput) error
	QgaRequestSetUserPassword(ctx context.Context, task taskman.ITask, host *SHost, guest *SGuest, input *api.ServerQgaSetPasswordInput) error
	RequestQgaCommand(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject, host *SHost, guest *SGuest) (jsonutils.JSONObject, error)
	QgaRequestGuestInfoTask(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject, host *SHost, guest *SGuest) (jsonutils.JSONObject, error)
	QgaRequestSetNetwork(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject, host *SHost, guest *SGuest) (jsonutils.JSONObject, error)
	QgaRequestGetNetwork(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject, host *SHost, guest *SGuest) (jsonutils.JSONObject, error)

	FetchMonitorUrl(ctx context.Context, guest *SGuest) string
	RequestResetNicTrafficLimit(ctx context.Context, task taskman.ITask, host *SHost, guest *SGuest, input []api.ServerNicTrafficLimit) error
	RequestSetNicTrafficLimit(ctx context.Context, task taskman.ITask, host *SHost, guest *SGuest, input []api.ServerNicTrafficLimit) error

	SyncOsInfo(ctx context.Context, userCred mcclient.TokenCredential, g *SGuest, extVM cloudprovider.IOSInfo) error

	RequestStartRescue(ctx context.Context, task taskman.ITask, body jsonutils.JSONObject, host *SHost, guest *SGuest) error
	RequestStopRescue(ctx context.Context, task taskman.ITask, body jsonutils.JSONObject, host *SHost, guest *SGuest) error

	ValidateSetOSInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, input *api.ServerSetOSInfoInput) error
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

func GetNotSupportAutoRenewHypervisors() []string {
	hypervisors := []string{}
	for hypervisor, driver := range guestDrivers {
		if !driver.IsSupportSetAutoRenew() {
			hypervisors = append(hypervisors, hypervisor)
		}
	}
	return hypervisors
}
