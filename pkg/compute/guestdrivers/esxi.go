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

package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SESXiGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SESXiGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SESXiGuestDriver) DoScheduleCPUFilter() bool { return true }

func (self *SESXiGuestDriver) DoScheduleMemoryFilter() bool { return true }

func (self *SESXiGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SESXiGuestDriver) DoScheduleStorageFilter() bool { return true }

func (self *SESXiGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_ESXI
}

func (self *SESXiGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_VMWARE
}

func (self *SESXiGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_ON_PREMISE
	keys.Provider = api.CLOUD_PROVIDER_VMWARE
	keys.Brand = api.CLOUD_PROVIDER_VMWARE
	keys.Hypervisor = api.HYPERVISOR_ESXI
	return keys
}

func (self *SESXiGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (self *SESXiGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SESXiGuestDriver) RequestSyncSecgroupsOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return nil // do nothing, not support securitygroup
}

func (self *SESXiGuestDriver) GetMaxSecurityGroupCount() int {
	//暂不支持绑定安全组
	return 0
}

func (self *SESXiGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SESXiGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SESXiGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SESXiGuestDriver) CanKeepDetachDisk() bool {
	return false
}

// func (self *SESXiGuestDriver) RequestDeleteDetachedDisk(ctx context.Context, disk *models.SDisk, task taskman.ITask, isPurge bool) error {
// 	err := disk.RealDelete(ctx, task.GetUserCred())
// 	if err != nil {
// 		return err
// 	}
// 	task.ScheduleRun(nil)
// 	return nil
// }

func (self *SESXiGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SESXiGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SESXiGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SESXiGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	return httperrors.NewInputParameterError("%s not support create eip", self.GetHypervisor())
}

func (self *SESXiGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == api.DISK_TYPE_SYS {
		return fmt.Errorf("Cannot resize system disk")
	}
	/*if !utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_PUBLIC_CLOUD, models.STORAGE_CLOUD_SSD, models.STORAGE_CLOUD_EFFICIENCY}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}*/
	return nil
}

type SEsxiImageInfo struct {
	ImageType          string
	ImageExternalId    string
	StorageCacheHostIp string
}

func (self *SESXiGuestDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	desc := guest.GetJsonDescAtHypervisor(ctx, host)
	// add image_info
	disks, _ := desc.GetArray("disks")
	if len(disks) == 0 {
		return desc
	}
	templateId, _ := disks[0].GetString("template_id")
	if len(templateId) == 0 {
		return desc
	}
	model, err := models.CachedimageManager.FetchById(templateId)
	if err != nil {
		log.Errorf("fail to Fetch cachedimage by '%s' in SESXiGuestDriver.GetJsonDescAtHost: %s", templateId, err)
		return desc
	}
	img := model.(*models.SCachedimage)
	if img.ImageType != cloudprovider.CachedImageTypeSystem {
		return desc
	}
	sciSubQ := models.StoragecachedimageManager.Query("storagecache_id").Equals("cachedimage_id", templateId).Equals("status", api.CACHED_IMAGE_STATUS_ACTIVE).SubQuery()
	scQ := models.StoragecacheManager.Query().In("id", sciSubQ)
	storageCaches := make([]models.SStoragecache, 0, 1)
	err = db.FetchModelObjects(models.StoragecacheManager, scQ, &storageCaches)
	if err != nil {
		log.Errorf("fail to fetch storageCache associated with cacheimage '%s'", templateId)
		return desc
	}
	if len(storageCaches) == 0 {
		log.Errorf("no such storage cache associated with cacheimage '%s'", templateId)
		return desc
	}
	if len(storageCaches) > 1 {
		log.Errorf("there are multiple storageCache associated with caheimage '%s' ??!!", templateId)
	}

	var hostIp string
	storageCacheHost, err := storageCaches[0].GetHost()
	if err != nil {
		log.Errorf("fail to GetHost of storageCache %s", storageCaches[0].Id)
		hostIp = storageCaches[0].ExternalId
	}
	hostIp = storageCacheHost.AccessIp
	imageInfo := SEsxiImageInfo{
		ImageType:          img.ImageType,
		ImageExternalId:    img.ExternalId,
		StorageCacheHostIp: hostIp,
	}
	dict := disks[0].(*jsonutils.JSONDict)
	dict.Add(jsonutils.Marshal(imageInfo), "image_info")
	return desc
}

