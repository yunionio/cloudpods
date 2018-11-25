package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAzureGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SAzureGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SAzureGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_AZURE
}

func (self *SAzureGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	storages := host.GetAttachedStorages("")
	for i := 0; i < len(storages); i += 1 {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}
	for _, stype := range []string{"standard_lrs", "premium_lrs"} {
		for i := 0; i < len(storages); i += 1 {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SAzureGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SAzureGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	//https://docs.microsoft.com/en-us/rest/api/compute/disks/update
	//Resizes are only allowed if the disk is not attached to a running VM, and can only increase the disk's size
	if !utils.IsInStringArray(guest.Status, []string{models.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SAzureGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SAzureGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("net.0") && data.Contains("net.1") {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	return data, nil
}

func (self *SAzureGuestDriver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("name") {
		return nil, httperrors.NewInputParameterError("cannot support change azure instance name")
	}
	return data, nil
}

func (self *SAzureGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())
	publicKey, _ := config.GetString("public_key")
	resetPassword := jsonutils.QueryBoolean(config, "reset_password", false)
	passwd, _ := config.GetString("password")
	if resetPassword && len(passwd) == 0 {
		passwd = seclib2.RandomPassword2(12)
	}

	adminPublicKey, _ := config.GetString("admin_public_key")
	projectPublicKey, _ := config.GetString("project_public_key")
	oUserData, _ := config.GetString("user_data")

	userData := generateUserData(adminPublicKey, projectPublicKey, oUserData)

	desc := SManagedVMCreateConfig{}
	if err := config.Unmarshal(&desc, "desc"); err != nil {
		return err
	}
	if action, err := config.GetString("action"); err != nil {
		return err
	} else if ihost, err := host.GetIHost(); err != nil {
		return err
	} else if action == "create" {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			nets := guest.GetNetworks()
			net := nets[0].GetNetwork()
			vpc := net.GetVpc()

			ivpc, err := vpc.GetIVpc()
			if err != nil {
				log.Errorf("getIVPC fail %s", err)
				return nil, err
			}

			if len(passwd) == 0 {
				passwd = seclib2.RandomPassword2(12)
			}

			secgrpId, err := ivpc.SyncSecurityGroup(desc.SecGroupId, desc.SecGroupName, desc.SecRules)
			if err != nil {
				log.Errorf("SyncSecurityGroup fail %s", err)
				return nil, err
			}

			if iVM, err := ihost.CreateVM(desc.Name, desc.ExternalImageId, desc.SysDiskSize, desc.Cpu, desc.Memory, desc.ExternalNetworkId,
				desc.IpAddr, desc.Description, passwd, desc.StorageType, desc.DataDisks, publicKey, secgrpId, userData); err != nil {
				return nil, err
			} else {
				log.Debugf("VMcreated %s, wait status running ...", iVM.GetGlobalId())
				if err = cloudprovider.WaitStatus(iVM, models.VM_RUNNING, time.Second*5, time.Second*1800); err != nil {
					return nil, err
				}
				if iVM, err = ihost.GetIVMById(iVM.GetGlobalId()); err != nil {
					log.Errorf("cannot find vm %s", err)
					return nil, err
				}

				data := fetchIVMinfo(desc, iVM, guest.Id, ansible.PUBLIC_CLOUD_ANSIBLE_USER, passwd, action)
				return data, nil
			}
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
			err := iVM.DeployVM(ctx, name, passwd, publicKey, deleteKeypair, description)
			if err != nil {
				return nil, err
			}
			data := fetchIVMinfo(desc, iVM, guest.Id, ansible.PUBLIC_CLOUD_ANSIBLE_USER, passwd, action)
			return data, nil
		})
	} else if action == "rebuild" {
		iVM, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil || iVM == nil {
			log.Errorf("cannot find vm %s", err)
			return fmt.Errorf("cannot find vm")
		}

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			_, err := iVM.RebuildRoot(ctx, desc.ExternalImageId, passwd, publicKey, desc.SysDiskSize)
			if err != nil {
				return nil, err
			}

			log.Debugf("VMrebuildRoot %s, and status is ready", iVM.GetGlobalId())
			data := fetchIVMinfo(desc, iVM, guest.Id, ansible.PUBLIC_CLOUD_ANSIBLE_USER, passwd, action)

			return data, nil
		})

	} else {
		return fmt.Errorf("Action %s not supported", action)
	}
	return nil
}
