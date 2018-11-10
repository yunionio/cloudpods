package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SQcloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SQcloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SQcloudGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_QCLOUD
}

func (self *SQcloudGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	storages := host.GetAttachedStorages("")
	for i := 0; i < len(storages); i++ {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}
	for _, stype := range []string{"cloud_basic", "cloud_premium", "cloud_ssd", "local_basic", "local_ssd"} {
		for i := 0; i < len(storages); i++ {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SQcloudGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SQcloudGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("net.0") && data.Contains("net.1") {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	return data, nil
}

func (self *SQcloudGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())
	log.Debugf("RequestDeployGuestOnHost: %s", config)
	/* onfinish, err := config.GetString("on_finish")
	if err != nil {
		return err
	} */

	action, err := config.GetString("action")
	if err != nil {
		return err
	}

	publicKey, _ := config.GetString("public_key")

	adminPublicKey, _ := config.GetString("admin_public_key")
	projectPublicKey, _ := config.GetString("project_public_key")
	oUserData, _ := config.GetString("user_data")

	userData := generateUserData(adminPublicKey, projectPublicKey, oUserData)

	resetPassword := jsonutils.QueryBoolean(config, "reset_password", false)
	passwd, _ := config.GetString("password")
	if resetPassword && len(passwd) == 0 {
		passwd = seclib2.RandomPassword2(12)
	}

	ihost, err := host.GetIHost()
	if err != nil {
		return err
	}

	desc := SManagedVMCreateConfig{}
	err = config.Unmarshal(&desc, "desc")
	if err != nil {
		return err
	}

	if action == "create" {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

			nets := guest.GetNetworks()
			net := nets[0].GetNetwork()
			vpc := net.GetVpc()

			ivpc, err := vpc.GetIVpc()
			if err != nil {
				log.Errorf("getIVPC fail %s", err)
				return nil, err
			}

			secgrpId, err := ivpc.SyncSecurityGroup(desc.SecGroupId, desc.SecGroupName, desc.SecRules)
			if err != nil {
				log.Errorf("SyncSecurityGroup fail %s", err)
				return nil, err
			}

			iVM, err := ihost.CreateVM(desc.Name, desc.ExternalImageId, desc.SysDiskSize, desc.Cpu, desc.Memory, desc.ExternalNetworkId,
				desc.IpAddr, desc.Description, passwd, desc.StorageType, desc.DataDisks, publicKey, secgrpId, userData)
			if err != nil {
				return nil, err
			}
			log.Debugf("VMcreated %s, wait status running ...", iVM.GetGlobalId())
			err = cloudprovider.WaitStatus(iVM, models.VM_RUNNING, time.Second*5, time.Second*1800)
			if err != nil {
				return nil, err
			}
			log.Debugf("VMcreated %s, and status is ready", iVM.GetGlobalId())

			iVM, err = ihost.GetIVMById(iVM.GetGlobalId())
			if err != nil {
				log.Errorf("cannot find vm %s", err)
				return nil, err
			}
			data := fetchIVMinfo(desc, iVM, guest.Id, "root", passwd)
			return data, nil
		})
	} else if action == "deploy" {
		iVM, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil || iVM == nil {
			log.Errorf("cannot find vm %s", err)
			return fmt.Errorf("cannot find vm")
		}

		params := task.GetParams()
		log.Debugf("Deploy VM params %s", params.String())

		name, _ := params.GetString("name")
		description, _ := params.GetString("description")
		publicKey, _ := config.GetString("public_key")
		deleteKeypair := jsonutils.QueryBoolean(params, "__delete_keypair__", false)
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

			//腾讯云暂不支持更新自定义用户数据
			// if len(userData) > 0 {
			// 	err := iVM.UpdateUserData(userData)
			// 	if err != nil {
			// 		log.Errorf("update userdata fail %s", err)
			// 	}
			// }

			err := iVM.DeployVM(name, passwd, publicKey, deleteKeypair, description)
			if err != nil {
				return nil, err
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, "root", passwd)
			return data, nil
		})
	} else if action == "rebuild" {

		iVM, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil || iVM == nil {
			log.Errorf("cannot find vm %s", err)
			return fmt.Errorf("cannot find vm")
		}

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			//腾讯云暂不支持更新自定义用户数据
			// if len(userData) > 0 {
			// 	err := iVM.UpdateUserData(userData)
			// 	if err != nil {
			// 		log.Errorf("update userdata fail %s", err)
			// 	}
			// }

			diskId, err := iVM.RebuildRoot(desc.ExternalImageId, passwd, publicKey, desc.SysDiskSize)
			if err != nil {
				return nil, err
			}

			log.Debugf("VMrebuildRoot %s new diskID %s, wait status ready ...", iVM.GetGlobalId(), diskId)

			err = cloudprovider.WaitStatus(iVM, models.VM_READY, time.Second*5, time.Second*1800)
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

			data := fetchIVMinfo(desc, iVM, guest.Id, "root", passwd)

			return data, nil
		})

	} else {
		log.Errorf("RequestDeployGuestOnHost: Action %s not supported", action)
		return fmt.Errorf("Action %s not supported", action)
	}

	return nil
}

func (self *SQcloudGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {

	if data.Contains("disks") {
		diskInfo := make([]SDiskInfo, 0)
		err := data.Unmarshal(&diskInfo, "disks")
		if err != nil {
			return err
		}
		disks := guest.GetDisks()
		if len(disks) != len(diskInfo) {
			msg := fmt.Sprintf("inconsistent disk number: have %d want %d", len(disks), len(diskInfo))
			log.Errorf(msg)
			return fmt.Errorf(msg)
		}
		for i := 0; i < len(diskInfo); i += 1 {
			disk := disks[i].GetDisk()
			_, err = disk.GetModelManager().TableSpec().Update(disk, func() error {
				disk.DiskSize = diskInfo[i].Size
				disk.ExternalId = diskInfo[i].Uuid
				disk.DiskType = diskInfo[i].DiskType
				disk.Status = models.DISK_READY
				disk.BillingType = diskInfo[i].BillingType
				disk.FsFormat = diskInfo[i].FsFromat
				disk.AutoDelete = true
				disk.TemplateId = diskInfo[i].TemplateId
				disk.DiskFormat = diskInfo[i].DiskFormat
				disk.ExpiredAt = diskInfo[i].ExpiredAt
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
			} else {
				db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(), task.GetUserCred())
			}
		}
	}
	uuid, _ := data.GetString("uuid")
	if len(uuid) > 0 {
		guest.SetExternalId(uuid)
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

	guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	return nil
}

func (self *SQcloudGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SQcloudGuestDriver) RequestDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, snapshotId, diskId string) error {
	iDisk, _ := models.DiskManager.FetchById(diskId)
	disk := iDisk.(*models.SDisk)
	providerDisk, err := disk.GetIDisk()
	if err != nil {
		return err
	}
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		cloudSnapshot, err := providerDisk.CreateISnapshot(snapshot.Name, "")
		if err != nil {
			return nil, err
		}
		res := jsonutils.NewDict()
		res.Set("snapshot_id", jsonutils.NewString(cloudSnapshot.GetId()))
		res.Set("manager_id", jsonutils.NewString(cloudSnapshot.GetManagerId()))
		cloudRegion, err := models.CloudregionManager.FetchByExternalId("Aliyun/" + cloudSnapshot.GetRegionId())
		if err != nil {
			return nil, fmt.Errorf("Cloud region not found? %s", err)
		}
		res.Set("cloudregion_id", jsonutils.NewString(cloudRegion.GetId()))
		return res, nil
	})
	return nil
}
