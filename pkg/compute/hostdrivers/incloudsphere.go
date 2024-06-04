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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SInCloudSphereHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SInCloudSphereHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SInCloudSphereHostDriver) GetHostType() string {
	return api.HOST_TYPE_INCLOUD_SPHERE
}

func (self *SInCloudSphereHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_INCLOUD_SPHERE
}

func (self *SInCloudSphereHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_INCLOUD_SPHERE
}

func (self *SInCloudSphereHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return nil
}

func (self *SInCloudSphereHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return -1
}

func (self *SInCloudSphereHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, input *api.DiskResetInput) (*api.DiskResetInput, error) {
	return input, nil
}
