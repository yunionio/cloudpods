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

package hostdrivers

import (
	"context"
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHCSOPHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SHCSOPHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SHCSOPHostDriver) GetHostType() string {
	return api.HOST_TYPE_HCSOP
}

func (self *SHCSOPHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_HCSOP
}

func (self *SHCSOPHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_HCSOP
}

// 系统盘必须至少40G
func (self *SHCSOPHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if sizeGb < 1 || sizeGb > 65536 {
		return fmt.Errorf("The %s disk size must be in the range of 1G ~ 65536GB", storage.StorageType)
	}
	return nil
}

func (self *SHCSOPHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, input *api.DiskResetInput) (*api.DiskResetInput, error) {
	if len(guests) >= 1 {
		if disk.DiskType == api.DISK_TYPE_SYS {
			for _, g := range guests {
				if g.Status != api.VM_READY {
					return nil, httperrors.NewBadRequestError("Server %s must in status ready", g.GetName())
				}
			}
		} else {
			return nil, httperrors.NewBadRequestError("Disk must be detached")
		}
	}
	return input, nil
}
