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
	"math"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/cloudinit"
)

type SManagedVirtualizedGuestDriver struct {
	SVirtualizedGuestDriver
}

func (d SManagedVirtualizedGuestDriver) DoScheduleCPUFilter() bool { return false }

func (d SManagedVirtualizedGuestDriver) DoScheduleSKUFilter() bool { return true }

func (d SManagedVirtualizedGuestDriver) DoScheduleMemoryFilter() bool { return false }

func (d SManagedVirtualizedGuestDriver) DoScheduleStorageFilter() bool { return false }

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
			config.SysDisk.StorageExternalId = storage.ExternalId
			config.SysDisk.StorageType = storage.StorageType
			config.SysDisk.SizeGB = int(math.Ceil(float64(disk.DiskSize) / 1024))
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
				SizeGB:            disk.DiskSize / 1024,
				StorageType:       storage.StorageType,
				StorageExternalId: storage.ExternalId,
				Name:              disk.Name,
			}
			config.DataDisks = append(config.DataDisks, dataDisk)
		}
	}

	if guest.BillingType == billing_api.BILLING_TYPE_PREPAID {
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

func (self *SManagedVirtualizedGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if input.Cdrom != "" {
		return nil, httperrors.NewInputParameterError("%s not support cdrom params", input.Hypervisor)
	}
	return input, nil
}

func (self *SManagedVirtualizedGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iVM, err := guest.GetIVM()
		if err != nil {
			//若guest被删除,忽略错误，否则会无限删除guest失败(有挂载的云盘)
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "guest.GetIVM")
		}
		if len(disk.ExternalId) == 0 {
			return nil, nil
		}

		_, err = disk.GetIDisk()
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			//忽略云上磁盘已经被删除错误
			return nil, nil
		}

		err = iVM.DetachDisk(ctx, disk.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "iVM.DetachDisk")
		}

		err = cloudprovider.Wait(time.Second*5, time.Minute*3, func() (bool, error) {
			err := iVM.Refresh()
			if err != nil {
				return false, errors.Wrapf(err, "iVM.Refresh")
			}
			iDisks, err := iVM.GetIDisks()
			if err != nil {
				return false, errors.Wrapf(err, "RequestDetachDisk.iVM.GetIDisks")
			}

			exist := false
			for i := 0; i < len(iDisks); i++ {
				if iDisks[i].GetGlobalId() == disk.ExternalId {
					exist = true
				}
			}

			if !exist {
				return true, nil
			}
			return false, nil
		})

		if err != nil {
			return nil, errors.Wrapf(err, "RequestDetachDisk.Wait")
		}

		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestAttachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iVM, err := guest.GetIVM()
		if err != nil {
			return nil, errors.Wrapf(err, "guest.GetIVM")
		}
		if len(disk.ExternalId) == 0 {
			return nil, fmt.Errorf("disk %s(%s) is not a managed resource", disk.Name, disk.Id)
		}
		err = iVM.AttachDisk(ctx, disk.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "iVM.AttachDisk")
		}

		err = cloudprovider.Wait(time.Second*5, time.Minute*3, func() (bool, error) {
			err := iVM.Refresh()
			if err != nil {
				return false, errors.Wrapf(err, "iVM.Refresh")
			}

			iDisks, err := iVM.GetIDisks()
			if err != nil {
				return false, errors.Wrapf(err, "RequestAttachDisk.iVM.GetIDisks")
			}

			for i := 0; i < len(iDisks); i++ {
				if iDisks[i].GetGlobalId() == disk.ExternalId {
					err := cloudprovider.WaitStatus(iDisks[i], api.DISK_READY, 5*time.Second, 60*time.Second)
					if err != nil {
						return false, errors.Wrapf(err, "RequestAttachDisk.iVM.WaitStatus")
					}

					return true, nil
				}
			}

			return false, nil
		})

		if err != nil {
			return nil, errors.Wrapf(err, "RequestAttachDisk.Wait")
		}

		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	ihost, err := host.GetIHost()
	if err != nil {
		return nil, err
	}

	ivm, err := ihost.GetIVMById(guest.GetExternalId())
	if err != nil {
		return nil, err
	}

	result := jsonutils.NewDict()
	if ivm.GetStatus() != api.VM_RUNNING {
		err := ivm.StartVM(ctx)
		if err != nil {
			return nil, err
		}
		task.ScheduleRun(result)
	} else {
		result.Add(jsonutils.NewBool(true), "is_running")
	}

	return result, nil
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

	//创建并同步安全组规则
	{
		vpc, err := guest.GetVpc()
		if err != nil {
			return errors.Wrap(err, "guest.GetVpc")
		}
		region, err := vpc.GetRegion()
		if err != nil {
			return errors.Wrap(err, "vpc.GetRegion")
		}

		vpcId, err := region.GetDriver().GetSecurityGroupVpcId(ctx, task.GetUserCred(), region, host, vpc, false)
		if err != nil {
			return errors.Wrap(err, "GetSecurityGroupVpcId")
		}

		secgroups := guest.GetSecgroups()
		for i, secgroup := range secgroups {
			externalId, err := region.GetDriver().RequestSyncSecurityGroup(ctx, task.GetUserCred(), vpcId, vpc, &secgroup)
			if err != nil {
				return errors.Wrap(err, "RequestSyncSecurityGroup")
			}

			desc.ExternalSecgroupIds = append(desc.ExternalSecgroupIds, externalId)
			if i == 0 {
				desc.ExternalSecgroupId = externalId
			}
		}
	}

	desc.Account = guest.GetDriver().GetLinuxDefaultAccount(desc)

	if guest.GetDriver().IsNeedInjectPasswordByCloudInit(&desc) {
		err = desc.InjectPasswordByCloudInit()
		if err != nil {
			log.Warningf("failed to inject password by cloud-init error: %v", err)
		}
	}

	if len(desc.UserData) > 0 {
		oUserData, err := cloudinit.ParseUserData(desc.UserData)
		if err != nil {
			return err
		}
		switch guest.GetDriver().GetUserDataType() {
		case cloudprovider.CLOUD_SHELL:
			desc.UserData = oUserData.UserDataScriptBase64()
		case cloudprovider.CLOUD_SHELL_WITHOUT_ENCRYPT:
			desc.UserData = oUserData.UserDataScript()
		default:
			desc.UserData = oUserData.UserDataBase64()
		}
		if strings.ToLower(desc.OsType) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) {
			desc.UserData = oUserData.UserDataPowerShell()
		}
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
		region := host.GetRegion()
		if len(desc.InstanceType) == 0 && region != nil && utils.IsInStringArray(guest.Hypervisor, api.PUBLIC_CLOUD_HYPERVISORS) {
			sku, err := models.ServerSkuManager.GetMatchedSku(region.GetId(), int64(desc.Cpu), int64(desc.MemoryMB))
			if err != nil {
				return errors.Wrap(err, "ManagedVirtualizedGuestDriver.RequestDeployGuestOnHost.GetMatchedSku")
			}

			if sku == nil {
				return errors.Wrap(errors.ErrNotFound, "ManagedVirtualizedGuestDriver.RequestDeployGuestOnHost.GetMatchedSku")
			}

			desc.InstanceType = sku.Name
		}

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
	return api.VM_READY
}

