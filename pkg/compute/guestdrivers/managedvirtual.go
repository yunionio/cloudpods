package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SManagedVirtualizedGuestDriver struct {
	SVirtualizedGuestDriver
}

func (self *SManagedVirtualizedGuestDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	config := cloudprovider.SManagedVMCreateConfig{}
	config.Name = guest.Name
	config.Cpu = int(guest.VcpuCount)
	config.MemoryMB = guest.VmemSize
	config.Description = guest.Description

	config.InstanceType = guest.InstanceType

	if len(guest.KeypairId) > 0 {
		config.PublicKey = guest.GetKeypairPublicKey()
	}

	nics, _ := guest.GetNetworks("")
	if len(nics) > 0 {
		net := nics[0].GetNetwork()
		config.ExternalNetworkId = net.ExternalId
		config.IpAddr = nics[0].IpAddr
	}

	disks := guest.GetDisks()
	config.DataDisks = []cloudprovider.SDiskInfo{}

	for i := 0; i < len(disks); i += 1 {
		disk := disks[i].GetDisk()
		storage := disk.GetStorage()
		if i == 0 {
			config.SysDisk.Name = disk.Name
			config.SysDisk.StorageType = storage.StorageType
			config.SysDisk.SizeGB = disk.DiskSize / 1024
			cache := storage.GetStoragecache()
			imageId := disk.GetTemplateId()
			//避免因同步过来的instance没有对应的imagecache信息，重置密码时引发空指针访问
			if scimg := models.StoragecachedimageManager.GetStoragecachedimage(cache.Id, imageId); scimg != nil {
				config.ExternalImageId = scimg.ExternalId
				img := scimg.GetCachedimage()
				config.OsDistribution, _ = img.Info.GetString("properties", "os_distribution")
				config.OsVersion, _ = img.Info.GetString("properties", "os_version")
				config.OsType, _ = img.Info.GetString("properties", "os_type")
				config.ImageType = img.ImageType
			}
		} else {
			dataDisk := cloudprovider.SDiskInfo{
				SizeGB:      disk.DiskSize / 1024,
				StorageType: storage.StorageType,
				Name:        disk.Name,
			}
			config.DataDisks = append(config.DataDisks, dataDisk)
		}
	}

	if guest.BillingType == models.BILLING_TYPE_PREPAID {
		bc, err := billing.ParseBillingCycle(guest.BillingCycle)
		if err != nil {
			log.Errorf("fail to parse billing cycle %s: %s", guest.BillingCycle, err)
		}
		if bc.IsValid() {
			config.BillingCycle = &bc
		}
	}

	return jsonutils.Marshal(&config)
}

func (self *SManagedVirtualizedGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	diskCat := guest.CategorizeDisks()
	var imageId string
	if diskCat.Root != nil {
		imageId = diskCat.Root.GetTemplateId()
	}
	if len(imageId) == 0 {
		task.ScheduleRun(nil)
		return nil
	}
	storage := diskCat.Root.GetStorage()
	if storage == nil {
		return fmt.Errorf("no valid storage")
	}
	storageCache := storage.GetStoragecache()
	if storageCache == nil {
		return fmt.Errorf("no valid storage cache")
	}
	return storageCache.StartImageCacheTask(ctx, task.GetUserCred(), imageId, diskCat.Root.DiskFormat, false, task.GetTaskId())
}

func (self *SManagedVirtualizedGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SManagedVirtualizedGuestDriver) RequestAttachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

// func (self *SManagedVirtualizedGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
// 	return nil
// }

func (self *SManagedVirtualizedGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	ihost, e := host.GetIHost()
	if e != nil {
		return nil, e
	}

	ivm, e := ihost.GetIVMById(guest.GetExternalId())
	if e != nil {
		return nil, e
	}

	result := jsonutils.NewDict()
	if ivm.GetStatus() != models.VM_RUNNING {
		if err := ivm.StartVM(ctx); err != nil {
			return nil, e
		} else {
			task.ScheduleRun(result)
		}
	} else {
		result.Add(jsonutils.NewBool(true), "is_running")
	}

	return result, e
}

