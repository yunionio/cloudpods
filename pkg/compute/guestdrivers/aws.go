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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SAwsGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SAwsGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func fetchAwsUserName(desc cloudprovider.SManagedVMCreateConfig) string {
	// 非公有云官方镜像
	if desc.ImageType != "system" {
		return "root"
	}

	// 公有云官方镜像
	dist := strings.ToLower(desc.OsDistribution)
	if strings.Contains(dist, "centos") {
		return "centos"
	} else if strings.Contains(dist, "ubuntu") {
		return "ubuntu"
	} else if strings.Contains(dist, "windows") {
		return "Administrator"
	} else if strings.Contains(dist, "debian") {
		return "admin"
	} else if strings.Contains(dist, "suse") {
		return "ec2-user"
	} else if strings.Contains(dist, "fedora") {
		return "ec2-user"
	} else if strings.Contains(dist, "rhel") || strings.Contains(dist, "redhat") {
		return "ec2-user"
	} else if strings.Contains(dist, "amazon linux") {
		return "ec2-user"
	} else {
		return "ec2-user"
	}
}

func (self *SAwsGuestDriver) IsNeedInjectPasswordByCloudInit(desc *cloudprovider.SManagedVMCreateConfig) bool {
	return true
}

func (self *SAwsGuestDriver) GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string {
	// return fetchAwsUserName(desc)
	if desc.OsType == "Windows" {
		return api.VM_AWS_DEFAULT_WINDOWS_LOGIN_USER
	}

	return api.VM_AWS_DEFAULT_LOGIN_USER
}

func (self *SAwsGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_AWS
}

func (self *SAwsGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_AWS
}

func (self *SAwsGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseQuotaKeys = quotas.OwnerIdQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_AWS
	keys.Brand = api.CLOUD_PROVIDER_AWS
	keys.Hypervisor = api.HYPERVISOR_AWS
	return keys
}

func (self *SAwsGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_GP2_SSD
}

func (self *SAwsGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SAwsGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_GP2_SSD,
		api.STORAGE_IO1_SSD,
		api.STORAGE_ST1_HDD,
		api.STORAGE_SC1_HDD,
		api.STORAGE_STANDARD_HDD,
	}
}

func (self *SAwsGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	return self.chooseHostStorage(self, host, backend, storageIds)
}

func (self *SAwsGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SAwsGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if len(input.Eip) > 0 || input.EipBw > 0 {
		// 未明确指定network时，由调度器进行调度，跳过support_eip检查
		if len(input.Networks) > 0 && len(input.Networks[0].Network) > 0 {
			inetwork, err := db.FetchByIdOrName(models.NetworkManager, userCred, input.Networks[0].Network)
			if err != nil {
				return nil, errors.Wrap(err, "SAwsGuestDriver.ValidateCreateData.Networks.FetchByIdOrName")
			}

			support_eip := inetwork.(*models.SNetwork).GetMetadataJson("ext:support_eip", nil)
			if support_eip != nil {
				if ok, _ := support_eip.Bool(); !ok {
					return nil, httperrors.NewInputParameterError("network %s associated route table has no internet gateway attached.", inetwork.GetName())
				}
			}
		}
	}

	return self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
}

func (self *SAwsGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	// https://docs.amazonaws.cn/AWSEC2/latest/UserGuide/stop-start.html
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == api.DISK_TYPE_SYS && !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_IO1_SSD, api.STORAGE_STANDARD_HDD, api.STORAGE_GP2_SSD}) {
		return fmt.Errorf("Cannot resize system disk with unsupported volumes type %s", storage.StorageType)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_GP2_SSD, api.STORAGE_IO1_SSD, api.STORAGE_ST1_HDD, api.STORAGE_SC1_HDD, api.STORAGE_STANDARD_HDD}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	return nil
}

func (self *SAwsGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SAwsGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SAwsGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SAwsGuestDriver) RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, server *models.SGuest, eip *models.SElasticip, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if server.Status != api.VM_ASSOCIATE_EIP {
			server.SetStatus(userCred, api.VM_ASSOCIATE_EIP, "associate eip")
		}

		extEip, err := eip.GetIEip()
		if err != nil {
			return nil, fmt.Errorf("SAwsGuestDriver.RequestAssociateEip fail to find iEIP for eip %s", err)
		}

		conf := &cloudprovider.AssociateConfig{
			InstanceId:    server.ExternalId,
			Bandwidth:     eip.Bandwidth,
			AssociateType: api.EIP_ASSOCIATE_TYPE_SERVER,
		}

		err = extEip.Associate(conf)
		if err != nil {
			return nil, fmt.Errorf("SAwsGuestDriver.RequestAssociateEip fail to remote associate EIP %s", err)
		}

		err = cloudprovider.WaitStatus(extEip, api.EIP_STATUS_READY, 3*time.Second, 60*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "SAwsGuestDriver.RequestAssociateEip.WaitStatus")
		}

		err = eip.AssociateVM(ctx, userCred, server)
		if err != nil {
			return nil, fmt.Errorf("SAwsGuestDriver.RequestAssociateEip fail to local associate EIP %s", err)
		}

		eip.SetStatus(userCred, api.EIP_STATUS_READY, "associate")

		// 如果aws已经绑定了EIP，则要把多余的公有IP删除
		if extEip.GetMode() == api.EIP_MODE_STANDALONE_EIP {
			publicIP, err := server.GetPublicIp()
			if err != nil {
				return nil, errors.Wrap(err, "AwsGuestDriver.GetPublicIp")
			}

			if publicIP != nil {
				err = db.DeleteModel(ctx, userCred, publicIP)
				if err != nil {
					return nil, errors.Wrap(err, "AwsGuestDriver.DeletePublicIp")
				}
			}
		}

		return nil, nil
	})

	return nil
}