func (self *SManagedVirtualizedGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SManagedVirtualizedGuestDriver) GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string {
	userName := "root"
	if strings.ToLower(desc.OsType) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) {
		userName = "Administrator"
	}
	return userName
}

func (self *SManagedVirtualizedGuestDriver) RemoteDeployGuestForCreate(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error) {
	ihost, err := host.GetIHost()
	if err != nil {
		return nil, errors.Wrapf(err, "RemoteDeployGuestForCreate.GetIHost")
	}

	iVM, err := func() (cloudprovider.ICloudVM, error) {
		lockman.LockObject(ctx, host)
		defer lockman.ReleaseObject(ctx, host)

		iVM, err := ihost.CreateVM(&desc)
		if err != nil {
			return nil, err
		}
		db.SetExternalId(guest, userCred, iVM.GetGlobalId())
		err = iVM.SetSecurityGroups(desc.ExternalSecgroupIds)
		if err != nil {
			log.Errorf("failed to set multi secgroup for instance %s error: %v", guest.Name, err)
		}

		if hostId := iVM.GetIHostId(); len(hostId) > 0 {
			host, err := db.FetchByExternalId(models.HostManager, hostId)
			if err != nil {
				log.Warningf("failed to found new hostId(%s) for ivm %s(%s) error: %v", hostId, guest.Name, guest.Id, err)
			} else if host.GetId() != guest.HostId {
				guest.OnScheduleToHost(ctx, userCred, host.GetId())
			}
		}

		return iVM, nil
	}()

	if err != nil {
		return nil, err
	}

	initialState := guest.GetDriver().GetGuestInitialStateAfterCreate()
	log.Debugf("VMcreated %s, wait status %s ...", iVM.GetGlobalId(), initialState)
	err = cloudprovider.WaitStatusWithInstanceErrorCheck(iVM, initialState, time.Second*5, time.Second*1800, func() error {
		return iVM.GetError()
	})
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
		return nil, errors.Wrap(err, "GuestDriver.RemoteDeployGuestForCreate.RetryUntil")
	}

	guest.GetDriver().RemoteActionAfterGuestCreated(ctx, userCred, guest, host, iVM, &desc)

	data := fetchIVMinfo(desc, iVM, guest.Id, desc.Account, desc.Password, desc.PublicKey, "create")
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

		// 避免DeployVM函数里面执行顺序不一致导致与预期结果不符
		if deleteKeypair {
			desc.Password, desc.PublicKey = "", ""
		}

		if len(desc.PublicKey) > 0 {
			desc.Password = ""
		}

		e := iVM.DeployVM(ctx, desc.Name, desc.Account, desc.Password, desc.PublicKey, deleteKeypair, desc.Description)
		if e != nil {
			return e
		}

		if len(desc.Password) == 0 {
			//可以从秘钥解密旧密码
			desc.Password = guest.GetOldPassword(ctx, task.GetUserCred())
		}
		return nil
	}()
	if err != nil {
		return nil, err
	}

	data := fetchIVMinfo(desc, iVM, guest.Id, desc.Account, desc.Password, desc.PublicKey, "deploy")

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

		conf := cloudprovider.SManagedVMRebuildRootConfig{
			Account:   desc.Account,
			ImageId:   desc.ExternalImageId,
			Password:  desc.Password,
			PublicKey: desc.PublicKey,
			SysSizeGB: desc.SysDisk.SizeGB,
			OsType:    desc.OsType,
		}
		return iVM.RebuildRoot(ctx, &conf)
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
			if idisks[0].GetGlobalId() == diskId {
				break
			}
			if waited > maxWaitSecs {
				return nil, fmt.Errorf("inconsistent sys disk id after rebuild root")
			}
			log.Debugf("current system disk id inconsistent %s != %s, try after 5 seconds", idisks[0].GetGlobalId(), diskId)
			time.Sleep(time.Second * 5)
			waited += 5
		}
	}

	data := fetchIVMinfo(desc, iVM, guest.Id, desc.Account, desc.Password, desc.PublicKey, "rebuild")

	return data, nil
}

