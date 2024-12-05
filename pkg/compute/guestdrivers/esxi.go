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
	"strconv"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SESXiGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

type SOneCloudGuestDriver struct {
	SESXiGuestDriver
}

func (self *SOneCloudGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func init() {
	driver := SESXiGuestDriver{}
	models.RegisterGuestDriver(&driver)

	driver2 := SOneCloudGuestDriver{}
	models.RegisterGuestDriver(&driver2)
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
		Storages: cloudprovider.Storage{
			SysDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_LOCAL, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_NAS, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_NFS, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_VSAN, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CIFS, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
			},
			DataDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_LOCAL, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_NAS, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_NFS, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_VSAN, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CIFS, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
			},
		},
	}
}

func (self *SESXiGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
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
		storages, err := ispGuest.GetStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "GetStorages")
		}
		if len(storages) == 0 {
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		if utils.IsInStringArray(storages[0].GetId(), storageIds) {
			return &storages[0], nil
		}
		return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
	}
}

func (self *SESXiGuestDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	iVM, err := guest.GetIVM(ctx)
	if err != nil {
		return nil, err
	}
	return iVM.GetVNCInfo(input)
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
	return []string{api.VM_READY, api.VM_RUNNING}, nil
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
	for i := 0; i < len(data.Disks); i++ {
		data.Disks[i].Format = "vmdk"
	}

	if data.CpuSockets > data.VcpuCount {
		return nil, httperrors.NewInputParameterError("The number of cpu sockets cannot be greater than the number of cpus")
	}

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
		nDataDisk.Format = "vmdk"
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
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
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
		desc.Disks[i].Preallocation = disk.Preallocation
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
		storageCacheHost, err = storageCaches[0].GetMasterHost()
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
	tags, _ := guest.GetAllUserMetadata()
	config.Set("tags", jsonutils.Marshal(tags))

	account := host.GetCloudaccount()
	accessInfo, err := account.GetVCenterAccessInfo(storage.ExternalId)
	if err != nil {
		return err
	}

	provider := host.GetCloudprovider()

	action, _ := config.GetString("action")
	if action == "create" {
		project, err := db.TenantCacheManager.FetchTenantById(ctx, guest.ProjectId)
		if err != nil {
			return errors.Wrapf(err, "FetchTenantById(%s)", guest.ProjectId)
		}

		projects, err := provider.GetExternalProjectsByProjectIdOrName(project.Id, project.Name)
		if err != nil {
			return errors.Wrapf(err, "GetExternalProjectsByProjectIdOrName(%s,%s)", project.Id, project.Name)
		}

		extProj := models.GetAvailableExternalProject(project, projects)
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
		ivm, err := guest.GetIVM(ctx)
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
		ivm, err := guest.GetIVM(ctx)
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
		oldHost, _ := guest.GetHost()
		hostIp, _ := data.GetString("host_ip")
		host, err := models.HostManager.GetHostByIp(oldHost.ManagerId, api.HOST_TYPE_ESXI, hostIp)
		if err != nil {
			return err
		}
		if host.Id != guest.HostId {
			models.HostManager.ClearSchedDescCache(host.Id)
			models.HostManager.ClearSchedDescCache(guest.HostId)
			guest.OnScheduleToHost(ctx, task.GetUserCred(), host.Id)
		}
	}

	err := self.SManagedVirtualizedGuestDriver.OnGuestDeployTaskDataReceived(ctx, guest, task, data)
	if err != nil {
		return nil
	}

	osInfo := struct {
		Arch    string
		Distro  string
		Os      string
		Version string
	}{}
	data.Unmarshal(&osInfo)

	osinfo := map[string]interface{}{}
	for k, v := range map[string]string{
		"os_arch":         osInfo.Arch,
		"os_distribution": osInfo.Distro,
		"os_type":         osInfo.Os,
		"os_name":         osInfo.Os,
		"os_version":      osInfo.Version,
	} {
		if len(v) > 0 {
			osinfo[k] = v
		}
	}
	if len(osinfo) > 0 {
		err := guest.SetAllMetadata(ctx, osinfo, task.GetUserCred())
		if err != nil {
			return errors.Wrap(err, "SetAllMetadata")
		}
	}
	return nil
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
	return nil
}

func (self *SESXiGuestDriver) CheckLiveMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput) error {
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
		guest.SetStatus(ctx, userCred, api.VM_RUNNING, "StartOnHost")
		return task.ScheduleRun(result)
	}
	return guest.SetStatus(ctx, userCred, api.VM_RUNNING, "StartOnHost")
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