func (self *SManagedVirtualizedGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	log.Debugf("RequestDeployGuestOnHost: %s", config)

	desc := cloudprovider.SManagedVMCreateConfig{}
	if err := desc.GetConfig(config); err != nil {
		return err
	}

	action, err := config.GetString("action")
	if err != nil {
		return err
	}

	ihost, err := host.GetIHost()
	if err != nil {
		return err
	}

	switch action {
	case "create":
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return guest.GetDriver().RemoteDeployGuestForCreate(ctx, task.GetUserCred(), guest, host, desc)
		})
	case "deploy":
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return guest.GetDriver().RemoteDeployGuestForDeploy(ctx, guest, ihost, task, desc)
		})
	case "rebuild":
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return guest.GetDriver().RemoteDeployGuestForRebuildRoot(ctx, guest, ihost, task, desc)
		})
	default:
		log.Errorf("RequestDeployGuestOnHost: Action %s not supported", action)
		return fmt.Errorf("Action %s not supported", action)
	}
	return nil
}

func (self *SManagedVirtualizedGuestDriver) GetGuestInitialStateAfterCreate() string {
	return models.VM_READY
}

func (self *SManagedVirtualizedGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return models.VM_READY
}

func (self *SManagedVirtualizedGuestDriver) GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string {
	return "root"
}

func (self *SManagedVirtualizedGuestDriver) RemoteDeployGuestForCreate(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error) {
	if len(desc.Password) == 0 {
		//Azure创建必须要设置密码
		desc.Password = seclib2.RandomPassword2(12)
	}

	ihost, _ := host.GetIHost()

	iVM, err := func() (cloudprovider.ICloudVM, error) {
		lockman.LockObject(ctx, host)
		defer lockman.ReleaseObject(ctx, host)

		iVM, err := ihost.CreateVM(&desc)
		if err != nil {
			return nil, err
		}
		guest.SetExternalId(userCred, iVM.GetGlobalId())

		return iVM, nil
	}()

	if err != nil {
		return nil, err
	}

	initialState := guest.GetDriver().GetGuestInitialStateAfterCreate()
	log.Debugf("VMcreated %s, wait status %s ...", iVM.GetGlobalId(), initialState)
	err = cloudprovider.WaitStatus(iVM, initialState, time.Second*5, time.Second*1800)
	if err != nil {
		return nil, err
	}
	log.Debugf("VMcreated %s, and status is running", iVM.GetGlobalId())

	iVM, err = ihost.GetIVMById(iVM.GetGlobalId())
	if err != nil {
		log.Errorf("cannot find vm %s", err)
		return nil, err
	}

	err = cloudprovider.RetryUntil(func() (bool, error) {
		idisks, err := iVM.GetIDisks()
		if err != nil {
			log.Errorf("fail to find vm disks %s after create", err)
			return false, err
		}
		if len(idisks) == len(desc.DataDisks)+1 {
			return true, nil
		} else {
			return false, nil
		}
	}, 10)
	if err != nil {
		return nil, err
	}

	account := guest.GetDriver().GetLinuxDefaultAccount(desc)

	data := fetchIVMinfo(desc, iVM, guest.Id, account, desc.Password, "create")
	return data, nil
}

