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
	"sort"
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

func (self *SESXiGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
				Changeable:     true,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
				Changeable:     false,
			},
		},
	}
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
	return ""
}

func (self *SESXiGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	switch {
	case !options.Options.LockStorageFromCachedimage:
		return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
	case len(diskConfig.ImageId) > 0:
		var (
			image *cloudprovider.SImage
			err   error
		)
		obj, err := models.CachedimageManager.FetchById(diskConfig.ImageId)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fetch cachedimage %s", diskConfig.ImageId)
		}
		cachedimage := obj.(*models.SCachedimage)
		if len(cachedimage.ExternalId) > 0 || cloudprovider.TImageType(cachedimage.ImageType) != cloudprovider.ImageTypeSystem {
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		storages, err := cachedimage.GetStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to GetStorages of cachedimage %s", diskConfig.ImageId)
		}
		if len(storages) == 0 {
			log.Warningf("there no storage associated with cachedimage %q", image.Id)
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		if len(storages) > 1 {
			log.Warningf("there are multiple storageCache associated with caheimage %q", image.Id)
		}
		wantStorageIds := make([]string, len(storages))
		for i := range wantStorageIds {
			wantStorageIds[i] = storages[i].GetId()
		}
		for i := range wantStorageIds {
			if utils.IsInStringArray(wantStorageIds[i], storageIds) {
				log.Infof("use storage %q in where cachedimage %q", wantStorageIds[i], image.Id)
				return &storages[i], nil
			}
		}
		return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
	default:
		ispId := guest.GetMetadata(context.Background(), "__base_instance_snapshot_id", nil)
		if len(ispId) == 0 {
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		obj, err := models.InstanceSnapshotManager.FetchById(ispId)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fetch InstanceSnapshot %q", ispId)
		}
		isp := obj.(*models.SInstanceSnapshot)
		ispGuest, err := isp.GetGuest()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fetch Guest of InstanceSnapshot %q", ispId)
		}
		storages := ispGuest.GetStorages()
		if len(storages) == 0 {
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		if utils.IsInStringArray(storages[0].GetId(), storageIds) {
			return storages[0], nil
		}
		return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
	}
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

func (self *SESXiGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	// check disk config
	if len(data.Disks) == 0 {
		return data, nil
	}
	rootDisk := data.Disks[0]
	if len(rootDisk.ImageId) == 0 {
		return data, nil
	}
	image, err := models.CachedimageManager.GetImageInfo(ctx, userCred, rootDisk.ImageId, false)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to GetImageInfo of image %q", rootDisk.ImageId)
	}
	if len(image.SubImages) <= 1 {
		return data, nil
	}
	sort.Slice(image.SubImages, func(i, j int) bool {
		return image.SubImages[i].Index < image.SubImages[j].Index
	})
	newDataDisks := make([]*api.DiskConfig, 0, len(image.SubImages)+len(data.Disks)-1)
	for i, subImage := range image.SubImages {
		nDataDisk := *rootDisk
		nDataDisk.SizeMb = subImage.MinDiskMB
		nDataDisk.Index = i
		if i > 0 {
			nDataDisk.ImageId = ""
		}
		newDataDisks = append(newDataDisks, &nDataDisk)
	}
	for i := 1; i < len(data.Disks); i++ {
		data.Disks[i].Index += len(image.SubImages) - 1
		newDataDisks = append(newDataDisks, data.Disks[i])
	}
	data.Disks = newDataDisks
	return data, nil
}