func (self *SManagedVirtualizedGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost()
		if err != nil {
			//私有云宿主机有可能下线,会导致虚拟机无限删除失败
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			log.Errorf("host.GetIHost fail %s", err)
			return nil, err
		}

		// 创建失败时external id为空。此时直接返回即可。不需要再调用公有云api
		if len(guest.ExternalId) == 0 {
			return nil, nil
		}

		ivm, err := ihost.GetIVMById(guest.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
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
					if errors.Cause(err) == cloudprovider.ErrNotFound {
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

	err = guest.SyncAllWithCloudVM(ctx, userCred, host, ivm)
	if err != nil {
		return nil, err
	}

	status := GetCloudVMStatus(ivm)
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

func (self *SManagedVirtualizedGuestDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, instanceType string, vcpuCount, vmemSize int64) error {
	host := guest.GetHost()
	ihost, err := host.GetIHost()
	if err != nil {
		return err
	}

	iVM, err := ihost.GetIVMById(guest.GetExternalId())
	if err != nil {
		return err
	}

	if len(instanceType) == 0 {
		sku, err := models.ServerSkuManager.GetMatchedSku(host.GetRegion().GetId(), vcpuCount, vmemSize)
		if err != nil {
			return errors.Wrap(err, "ManagedVirtualizedGuestDriver.RequestChangeVmConfig.GetMatchedSku")
		}

		if sku == nil {
			return errors.Wrap(errors.ErrNotFound, "ManagedVirtualizedGuestDriver.RequestChangeVmConfig.GetMatchedSku")
		}

		instanceType = sku.Name
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		config := &cloudprovider.SManagedVMChangeConfig{
			Cpu:          int(vcpuCount),
			MemoryMB:     int(vmemSize),
			InstanceType: instanceType,
		}
		err := iVM.ChangeConfig(ctx, config)
		if err != nil {
			return nil, errors.Wrap(err, "GuestDriver.RequestChangeVmConfig.ChangeConfig")
		}

		err = cloudprovider.WaitCreated(time.Second*5, time.Minute*5, func() bool {
			err := iVM.Refresh()
			if err != nil {
				return false
			}
			status := iVM.GetStatus()
			if status == api.VM_READY || status == api.VM_RUNNING {
				iInstanceType := iVM.GetInstanceType()
				if len(instanceType) > 0 && len(iInstanceType) > 0 && instanceType == iInstanceType {
					return true
				} else {
					// aws 目前取不到内存。返回值永远为0
					if iVM.GetVcpuCount() == int(vcpuCount) && (iVM.GetVmemSizeMB() == int(vmemSize) || iVM.GetVmemSizeMB() == 0) {
						return true
					}
				}
			}
			return false
		})
		if err != nil {
			return nil, errors.Wrap(err, "GuestDriver.RequestChangeVmConfig.WaitCreated")
		}

		instanceType = iVM.GetInstanceType()
		if len(instanceType) > 0 {
			_, err = db.Update(guest, func() error {
				guest.InstanceType = instanceType
				return nil
			})
			if err != nil {
				return nil, errors.Wrap(err, "GuestDriver.RequestChangeVmConfig.Update")
			}
		}

		return nil, nil
	})

	return nil
}

func (self *SManagedVirtualizedGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {

	uuid, _ := data.GetString("uuid")
	if len(uuid) > 0 {
		db.SetExternalId(guest, task.GetUserCred(), uuid)
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
				disk.Status = api.DISK_READY

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

				if len(diskInfo[i].StorageExternalId) > 0 {
					storage, err := db.FetchByExternalId(models.StorageManager, diskInfo[i].StorageExternalId)
					if err != nil {
						log.Warningf("failed to found storage by externalId %s error: %v", diskInfo[i].StorageExternalId, err)
					} else if disk.StorageId != storage.GetId() {
						disk.StorageId = storage.GetId()
					}
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
		guest.SaveRenewInfo(ctx, task.GetUserCred(), nil, &exp, "")
	}

	guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestSyncSecgroupsOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	iVM, err := guest.GetIVM()
	if err != nil {
		return err
	}

	vpc, err := guest.GetVpc()
	if err != nil {
		return errors.Wrap(err, "guest.GetVpc")
	}

	region := host.GetRegion()

	vpcId, err := region.GetDriver().GetSecurityGroupVpcId(ctx, task.GetUserCred(), region, host, vpc, false)
	if err != nil {
		return errors.Wrap(err, "GetSecurityGroupVpcId")
	}

	secgroups := guest.GetSecgroups()
	externalIds := []string{}
	for _, secgroup := range secgroups {
		externalId, err := region.GetDriver().RequestSyncSecurityGroup(ctx, task.GetUserCred(), vpcId, vpc, &secgroup)
		if err != nil {
			return errors.Wrap(err, "RequestSyncSecurityGroup")
		}
		externalIds = append(externalIds, externalId)
	}
	return iVM.SetSecurityGroups(externalIds)
}

func (self *SManagedVirtualizedGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		if jsonutils.QueryBoolean(task.GetParams(), "fw_only", false) {
			err := guest.GetDriver().RequestSyncSecgroupsOnHost(ctx, guest, host, task)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestRenewInstance(guest *models.SGuest, bc billing.SBillingCycle) (time.Time, error) {
	iVM, err := guest.GetIVM()
	if err != nil {
		return time.Time{}, err
	}
	oldExpired := iVM.GetExpiredAt()
	err = iVM.Renew(bc)
	if err != nil {
		return time.Time{}, err
	}
	//避免有些云续费后过期时间刷新比较慢问题
	cloudprovider.WaitCreated(15*time.Second, 5*time.Minute, func() bool {
		err := iVM.Refresh()
		if err != nil {
			log.Errorf("failed refresh instance %s error: %v", guest.Name, err)
		}
		newExipred := iVM.GetExpiredAt()
		if newExipred.After(oldExpired) {
			return true
		}
		return false
	})
	return iVM.GetExpiredAt(), nil
}

func (self *SManagedVirtualizedGuestDriver) IsSupportEip() bool {
	return true
}

func (self *SManagedVirtualizedGuestDriver) RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, server *models.SGuest, eip *models.SElasticip, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if server.Status != api.VM_ASSOCIATE_EIP {
			server.SetStatus(userCred, api.VM_ASSOCIATE_EIP, "associate eip")
		}

		extEip, err := eip.GetIEip()
		if err != nil {
			return nil, fmt.Errorf("ManagedVirtualizedGuestDriver.RequestAssociateEip fail to find iEIP for eip %s", err)
		}

		conf := &cloudprovider.AssociateConfig{
			InstanceId:    server.ExternalId,
			Bandwidth:     eip.Bandwidth,
			AssociateType: api.EIP_ASSOCIATE_TYPE_SERVER,
		}

		err = extEip.Associate(conf)
		if err != nil {
			return nil, fmt.Errorf("ManagedVirtualizedGuestDriver.RequestAssociateEip fail to remote associate EIP %s", err)
		}

		err = cloudprovider.WaitStatus(extEip, api.EIP_STATUS_READY, 3*time.Second, 60*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "ManagedVirtualizedGuestDriver.RequestAssociateEip.WaitStatus")
		}

		err = eip.AssociateVM(ctx, userCred, server)
		if err != nil {
			return nil, fmt.Errorf("ManagedVirtualizedGuestDriver.RequestAssociateEip fail to local associate EIP %s", err)
		}

		eip.SetStatus(userCred, api.EIP_STATUS_READY, "associate")
		return nil, nil
	})

	return nil
}

func (self *SManagedVirtualizedGuestDriver) chooseHostStorage(
	drv models.IGuestDriver,
	host *models.SHost,
	backend string,
	storageIds []string,
) *models.SStorage {
	if len(storageIds) != 0 {
		return models.StorageManager.FetchStorageById(storageIds[0])
	}
	storages := host.GetAttachedStorages("")
	for i := 0; i < len(storages); i += 1 {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}
	for _, stype := range drv.GetStorageTypes() {
		for i := 0; i < len(storages); i += 1 {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SManagedVirtualizedGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return false, nil
}

func GetCloudVMStatus(vm cloudprovider.ICloudVM) string {
	status := vm.GetStatus()
	switch status {
	case api.VM_RUNNING:
		status = cloudprovider.CloudVMStatusRunning
	case api.VM_READY:
		status = cloudprovider.CloudVMStatusStopped
	case api.VM_STARTING:
		status = cloudprovider.CloudVMStatusStopped
	case api.VM_STOPPING:
		status = cloudprovider.CloudVMStatusStopping
	case api.VM_CHANGE_FLAVOR:
		status = cloudprovider.CloudVMStatusChangeFlavor
	case api.VM_DEPLOYING:
		status = cloudprovider.CloudVMStatusDeploying
	case api.VM_SUSPEND:
		status = cloudprovider.CloudVMStatusSuspend
	default:
		status = cloudprovider.CloudVMStatusOther
	}

	return status
}