func (self *SManagedVirtualizedGuestDriver) RemoteDeployGuestForDeploy(ctx context.Context, guest *models.SGuest, ihost cloudprovider.ICloudHost, task taskman.ITask, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error) {
	iVM, err := ihost.GetIVMById(guest.GetExternalId())
	if err != nil || iVM == nil {
		log.Errorf("cannot find vm %s", err)
		return nil, fmt.Errorf("cannot find vm")
	}

	params := task.GetParams()
	log.Debugf("Deploy VM params %s", params.String())

	deleteKeypair := jsonutils.QueryBoolean(params, "__delete_keypair__", false)

	if len(desc.UserData) > 0 {
		err := iVM.UpdateUserData(desc.UserData)
		if err != nil {
			log.Errorf("update userdata fail %s", err)
		}
	}

	err = func() error {
		lockman.LockObject(ctx, guest)
		defer lockman.ReleaseObject(ctx, guest)

		return iVM.DeployVM(ctx, desc.Name, desc.Password, desc.PublicKey, deleteKeypair, desc.Description)
	}()
	if err != nil {
		return nil, err
	}

	account := guest.GetDriver().GetLinuxDefaultAccount(desc)
	data := fetchIVMinfo(desc, iVM, guest.Id, account, desc.Password, "deploy")

	return data, nil
}

func (self *SManagedVirtualizedGuestDriver) RemoteDeployGuestForRebuildRoot(ctx context.Context, guest *models.SGuest, ihost cloudprovider.ICloudHost, task taskman.ITask, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error) {
	iVM, err := ihost.GetIVMById(guest.GetExternalId())
	if err != nil || iVM == nil {
		log.Errorf("cannot find vm %s", err)
		return nil, fmt.Errorf("cannot find vm")
	}

	if len(desc.UserData) > 0 {
		err := iVM.UpdateUserData(desc.UserData)
		if err != nil {
			log.Errorf("update userdata fail %s", err)
		}
	}

	diskId, err := func() (string, error) {
		lockman.LockObject(ctx, guest)
		defer lockman.ReleaseObject(ctx, guest)

		return iVM.RebuildRoot(ctx, desc.ExternalImageId, desc.Password, desc.PublicKey, desc.SysDisk.SizeGB)
	}()
	if err != nil {
		return nil, err
	}

	initialState := guest.GetDriver().GetGuestInitialStateAfterRebuild()
	log.Debugf("VMrebuildRoot %s new diskID %s, wait status %s ...", iVM.GetGlobalId(), diskId, initialState)
	err = cloudprovider.WaitStatus(iVM, initialState, time.Second*5, time.Second*1800)
	if err != nil {
		return nil, err
	}
	log.Debugf("VMrebuildRoot %s, and status is ready", iVM.GetGlobalId())

	maxWaitSecs := 300
	waited := 0

	for {
		// hack, wait disk number consistent
		idisks, err := iVM.GetIDisks()
		if err != nil {
			log.Errorf("fail to find VM idisks %s", err)
			return nil, err
		}
		if len(idisks) < len(desc.DataDisks)+1 {
			if waited > maxWaitSecs {
				log.Errorf("inconsistent disk number, wait timeout, must be something wrong on remote")
				return nil, cloudprovider.ErrTimeout
			}
			log.Debugf("inconsistent disk number???? %d != %d", len(idisks), len(desc.DataDisks)+1)
			time.Sleep(time.Second * 5)
			waited += 5
		} else {
			if idisks[0].GetGlobalId() != diskId {
				log.Errorf("system disk id inconsistent %s != %s", idisks[0].GetGlobalId(), diskId)
				return nil, fmt.Errorf("inconsistent sys disk id after rebuild root")
			}

			break
		}
	}

	account := guest.GetDriver().GetLinuxDefaultAccount(desc)
	data := fetchIVMinfo(desc, iVM, guest.Id, account, desc.Password, "rebuild")

	return data, nil
}