func (self *SESXiGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	log.Debugf("RequestDeployGuestOnHost: %s", config)

	if !host.IsEsxiAgentReady() {
		return fmt.Errorf("No ESXi agent host")
	}

	diskCat := guest.CategorizeDisks()
	if diskCat.Root == nil {
		return fmt.Errorf("no root disk???")
	}
	storage := diskCat.Root.GetStorage()
	if storage == nil {
		return fmt.Errorf("root disk has no storage???")
	}

	config.Add(jsonutils.NewString(host.AccessIp), "host_ip")
	config.Add(jsonutils.NewString(guest.Id), "guest_id")
	extId := guest.Id
	if len(guest.ExternalId) > 0 {
		extId = guest.ExternalId
	}
	config.Add(jsonutils.NewString(extId), "guest_ext_id")

	account := host.GetCloudaccount()
	accessInfo, err := account.GetVCenterAccessInfo(storage.ExternalId)
	if err != nil {
		return err
	}

	action, _ := config.GetString("action")
	if action == "create" {
		project, err := db.TenantCacheManager.FetchTenantById(ctx, guest.ProjectId)
		if err != nil {
			return errors.Wrapf(err, "FetchTenantById(%s)", guest.ProjectId)
		}

		projects, err := account.GetExternalProjectsByProjectIdOrName(project.Id, project.Name)
		if err != nil {
			return errors.Wrapf(err, "GetExternalProjectsByProjectIdOrName(%s,%s)", project.Id, project.Name)
		}

		extProj := account.GetAvailableExternalProject(project, projects)
		if extProj != nil {
			config.Add(jsonutils.NewString(extProj.Name), "desc", "resource_pool")
		} else {
			config.Add(jsonutils.NewString(project.Name), "desc", "resource_pool")
		}
	}

	config.Add(jsonutils.Marshal(accessInfo), "datastore")

	url := "/disks/agent/deploy"

	body := jsonutils.NewDict()
	body.Add(config, "disk")

	header := task.GetTaskRequestHeader()

	_, err = host.EsxiRequest(ctx, httputils.POST, url, header, body)
	return err
}

func (self *SESXiGuestDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		host := guest.GetHost()
		if host == nil {
			return nil, errors.Error("fail to get host of guest")
		}
		ihost, err := host.GetIHost()
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil {
			return nil, err
		}
		vm := ivm.(*esxi.SVirtualMachine)
		err = vm.SuspendVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "VM.SuspendVM for vmware")
		}
		return nil, nil
	})
	return nil
}

func (self *SESXiGuestDriver) RqeuestResumeOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		host := guest.GetHost()
		if host == nil {
			return nil, errors.Error("fail to get host of guest")
		}
		ihost, err := host.GetIHost()
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil {
			return nil, err
		}
		vm := ivm.(*esxi.SVirtualMachine)
		err = vm.ResumeVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "VM.Resume for VMware")
		}
		return nil, nil
	})
	return nil
}

func (self *SESXiGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {

	if data.Contains("host_ip") {
		hostIp, _ := data.GetString("host_ip")
		host, err := models.HostManager.GetHostByIp(hostIp)
		if err != nil {
			log.Errorf("fail to find host with IP %s: %s", hostIp, err)
			return err
		}
		if host.Id != guest.HostId {
			models.HostManager.ClearSchedDescCache(host.Id)
			models.HostManager.ClearSchedDescCache(guest.HostId)
			guest.OnScheduleToHost(ctx, task.GetUserCred(), host.Id)
		}
	}

	return self.SManagedVirtualizedGuestDriver.OnGuestDeployTaskDataReceived(ctx, guest, task, data)
}

func (self *SESXiGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SESXiGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ESXiGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SESXiGuestDriver) RequestRenewInstance(guest *models.SGuest, bc billing.SBillingCycle) (time.Time, error) {
	return time.Time{}, nil
}

func (self *SESXiGuestDriver) IsSupportEip() bool {
	return false
}

func (self *SESXiGuestDriver) RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, eip *models.SElasticip, task taskman.ITask) error {
	return fmt.Errorf("ESXiGuestDriver not support associate eip")
}

