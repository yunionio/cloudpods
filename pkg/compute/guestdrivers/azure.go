package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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

func (self *SAzureGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

type SAzureVMCreateConfig struct {
	Name              string
	ExternalImageId   string
	OsDistribution    string
	OsVersion         string
	Cpu               int
	Memory            int
	ExternalNetworkId string
	IpAddr            string
	Description       string
	StorageType       string
	SysDiskSize       int
	DataDisks         []int
	PublicKey         string
}

func (self *SAzureGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())

	if action, err := config.GetString("action"); err != nil {
		return err
	} else if ihost, err := host.GetIHost(); err != nil {
		return err
	} else if action == "create" {
		desc := SAzureVMCreateConfig{}
		if err := config.Unmarshal(&desc, "desc"); err != nil {
			return err
		}

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			passwd := seclib2.RandomPassword2(12)

			if iVM, err := ihost.CreateVM(desc.Name, desc.ExternalImageId, desc.SysDiskSize, desc.Cpu, desc.Memory, desc.ExternalNetworkId,
				desc.IpAddr, desc.Description, passwd, desc.StorageType, desc.DataDisks, desc.PublicKey); err != nil {
				return nil, err
			} else {
				log.Debugf("VMcreated %s, wait status ready ...", iVM.GetGlobalId())

				if err := cloudprovider.WaitStatus(iVM, models.VM_READY, time.Second*5, time.Second*1800); err != nil {
					return nil, err
				}

				if iVM, err = ihost.GetIVMById(iVM.GetGlobalId()); err != nil {
					log.Errorf("cannot find vm %s", err)
					return nil, err
				}

				if len(guest.SecgrpId) > 0 {
					if err := iVM.SyncSecurityGroup(guest.SecgrpId, guest.GetSecgroupName(), guest.GetSecRules()); err != nil {
						log.Errorf("SyncSecurityGroup error: %v", err)
						return nil, err
					}
				}

				data := jsonutils.NewDict()

				if encpasswd, err := utils.EncryptAESBase64(guest.Id, passwd); err != nil {
					log.Errorf("encrypt password failed %s", err)
				} else {
					data.Add(jsonutils.NewString(iVM.GetOSType()), "os")
					data.Add(jsonutils.NewString("root"), "account")
					data.Add(jsonutils.NewString(encpasswd), "key")

					if len(desc.OsDistribution) > 0 {
						data.Add(jsonutils.NewString(desc.OsDistribution), "distro")
					}
					if len(desc.OsVersion) > 0 {
						data.Add(jsonutils.NewString(desc.OsVersion), "version")
					}
				}

				if idisks, err := iVM.GetIDisks(); err != nil {
					log.Errorf("GetiDisks error %s", err)
				} else {
					diskInfo := make([]SDiskInfo, len(idisks))
					for i := 0; i < len(idisks); i++ {
						dinfo := SDiskInfo{}
						dinfo.Uuid = idisks[i].GetGlobalId()
						dinfo.Size = idisks[i].GetDiskSizeMB()
						if metaData := idisks[i].GetMetadata(); metaData != nil {
							dinfo.Metadata = make(map[string]string, 0)
							if err := metaData.Unmarshal(dinfo.Metadata); err != nil {
								log.Errorf("Get disk %s metadata info error: %v", idisks[i].GetName(), err)
							}
						}
						diskInfo[i] = dinfo
					}
					data.Add(jsonutils.Marshal(&diskInfo), "disks")
				}

				data.Add(jsonutils.NewString(iVM.GetGlobalId()), "uuid")
				data.Add(iVM.GetMetadata(), "metadata")
				return data, nil
			}
		})
	} else if action == "deploy" {

	} else {
		return fmt.Errorf("Action %s not supported", action)
	}
	return nil
}