func (self *SManagedVirtualizedGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost()
		if err != nil {
			log.Errorf("host.GetIHost fail %s", err)
			return nil, err
		}

		// 创建失败时external id为空。此时直接返回即可。不需要再调用公有云api
		if len(guest.ExternalId) == 0 {
			return nil, nil
		}

		ivm, err := ihost.GetIVMById(guest.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			log.Errorf("ihost.GetIVMById fail %s", err)
			return nil, err
		}
		err = ivm.DeleteVM(ctx)
		if err != nil {
			log.Errorf("ivm.DeleteVM fail %s", err)
			return nil, err
		}

		for _, guestdisk := range guest.GetDisks() {
			if disk := guestdisk.GetDisk(); disk != nil && disk.AutoDelete {
				idisk, err := disk.GetIDisk()
				if err != nil {
					if err == cloudprovider.ErrNotFound {
						continue
					}
					log.Errorf("disk.GetIDisk fail %s", err)
					return nil, err
				}
				err = idisk.Delete(ctx)
				if err != nil {
					log.Errorf("idisk.Delete fail %s", err)
					return nil, err
				}
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost()
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.ExternalId)
		if err != nil {
			return nil, err
		}
		err = ivm.StopVM(ctx, true)
		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	ihost, err := host.GetIHost()
	if err != nil {
		return nil, err
	}
	ivm, err := ihost.GetIVMById(guest.ExternalId)
	if err != nil {
		log.Errorf("fail to find ivm by id %s", err)
		return nil, err
	}

	status := ivm.GetStatus()
	switch status {
	case models.VM_RUNNING:
		status = cloudprovider.CloudVMStatusRunning
	case models.VM_READY:
		status = cloudprovider.CloudVMStatusStopped
	case models.VM_STARTING:
		status = cloudprovider.CloudVMStatusStopped
	case models.VM_STOPPING:
		status = cloudprovider.CloudVMStatusRunning
	case models.VM_CHANGE_FLAVOR:
		status = cloudprovider.CloudVMStatusChangeFlavor
	case models.VM_DEPLOYING:
		status = cloudprovider.CloudVMStatusDeploying
	default:
		status = cloudprovider.CloudVMStatusOther
	}

	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(status), "status")
	return body, nil
}

func (self *SManagedVirtualizedGuestDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	ihost, err := host.GetIHost()
	if err != nil {
		return nil, err
	}

	iVM, err := ihost.GetIVMById(guest.ExternalId)
	if err != nil {
		log.Errorf("cannot find vm %s %s", iVM, err)
		return nil, err
	}

	data, err := iVM.GetVNCInfo()
	if err != nil {
		return nil, err
	}

	dataDict := data.(*jsonutils.JSONDict)

	return dataDict, nil
}

func (self *SManagedVirtualizedGuestDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ManagedGuestRebuildRootTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SManagedVirtualizedGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ManagedGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

type SManagedVMChangeConfig struct {
	InstanceId   string
	InstanceType string // InstanceType 不为空时，直接采用InstanceType更新主机。
	Cpu          int
	Memory       int
}

func (self *SManagedVirtualizedGuestDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, instanceType string, vcpuCount, vmemSize int64) error {
	ihost, err := guest.GetHost().GetIHost()
	if err != nil {
		return err
	}

	iVM, err := ihost.GetIVMById(guest.GetExternalId())
	if err != nil {
		return err
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if len(instanceType) > 0 {
			return nil, iVM.ChangeConfig2(ctx, instanceType)
		} else {
			return nil, iVM.ChangeConfig(ctx, int(vcpuCount), int(vmemSize))
		}
	})

	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, snapshotId, diskId string) error {
	iDisk, _ := models.DiskManager.FetchById(diskId)
	disk := iDisk.(*models.SDisk)
	providerDisk, err := disk.GetIDisk()
	if err != nil {
		return err
	}
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		cloudSnapshot, err := providerDisk.CreateISnapshot(ctx, snapshot.Name, "")
		if err != nil {
			return nil, err
		}
		res := jsonutils.NewDict()
		res.Set("snapshot_id", jsonutils.NewString(cloudSnapshot.GetId()))
		return res, nil
	})
	return nil
}

