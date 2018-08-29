package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IGuestDriver interface {
	GetHypervisor() string

	GetMaxVCpuCount() int
	GetMaxVMemSizeGB() int

	GetJsonDescAtHost(ctx context.Context, guest *SGuest, host *SHost) jsonutils.JSONObject

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)

	ValidateCreateHostData(ctx context.Context, userCred mcclient.TokenCredential, bmName string, host *SHost, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)

	PrepareDiskRaidConfig(host *SHost, params *jsonutils.JSONDict) error

	GetNamedNetworkConfiguration(guest *SGuest, userCred mcclient.TokenCredential, host *SHost, netConfig *SNetworkConfig) (*SNetwork, string, int8, IPAddlocationDirection)

	Attach2RandomNetwork(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *SNetworkConfig, pendingUsage quotas.IQuota) error

	ChooseHostStorage(host *SHost, backend string) *SStorage

	StartGuestCreateTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, pendingUsage quotas.IQuota, parentTaskId string) error

	RequestGuestCreateAllDisks(ctx context.Context, guest *SGuest, task taskman.ITask) error

	OnGuestCreateTaskComplete(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestGuestCreateInsertIso(ctx context.Context, imageId string, guest *SGuest, task taskman.ITask) error

	StartGuestStopTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error
	StartGuestResetTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, isHard bool, parentTaskId string) error

	RequestSoftReset(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestDeployGuestOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	OnGuestDeployTaskDataReceived(ctx context.Context, guest *SGuest, task taskman.ITask, data jsonutils.JSONObject) error

	OnGuestDeployTaskComplete(ctx context.Context, guest *SGuest, task taskman.ITask) error

	StartGuestSyncstatusTask(guest *SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error

	RequestSyncConfigOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	RequestSyncstatusOnHost(ctx context.Context, guest *SGuest, host *SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error)

	RequestStartOnHost(ctx context.Context, guest *SGuest, host *SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error)

	RequestStopOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	StartDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	RequestStopGuestForDelete(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *SGuest, task taskman.ITask) error

	RequestUndeployGuestOnHost(ctx context.Context, guest *SGuest, host *SHost, task taskman.ITask) error

	OnDeleteGuestFinalCleanup(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential) error

	PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, data *jsonutils.JSONDict) error

	CheckDiskTemplateOnStorage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, storageId string, task taskman.ITask) error

	GetGuestVncInfo(userCred mcclient.TokenCredential, guest *SGuest, host *SHost) (*jsonutils.JSONDict, error)

	RequestDetachDisk(ctx context.Context, guest *SGuest, task taskman.ITask) error
	GetDetachDiskStatus() ([]string, error)
	CanKeepDetachDisk() bool

	RequestDeleteDetachedDisk(ctx context.Context, disk *SDisk, task taskman.ITask, isPurge bool) error
	StartGuestDetachdiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error

	StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, params *jsonutils.JSONDict, parentTaskId string) error
	RqeuestSuspendOnHost(ctx context.Context, guest *SGuest, task taskman.ITask) error

	AllowReconfigGuest() bool
	DoGuestCreateDisksTask(ctx context.Context, guest *SGuest, task taskman.ITask) error
	RequestChangeVmConfig(ctx context.Context, guest *SGuest, task taskman.ITask, vcpuCount, vmemSize int64) error

	RequestGuestHotAddIso(ctx context.Context, guest *SGuest, path string, task taskman.ITask) error
	RequestRebuildRootDisk(ctx context.Context, guest *SGuest, task taskman.ITask) error
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
		log.Fatalf("Unsupported hypervisor %s", hypervisor)
		return nil
	}
}
