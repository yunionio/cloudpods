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
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/google"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGoogleGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SGoogleGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SGoogleGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_GOOGLE
	keys.Brand = api.CLOUD_PROVIDER_GOOGLE
	keys.Hypervisor = api.HYPERVISOR_GOOGLE
	return keys
}

func (self *SGoogleGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_GOOGLE
}

func (self *SGoogleGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_GOOGLE
}

func (self *SGoogleGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
			},
		},
		Storages: cloudprovider.Storage{
			DataDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GOOGLE_PD_SSD, MaxSizeGb: 65536, MinSizeGb: 10, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GOOGLE_PD_STANDARD, MaxSizeGb: 65536, MinSizeGb: 10, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GOOGLE_PD_BALANCED, MaxSizeGb: 65536, MinSizeGb: 10, StepSizeGb: 1, Resizable: false},
			},
			SysDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GOOGLE_PD_SSD, MaxSizeGb: 65536, MinSizeGb: 10, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GOOGLE_PD_STANDARD, MaxSizeGb: 65536, MinSizeGb: 10, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GOOGLE_PD_BALANCED, MaxSizeGb: 65536, MinSizeGb: 10, StepSizeGb: 1, Resizable: false},
			},
		},
	}
}

func (self *SGoogleGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_GOOGLE_PD_STANDARD
}

func (self *SGoogleGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SGoogleGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_GOOGLE_PD_SSD,
		api.STORAGE_GOOGLE_PD_STANDARD,
		api.STORAGE_GOOGLE_PD_BALANCED,
		api.STORAGE_GOOGLE_LOCAL_SSD,
	}
}

func (self *SGoogleGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SGoogleGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SGoogleGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SGoogleGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SGoogleGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SGoogleGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SGoogleGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SGoogleGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_GOOGLE_PD_SSD, api.STORAGE_GOOGLE_PD_STANDARD, api.STORAGE_GOOGLE_PD_BALANCED}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	return nil
}

func (self *SGoogleGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	input, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) > 2 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	localDisk := 0
	for i, disk := range input.Disks {
		minGB := -1
		maxGB := -1
		switch disk.Backend {
		case api.STORAGE_GOOGLE_PD_SSD, api.STORAGE_GOOGLE_PD_STANDARD, api.STORAGE_GOOGLE_PD_BALANCED:
			minGB = 10
			maxGB = 65536
		case api.STORAGE_GOOGLE_LOCAL_SSD:
			minGB = 375
			maxGB = 375
			localDisk++
		default:
			return nil, httperrors.NewInputParameterError("Unknown google storage type %s", disk.Backend)
		}
		if i == 0 && disk.Backend == api.STORAGE_GOOGLE_LOCAL_SSD {
			return nil, httperrors.NewInputParameterError("System disk does not support %s disk", disk.Backend)
		}
		if disk.SizeMb < minGB*1024 || disk.SizeMb > maxGB*1024 {
			return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of %dGB ~ %dGB", disk.Backend, minGB, maxGB)
		}
	}
	if localDisk > 8 {
		return nil, httperrors.NewInputParameterError("%s disk cannot exceed 8", api.STORAGE_GOOGLE_LOCAL_SSD)
	}
	if localDisk > 0 && strings.HasPrefix(input.InstanceType, "e2") {
		return nil, httperrors.NewNotSupportedError("%s for %s features are not compatible for creating instance", input.InstanceType, api.STORAGE_GOOGLE_LOCAL_SSD)
	}
	return input, nil
}

func (self *SGoogleGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SGoogleGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

// 谷歌云的用户自定义脚本不支持base64加密
func (self *SGoogleGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL_WITHOUT_ENCRYPT
}

func (self *SGoogleGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	ivm, err := guest.GetIVM(ctx)
	if err != nil {
		return errors.Wrap(err, "GetIVM")
	}

	result := jsonutils.NewDict()
	if ivm.GetStatus() != api.VM_RUNNING {
		err := ivm.StartVM(ctx)
		if err != nil {
			return errors.Wrap(err, "ivm.StartVM")
		}
		vm := ivm.(*google.SInstance)
		updateUserdata := false
		for _, item := range vm.Metadata.Items {
			if item.Key == google.METADATA_STARTUP_SCRIPT || item.Key == google.METADATA_STARTUP_SCRIPT_POWER_SHELL {
				updateUserdata = true
				break
			}
		}
		if updateUserdata {
			keyword := "Finished running startup scripts"
			err = cloudprovider.Wait(time.Second*5, time.Minute*6, func() (bool, error) {
				output, err := ivm.GetSerialOutput(1)
				if err != nil {
					return false, errors.Wrap(err, "iVM.GetSerialOutput")
				}
				log.Debugf("wait for google startup scripts finish")
				if strings.Contains(output, keyword) {
					log.Debugf(keyword)
					return true, nil
				}
				return false, nil
			})
			if err != nil {
				log.Errorf("failed wait google cloud startup scripts finish err: %v", err)
			}
			log.Debugf("clean google instance %s(%s) startup-script", guest.Name, guest.Id)
			err := ivm.UpdateUserData("")
			if err != nil {
				log.Errorf("failed to update google userdata")
			}
		}
		guest.SetStatus(ctx, userCred, api.VM_RUNNING, "StartOnHost")
		return task.ScheduleRun(result)
	}
	return guest.SetStatus(ctx, userCred, api.VM_RUNNING, "StartOnHost")
}

func (self *SGoogleGuestDriver) RemoteActionAfterGuestCreated(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, iVM cloudprovider.ICloudVM, desc *cloudprovider.SManagedVMCreateConfig) {
	keywords := map[string]string{
		strings.ToLower(osprofile.OS_TYPE_WINDOWS): "Finished with sysprep specialize phase",
		strings.ToLower(osprofile.OS_TYPE_LINUX):   "Finished running startup scripts",
	}
	if keyword, ok := keywords[strings.ToLower(desc.OsType)]; ok {
		err := cloudprovider.Wait(time.Second*5, time.Minute*6, func() (bool, error) {
			output, err := iVM.GetSerialOutput(1)
			if err != nil {
				return false, errors.Wrap(err, "iVM.GetSerialOutput")
			}
			log.Debugf("wait for google sysprep finish")
			if strings.Contains(output, keyword) {
				log.Debugf(keyword)
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			log.Errorf("failed wait google %s error: %v", keyword, err)
		}
	}
	log.Debugf("clean google instance %s(%s) startup-script", guest.Name, guest.Id)
	err := iVM.UpdateUserData("")
	if err != nil {
		log.Errorf("failed to update google userdata")
	}
}

func (self *SGoogleGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SGoogleGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}
