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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
)

type SAzureGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SAzureGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SAzureGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_AZURE
}

func (self *SAzureGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_AZURE
}

func (self *SAzureGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_AZURE
	keys.Brand = api.CLOUD_PROVIDER_AZURE
	keys.Hypervisor = api.HYPERVISOR_AZURE
	return keys
}

func (self *SAzureGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_STANDARD_LRS
}

func (self *SAzureGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 30
}

func (self *SAzureGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_STANDARD_LRS,
		api.STORAGE_STANDARDSSD_LRS,
		api.STORAGE_PREMIUM_LRS,
	}
}

func (self *SAzureGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SAzureGuestDriver) GetMaxSecurityGroupCount() int {
	return 1
}

func (self *SAzureGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) IsRebuildRootSupportChangeUEFI() bool {
	return false
}

func (self *SAzureGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SAzureGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	//https://docs.microsoft.com/en-us/rest/api/compute/disks/update
	//Resizes are only allowed if the disk is not attached to a running VM, and can only increase the disk's size
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SAzureGuestDriver) ValidateImage(ctx context.Context, image *cloudprovider.SImage) error {
	if len(image.ExternalId) == 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		result, err := modules.Images.GetSpecific(s, image.Id, "subformats", nil)
		if err != nil {
			return errors.Wrap(err, "get subformats")
		}
		subFormats := []struct {
			Format string
		}{}
		err = result.Unmarshal(&subFormats)
		if err != nil {
			return errors.Wrap(err, "Unmarshal subformats")
		}
		for i := range subFormats {
			if subFormats[i].Format == "vhd" {
				return nil
			}
		}
		return httperrors.NewResourceNotFoundError("failed to find subformat vhd for image %s, please append 'vhd' for glance options(target_image_formats)", image.Name)
	}
	return nil
}

func (self *SAzureGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	input, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) > 2 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	if len(input.Disks) > 0 && len(input.Disks[0].ImageId) > 0 {
		_image, err := models.CachedimageManager.FetchById(input.Disks[0].ImageId)
		if err != nil {
			return nil, errors.Wrap(err, "FetchById")
		}

		cachedimage := _image.(*models.SCachedimage)
		image, err := cachedimage.GetImage()
		if err != nil {
			return nil, errors.Wrap(err, "GetImage")
		}

		err = self.ValidateImage(ctx, image)
		if err != nil {
			return nil, err
		}

		if len(image.ExternalId) > 0 && len(input.InstanceType) > 0 {
			if cachedimage.UEFI.IsFalse() {
				if strings.HasPrefix(input.InstanceType, "Standard_M") && strings.HasSuffix(input.InstanceType, "v2") {
					return nil, httperrors.NewNotSupportedError("Azure Mv2-series instance sku only support UEFI image")
				}
			} else {
				// https://docs.microsoft.com/en-us/azure/virtual-machines/windows/generation-2
				if !(strings.HasPrefix(input.InstanceType, "Standard_B") || // B-series
					(strings.HasPrefix(input.InstanceType, "Standard_DC") && strings.HasSuffix(input.InstanceType, "s_v2") || input.InstanceType == "Standard_DC8_v2") || // DCsv2-series
					(strings.HasPrefix(input.InstanceType, "Standard_DS") && strings.HasSuffix(input.InstanceType, "v2")) || // DSv2-series
					(strings.HasPrefix(input.InstanceType, "Standard_DS") && strings.HasSuffix(input.InstanceType, "s_v3")) || // Dsv3-series
					(strings.HasPrefix(input.InstanceType, "Standard_D") && strings.HasSuffix(input.InstanceType, "as_v4")) || // Dasv4-series
					(strings.HasPrefix(input.InstanceType, "Standard_E") && strings.HasSuffix(input.InstanceType, "s_v3")) || // Esv3-series
					(strings.HasPrefix(input.InstanceType, "Standard_E") && strings.HasSuffix(input.InstanceType, "as_v4")) || // Easv4-series
					(strings.HasPrefix(input.InstanceType, "Standard_F") && strings.HasSuffix(input.InstanceType, "s_v2")) || // Fsv2-series
					(strings.HasPrefix(input.InstanceType, "Standard_GS")) || // GS-series
					(strings.HasPrefix(input.InstanceType, "Standard_HB")) || // HB-series
					(strings.HasPrefix(input.InstanceType, "Standard_HC")) || // HC-series
					(strings.HasPrefix(input.InstanceType, "Standard_L") && strings.HasSuffix(input.InstanceType, "s")) || // Ls-series
					(strings.HasPrefix(input.InstanceType, "Standard_L") && strings.HasSuffix(input.InstanceType, "s_v2")) || // Ls-series
					(strings.HasPrefix(input.InstanceType, "Standard_M")) || // M-series
					(strings.HasPrefix(input.InstanceType, "Standard_M") && strings.HasSuffix(input.InstanceType, "s_v2")) || // Mv2-series
					(strings.HasPrefix(input.InstanceType, "Standard_NC") && strings.HasSuffix(input.InstanceType, "s_v2")) || // NCv2-series
					(strings.HasPrefix(input.InstanceType, "Standard_NC") && strings.HasSuffix(input.InstanceType, "s_v3")) || // NCv3-series
					(strings.HasPrefix(input.InstanceType, "Standard_ND")) || // ND-series
					(strings.HasPrefix(input.InstanceType, "Standard_NV") && strings.HasSuffix(input.InstanceType, "s_v3"))) { // NVv3-series
					return nil, httperrors.NewUnsupportOperationError("Azure UEFI image %s not support this instance sku", image.Name)
				}
			}
		}
	}
	return input, nil
}

func (self *SAzureGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SAzureGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SAzureGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_AZURE_DEFAULT_LOGIN_USER,
				Changeable:     false,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				// Administrator 为Azure保留用户，不能使用
				DefaultAccount: api.VM_AZURE_DEFAULT_LOGIN_USER,
				Changeable:     false,
			},
		},
		Storages: cloudprovider.Storage{
			DataDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_STANDARDSSD_LRS, MaxSizeGb: 32767, MinSizeGb: 1, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_STANDARD_LRS, MaxSizeGb: 32767, MinSizeGb: 1, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_PREMIUM_LRS, MaxSizeGb: 32767, MinSizeGb: 1, StepSizeGb: 1, Resizable: false},
			},
			SysDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_STANDARDSSD_LRS, MaxSizeGb: 32767, MinSizeGb: 1, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_STANDARD_LRS, MaxSizeGb: 32767, MinSizeGb: 1, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_PREMIUM_LRS, MaxSizeGb: 32767, MinSizeGb: 1, StepSizeGb: 1, Resizable: false},
			},
		},
	}
}

func (self *SAzureGuestDriver) GetDefaultAccount(osType, osDist, imageType string) string {
	return api.VM_AZURE_DEFAULT_LOGIN_USER
}

func (self *SAzureGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}