func (self *SESXiGuestDriver) SyncOsInfo(ctx context.Context, userCred mcclient.TokenCredential, g *models.SGuest, extVM cloudprovider.IOSInfo) error {
	ometa, err := g.GetAllMetadata(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "GetAllMetadata")
	}
	// save os info
	osinfo := map[string]interface{}{}
	for k, v := range map[string]string{
		"os_full_name":    extVM.GetFullOsName(),
		"os_name":         string(extVM.GetOsType()),
		"os_arch":         extVM.GetOsArch(),
		"os_type":         string(extVM.GetOsType()),
		"os_distribution": extVM.GetOsDist(),
		"os_version":      extVM.GetOsVersion(),
		"os_language":     extVM.GetOsLang(),
	} {
		if len(v) == 0 || len(ometa[k]) > 0 {
			continue
		}
		osinfo[k] = v
	}
	if len(osinfo) > 0 {
		err := g.SetAllMetadata(ctx, osinfo, userCred)
		if err != nil {
			return errors.Wrap(err, "SetAllMetadata")
		}
	}
	return nil
}

func (drv *SESXiGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iVm, err := guest.GetIVM(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "GetIVM")
		}

		err = iVm.DeleteVM(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "DeleteVM")
		}

		return nil, cloudprovider.WaitDeleted(iVm, time.Second*10, time.Minute*3)
	})
	return nil
}

func (drv *SESXiGuestDriver) ValidateGuestHotChangeConfigInput(ctx context.Context, guest *models.SGuest, confs *api.ServerChangeConfigSettings) (*api.ServerChangeConfigSettings, error) {
	// cannot chagne esxi VM CPU cores per sockets
	corePerSocket := guest.VcpuCount / guest.CpuSockets
	if confs.VcpuCount%corePerSocket != 0 {
		return confs, errors.Wrapf(httperrors.ErrInputParameter, "cpu count %d should be times of %d", confs.VcpuCount, corePerSocket)
	}
	confs.CpuSockets = confs.VcpuCount / corePerSocket

	// https://kb.vmware.com/s/article/2008405
	// cannot increase memory beyond 3G if the initial CPU memory is lower than 3G
	startVmem := guest.VmemSize
	vmemMbStr := guest.GetMetadata(ctx, api.VM_METADATA_START_VMEM_MB, nil)
	if len(vmemMbStr) > 0 {
		vmemMb, _ := strconv.Atoi(vmemMbStr)
		if vmemMb > 0 {
			startVmem = int(vmemMb)
		}
	}
	maxAllowVmem := 16 * startVmem
	if startVmem <= 3*1024 {
		maxAllowVmem = 3 * 1024
	}
	if confs.VmemSize > maxAllowVmem {
		return confs, errors.Wrapf(httperrors.ErrInputParameter, "memory cannot be resized beyond %dMB", maxAllowVmem)
	}
	return confs, nil
}

func (esxi *SESXiGuestDriver) ValidateGuestChangeConfigInput(ctx context.Context, guest *models.SGuest, input api.ServerChangeConfigInput) (*api.ServerChangeConfigSettings, error) {
	confs, err := esxi.SBaseGuestDriver.ValidateGuestChangeConfigInput(ctx, guest, input)
	if err != nil {
		return nil, errors.Wrap(err, "SBaseGuestDriver.ValidateGuestChangeConfigInput")
	}

	if input.CpuSockets != nil && *input.CpuSockets > 0 {
		confs.CpuSockets = *input.CpuSockets
	}

	defaultStorageId := ""
	if root, _ := guest.GetSystemDisk(); root != nil {
		defaultStorageId = root.StorageId
	}
	storages, err := guest.GetStorages()
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorages")
	}
	storageMap := map[string]string{}
	for _, storage := range storages {
		storageMap[storage.StorageType] = storage.Id
		if len(defaultStorageId) == 0 {
			defaultStorageId = storage.Id
		}
	}
	for i := range confs.Create {
		confs.Create[i].Format = "vmdk"
		if len(confs.Create[i].Storage) == 0 {
			// 若不指定存储类型，默认和系统盘一致
			if len(confs.Create[i].Backend) == 0 {
				confs.Create[i].Storage = defaultStorageId
			} else if storageId, ok := storageMap[confs.Create[i].Backend]; ok { // 否则和已有磁盘存储保持一致
				confs.Create[i].Storage = storageId
			}
		}
	}
	return confs, nil
}