func (self *SESXiGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error {
	return httperrors.NewInputParameterError("%s not support create eip", self.GetHypervisor())
}

func (self *SESXiGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	count, err := guest.GetInstanceSnapshotCount()
	if err != nil {
		return errors.Wrapf(err, "unable to GetInstanceSnapshotCount for guest %q", guest.GetId())
	}
	if count > 0 {
		return httperrors.NewForbiddenError("can't resize disk for guest with instance snapshots")
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

type SEsxiInstanceSnapshotInfo struct {
	InstanceSnapshotId string
	InstanceId         string
}

func (self *SESXiGuestDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, params *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	desc := guest.GetJsonDescAtHypervisor(ctx, host)
	// add image_info
	if len(desc.Disks) == 0 {
		return jsonutils.Marshal(desc), nil
	}
	for i := range desc.Disks {
		diskId := desc.Disks[i].DiskId
		disk := models.DiskManager.FetchDiskById(diskId)
		if disk == nil {
			return nil, fmt.Errorf("unable to fetch disk %s", diskId)
		}
		storage, err := disk.GetStorage()
		if storage == nil {
			return nil, errors.Wrapf(err, "unable to fetch storage of disk %s", diskId)
		}
		desc.Disks[i].StorageId = storage.GetExternalId()
	}
	templateId := desc.Disks[0].TemplateId
	if len(templateId) == 0 {
		// try to check instance_snapshot_id
		ispId := guest.GetMetadata(ctx, "__base_instance_snapshot_id", userCred)
		if len(ispId) == 0 {
			return jsonutils.Marshal(desc), nil
		}
		obj, err := models.InstanceSnapshotManager.FetchById(ispId)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fetch InstanceSnapshot %q", ispId)
		}
		isp := obj.(*models.SInstanceSnapshot)
		ispGuest, err := isp.GetGuest()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fetch Guest of InstanceSnapshot %q", ispId)
		}
		desc.InstanceSnapshotInfo.InstanceSnapshotId = isp.GetExternalId()
		desc.InstanceSnapshotInfo.InstanceId = ispGuest.GetExternalId()
		return jsonutils.Marshal(desc), nil
	}
	model, err := models.CachedimageManager.FetchById(templateId)
	if err != nil {
		return jsonutils.Marshal(desc), errors.Wrapf(err, "CachedimageManager.FetchById(%s)", templateId)
	}
	img := model.(*models.SCachedimage)
	if cloudprovider.TImageType(img.ImageType) != cloudprovider.ImageTypeSystem {
		return jsonutils.Marshal(desc), nil
	}
	sciSubQ := models.StoragecachedimageManager.Query("storagecache_id").Equals("cachedimage_id", templateId).Equals("status", api.CACHED_IMAGE_STATUS_ACTIVE).SubQuery()
	scQ := models.StoragecacheManager.Query().In("id", sciSubQ)
	storageCaches := make([]models.SStoragecache, 0, 1)
	err = db.FetchModelObjects(models.StoragecacheManager, scQ, &storageCaches)
	if err != nil {
		return jsonutils.Marshal(desc), errors.Wrapf(err, "fetch storageCache associated with cacheimage %s", templateId)
	}
	if len(storageCaches) == 0 {
		return jsonutils.Marshal(desc), errors.Errorf("no such storage cache associated with cacheimage %s", templateId)
	}
	if len(storageCaches) > 1 {
		log.Warningf("there are multiple storageCache associated with caheimage '%s' ??!!", templateId)
	}

	var storageCacheHost *models.SHost
	// select storagecacheHost
	for i := range storageCaches {
		hosts, err := storageCaches[i].GetHosts()
		if err != nil {
			return jsonutils.Marshal(desc), errors.Wrap(err, "storageCaches.GetHosts")
		}
		for i := range hosts {
			if host.GetId() == hosts[i].GetId() {
				storageCacheHost = &hosts[i]
			}
		}
	}

	if storageCacheHost == nil {
		storageCacheHost, err = storageCaches[0].GetHost()
		if err != nil {
			return jsonutils.Marshal(desc), errors.Wrapf(err, "unable to GetHost of storageCache %s", storageCaches[0].Id)
		}
		if storageCacheHost == nil {
			return jsonutils.Marshal(desc), fmt.Errorf("unable to GetHost of storageCache %s: result is nil", storageCaches[0].Id)
		}
	}

	desc.Disks[0].ImageInfo.ImageType = img.ImageType
	desc.Disks[0].ImageInfo.ImageExternalId = img.ExternalId
	desc.Disks[0].ImageInfo.StorageCacheHostIp = storageCacheHost.AccessIp
	return jsonutils.Marshal(desc), nil
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
	storage, _ := diskCat.Root.GetStorage()
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

func (self *SESXiGuestDriver) RequestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		host, _ := guest.GetHost()
		if host == nil {
			return nil, errors.Error("fail to get host of guest")
		}
		ihost, err := host.GetIHost(ctx)
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

func (self *SESXiGuestDriver) RequestResumeOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		host, _ := guest.GetHost()
		if host == nil {
			return nil, errors.Error("fail to get host of guest")
		}
		ihost, err := host.GetIHost(ctx)
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

func (self *SESXiGuestDriver) RequestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, task taskman.ITask) error {
	disks := guest.CategorizeDisks()
	opts := api.DiskSaveInput{}
	task.GetParams().Unmarshal(&opts)
	return disks.Root.StartDiskSaveTask(ctx, userCred, opts, task.GetTaskId())
}

func (self *SESXiGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ESXiGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SESXiGuestDriver) RequestRenewInstance(ctx context.Context, guest *models.SGuest, bc billing.SBillingCycle) (time.Time, error) {
	return time.Time{}, nil
}

func (self *SESXiGuestDriver) IsSupportEip() bool {
	return false
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

func (self *SESXiGuestDriver) CheckMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestMigrateInput) error {
	if len(input.PreferHost) == 0 {
		return httperrors.NewBadRequestError("esxi guest migrate require prefer_host")
	}
	return nil
}

func (self *SESXiGuestDriver) CheckLiveMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput) error {
	if len(input.PreferHost) == 0 {
		return httperrors.NewBadRequestError("esxi guest migrate require prefer_host")
	}
	return nil
}