func (self *SManagedVirtualizedGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {

	uuid, _ := data.GetString("uuid")
	if len(uuid) > 0 {
		guest.SetExternalId(task.GetUserCred(), uuid)
	}

	recycle := false
	if guest.IsPrepaidRecycle() {
		recycle = true
	}

	if data.Contains("disks") {
		diskInfo := make([]SDiskInfo, 0)
		err := data.Unmarshal(&diskInfo, "disks")
		if err != nil {
			return err
		}

		disks := guest.GetDisks()
		if len(disks) != len(diskInfo) {
			msg := fmt.Sprintf("inconsistent disk number: guest have %d disks, data contains %d disks", len(disks), len(diskInfo))
			log.Errorf(msg)
			return fmt.Errorf(msg)
		}
		for i := 0; i < len(diskInfo); i += 1 {
			disk := disks[i].GetDisk()
			_, err = db.Update(disk, func() error {
				disk.DiskSize = diskInfo[i].Size
				disk.ExternalId = diskInfo[i].Uuid
				disk.DiskType = diskInfo[i].DiskType
				disk.Status = models.DISK_READY

				disk.FsFormat = diskInfo[i].FsFromat
				if diskInfo[i].AutoDelete {
					disk.AutoDelete = true
				}
				// disk.TemplateId = diskInfo[i].TemplateId
				disk.AccessPath = diskInfo[i].Path

				if !recycle {
					disk.BillingType = diskInfo[i].BillingType
					disk.ExpiredAt = diskInfo[i].ExpiredAt
				}

				if len(diskInfo[i].Metadata) > 0 {
					for key, value := range diskInfo[i].Metadata {
						if err := disk.SetMetadata(ctx, key, value, task.GetUserCred()); err != nil {
							log.Errorf("set disk %s mata %s => %s error: %v", disk.Name, key, value, err)
						}
					}
				}
				return nil
			})
			if err != nil {
				msg := fmt.Sprintf("save disk info failed %s", err)
				log.Errorf(msg)
				break
			}
			db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(ctx), task.GetUserCred())
			guestdisk := guest.GetGuestDisk(disk.Id)
			_, err = db.Update(guestdisk, func() error {
				guestdisk.Driver = diskInfo[i].Driver
				guestdisk.CacheMode = diskInfo[i].CacheMode
				return nil
			})
			if err != nil {
				msg := fmt.Sprintf("save disk info failed %s", err)
				log.Errorf(msg)
				break
			}
		}
	}

	if metaData, _ := data.Get("metadata"); metaData != nil {
		meta := make(map[string]string, 0)
		if err := metaData.Unmarshal(meta); err != nil {
			log.Errorf("Get guest %s metadata error: %v", guest.Name, err)
		} else {
			for key, value := range meta {
				if err := guest.SetMetadata(ctx, key, value, task.GetUserCred()); err != nil {
					log.Errorf("set guest %s mata %s => %s error: %v", guest.Name, key, value, err)
				}
			}
		}
	}

	exp, err := data.GetTime("expired_at")
	if err == nil && !guest.IsPrepaidRecycle() {
		guest.SaveRenewInfo(ctx, task.GetUserCred(), nil, &exp)
	}

	guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost()
		if err != nil {
			return nil, err
		}
		iVM, err := ihost.GetIVMById(guest.ExternalId)
		if err != nil {
			return nil, err
		}

		if fwOnly, _ := task.GetParams().Bool("fw_only"); fwOnly {
			vpcId := ""
			guestnets, err := guest.GetNetworks("")
			if err != nil {
				return nil, err
			}
			for _, network := range guestnets {
				if vpc := network.GetNetwork().GetVpc(); vpc != nil {
					vpcId = vpc.ExternalId
					break
				}
			}
			iregion, err := host.GetIRegion()
			if err != nil {
				return nil, err
			}
			secgroups := guest.GetSecgroups()
			externalIds := []string{}
			for _, secgroup := range secgroups {
				lockman.LockRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-%s", guest.SecgrpId, vpcId))
				defer lockman.ReleaseRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-%s", guest.SecgrpId, vpcId))

				secgroupCache := models.SecurityGroupCacheManager.Register(ctx, task.GetUserCred(), secgroup.Id, vpcId, host.GetRegion().Id, host.ManagerId)
				if secgroupCache == nil {
					return nil, fmt.Errorf("failed to registor secgroupCache for secgroup: %s vpc: %s", secgroup.Id, vpcId)
				}
				extID, err := iregion.SyncSecurityGroup(secgroupCache.ExternalId, vpcId, secgroup.Name, secgroup.Description, secgroup.GetSecRules(""))
				if err != nil {
					return nil, err
				}
				if err = secgroupCache.SetExternalId(task.GetUserCred(), extID); err != nil {
					return nil, err
				}
				externalIds = append(externalIds, extID)
			}
			return nil, iVM.SetSecurityGroups(externalIds)
		}

		iDisks, err := iVM.GetIDisks()
		if err != nil {
			return nil, err
		}
		disks := make([]models.SDisk, 0)
		for _, guestdisk := range guest.GetDisks() {
			disk := guestdisk.GetDisk()
			disks = append(disks, *disk)
		}

		added := make([]models.SDisk, 0)
		commondb := make([]models.SDisk, 0)
		commonext := make([]cloudprovider.ICloudDisk, 0)
		removed := make([]cloudprovider.ICloudDisk, 0)

		if err := compare.CompareSets(disks, iDisks, &added, &commondb, &commonext, &removed); err != nil {
			return nil, err
		}
		for _, disk := range removed {
			if err := iVM.DetachDisk(ctx, disk.GetId()); err != nil {
				return nil, err
			}
		}
		for _, disk := range added {
			if err := iVM.AttachDisk(ctx, disk.ExternalId); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return nil
}

/*func (self *SManagedVirtualizedGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if ihost, err := host.GetIHost(); err != nil {
			return nil, err
		} else if iVM, err := ihost.GetIVMById(guest.ExternalId); err != nil {
			return nil, err
		} else {
			if fw_only, _ := task.GetParams().Bool("fw_only"); fw_only {
				if err := iVM.SyncSecurityGroup(guest.SecgrpId, guest.GetSecgroupName(), guest.GetSecRules()); err != nil {
					return nil, err
				}
			} else {
				if iDisks, err := iVM.GetIDisks(); err != nil {
					return nil, err
				} else {
					disks := make([]models.SDisk, 0)
					for _, guestdisk := range guest.GetDisks() {
						disk := guestdisk.GetDisk()
						disks = append(disks, *disk)
					}

					added := make([]models.SDisk, 0)
					commondb := make([]models.SDisk, 0)
					commonext := make([]cloudprovider.ICloudDisk, 0)
					removed := make([]cloudprovider.ICloudDisk, 0)

					if err := compare.CompareSets(disks, iDisks, &added, &commondb, &commonext, &removed); err != nil {
						return nil, err
					}
					for _, disk := range removed {
						if err := iVM.DetachDisk(ctx, disk.GetId()); err != nil {
							return nil, err
						}
					}
					for _, disk := range added {
						if err := iVM.AttachDisk(ctx, disk.ExternalId); err != nil {
							return nil, err
						}
					}
				}
			}
		}
		return nil, nil
	})
	return nil
}*/

func (self *SManagedVirtualizedGuestDriver) RequestRenewInstance(guest *models.SGuest, bc billing.SBillingCycle) (time.Time, error) {
	iVM, err := guest.GetIVM()
	if err != nil {
		return time.Time{}, err
	}
	err = iVM.Renew(bc)
	if err != nil {
		return time.Time{}, err
	}
	err = iVM.Refresh()
	if err != nil {
		return time.Time{}, err
	}
	return iVM.GetExpiredAt(), nil
}

func (self *SManagedVirtualizedGuestDriver) IsSupportEip() bool {
	return true
}