func (self *SESXiGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return false, nil
}

func (self *SESXiGuestDriver) IsSupportMigrate() bool {
	return true
}

func (self *SESXiGuestDriver) IsSupportLiveMigrate() bool {
	return true
}

func (self *SESXiGuestDriver) CheckMigrate(guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestMigrateInput) error {
	if len(input.PreferHost) == 0 {
		return httperrors.NewBadRequestError("esxi guest migrate require prefer_host")
	}
	return nil
}

func (self *SESXiGuestDriver) CheckLiveMigrate(guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput) error {
	if len(input.PreferHost) == 0 {
		return httperrors.NewBadRequestError("esxi guest migrate require prefer_host")
	}
	return nil
}

func (self *SESXiGuestDriver) RequestMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iVM, err := guest.GetIVM()
		if err != nil {
			return nil, errors.Wrap(err, "guest.GetIVM")
		}
		hostID, _ := data.GetString("prefer_host_id")
		if hostID == "" {
			return nil, errors.Wrapf(fmt.Errorf("require hostID"), "RequestMigrate")
		}
		iHost, err := models.HostManager.FetchById(hostID)
		if err != nil {
			return nil, errors.Wrapf(err, "models.HostManager.FetchById(%s)", hostID)
		}
		host := iHost.(*models.SHost)
		hostExternalId := host.ExternalId
		if err = iVM.MigrateVM(hostExternalId); err != nil {
			return nil, errors.Wrapf(err, "iVM.MigrateVM(%s)", hostExternalId)
		}
		hostExternalId = iVM.GetIHostId()
		if hostExternalId == "" {
			return nil, errors.Wrap(fmt.Errorf("empty hostExternalId"), "iVM.GetIHostId()")
		}
		iHost, err = db.FetchByExternalIdAndManagerId(models.HostManager, hostExternalId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			if host := guest.GetHost(); host != nil {
				return q.Equals("manager_id", host.ManagerId)
			}
			return q
		})
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchByExternalId(models.HostManager,%s)", hostExternalId)
		}
		host = iHost.(*models.SHost)
		_, err = db.Update(guest, func() error {
			guest.HostId = host.GetId()
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "db.Update guest.hostId")
		}
		return nil, nil
	})
	return nil
}

func (self *SESXiGuestDriver) RequestLiveMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iVM, err := guest.GetIVM()
		if err != nil {
			return nil, errors.Wrap(err, "guest.GetIVM")
		}
		hostID, _ := data.GetString("prefer_host_id")
		if hostID == "" {
			return nil, errors.Wrapf(fmt.Errorf("require hostID"), "RequestLiveMigrate")
		}
		iHost, err := models.HostManager.FetchById(hostID)
		if err != nil {
			return nil, errors.Wrapf(err, "models.HostManager.FetchById(%s)", hostID)
		}
		host := iHost.(*models.SHost)
		hostExternalId := host.ExternalId
		if err = iVM.LiveMigrateVM(hostExternalId); err != nil {
			return nil, errors.Wrapf(err, "iVM.LiveMigrateVM(%s)", hostExternalId)
		}
		hostExternalId = iVM.GetIHostId()
		if hostExternalId == "" {
			return nil, errors.Wrap(fmt.Errorf("empty hostExternalId"), "iVM.GetIHostId()")
		}
		iHost, err = db.FetchByExternalIdAndManagerId(models.HostManager, hostExternalId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			if host := guest.GetHost(); host != nil {
				return q.Equals("manager_id", host.ManagerId)
			}
			return q
		})
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchByExternalId(models.HostManager,%s)", hostExternalId)
		}
		host = iHost.(*models.SHost)
		_, err = db.Update(guest, func() error {
			guest.HostId = host.GetId()
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "db.Update guest.hostId")
		}
		return nil, nil
	})
	return nil
}
