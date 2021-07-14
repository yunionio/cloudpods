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

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SJDcloudHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SJDcloudHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SJDcloudHostDriver) GetHostType() string {
	return api.HOST_TYPE_JDCLOUD
}

func (self *SJDcloudHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_JDCLOUD
}

// ValidateResetDisk 仅可用状态的云硬盘支持恢复
// 卸载硬盘需要停止云主机
func (self *SJDcloudHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotSupportedError("not supported")
}

// ValidateDiskSize
// 云硬盘大小，单位为 GiB；ssd.io1 类型取值范围[20,16000]GB,步长为10G;
// ssd.gp1 类型取值范围[20,16000]GB,步长为10G;
// hdd.std1 类型取值范围[20,16000]GB,步长为10G
//
// 系统盘：
// local：不能指定大小，默认为40GB
// cloud：取值范围: [40,500]GB，并且不能小于镜像的最小系统盘大小，如果没有指定，默认以镜像中的系统盘大小为准
func (self *SJDcloudHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	switch storage.StorageType {
	case api.STORAGE_JDCLOUD_GP1, api.STORAGE_JDCLOUD_IO1, api.STORAGE_JDCLOUD_STD:
		if sizeGb < 20 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 16000GB", storage.StorageType)
		}
	default:
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}

	return nil
}