func (self *SESXiGuestDriver) RequestMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iVM, err := guest.GetIVM(ctx)
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
			if host, _ := guest.GetHost(); host != nil {
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
		iVM, err := guest.GetIVM(ctx)
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
			if host, _ := guest.GetHost(); host != nil {
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

func (self *SESXiGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	ivm, err := guest.GetIVM(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetIVM")
	}

	result := jsonutils.NewDict()
	if ivm.GetStatus() != api.VM_RUNNING {
		err := ivm.StartVM(ctx)
		if err != nil {
			return errors.Wrapf(err, "StartVM")
		}
		err = cloudprovider.WaitStatus(ivm, api.VM_RUNNING, time.Second*5, time.Minute*10)
		if err != nil {
			return errors.Wrapf(err, "Wait vm running")
		}
		guest.SetStatus(userCred, api.VM_RUNNING, "StartOnHost")
		return task.ScheduleRun(result)
	}
	return guest.SetStatus(userCred, api.VM_RUNNING, "StartOnHost")
}

func (self *SESXiGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask, syncStatus bool) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ivm, err := guest.GetIVM(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "guest.GetIVM")
		}
		opts := &cloudprovider.ServerStopOptions{}
		task.GetParams().Unmarshal(opts)
		err = ivm.StopVM(ctx, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "ivm.StopVM")
		}
		err = cloudprovider.WaitStatus(ivm, api.VM_READY, time.Second*3, time.Minute*5)
		if err != nil {
			return nil, errors.Wrapf(err, "wait server stop after 5 miniutes")
		}
		return nil, nil
	})
	return nil
}

func (self *SESXiGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "host.GetIHost")
		}
		ivm, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil {
			if errors.Cause(err) != errors.ErrNotFound {
				return nil, errors.Wrap(err, "ihost.GetIVMById")
			}
			// VM may be migrated by Vcenter, try to find VM from whole datacenter.
			ehost := ihost.(*esxi.SHost)
			dc, err := ehost.GetDatacenter()
			if err != nil {
				return nil, errors.Wrapf(err, "ehost.GetDatacenter")
			}
			vm, err := dc.FetchVMById(guest.GetExternalId())
			if err != nil {
				log.Errorf("fail to find ivm by id %q in dc %q: %v", guest.GetExternalId(), dc.GetName(), err)
				return nil, errors.Wrap(err, "dc.FetchVMById")
			}
			ihost = vm.GetIHost()
			host = models.HostManager.FetchHostByExtId(ihost.GetGlobalId())
			if host == nil {
				return nil, errors.Wrapf(errors.ErrNotFound, "find ivm %q in ihost %q which is not existed here", guest.GetExternalId(), ihost.GetGlobalId())
			}
			ivm = vm
		}
		err = guest.SyncAllWithCloudVM(ctx, userCred, host, ivm, true)
		if err != nil {
			return nil, errors.Wrap(err, "guest.SyncAllWithCloudVM")
		}

		status := GetCloudVMStatus(ivm)
		body := jsonutils.NewDict()
		body.Add(jsonutils.NewString(status), "status")
		return body, nil
	})
	return nil
}

func (self *SESXiGuestDriver) ValidateRebuildRoot(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, input *api.ServerRebuildRootInput) (*api.ServerRebuildRootInput, error) {
	// check snapshot
	count, err := guest.GetInstanceSnapshotCount()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to GetInstanceSnapshotCount for guest %q", guest.GetId())
	}
	if count > 0 {
		return input, httperrors.NewForbiddenError("can't rebuild root for a guest with instance snapshots")
	}
	return input, nil
}

func (self *SESXiGuestDriver) StartDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	params.Add(jsonutils.JSONTrue, "delete_snapshots")
	return self.SBaseGuestDriver.StartDeleteGuestTask(ctx, userCred, guest, params, parentTaskId)
}

func (self *SESXiGuestDriver) RequestRemoteUpdate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, replaceTags bool) error {
	return nil
}
