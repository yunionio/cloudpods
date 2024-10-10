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

package volume_mount

import (
	"context"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterContainerVolumeMountDriver(newCephFS())
}

type cephFS struct{}

func newCephFS() models.IContainerVolumeMountDriver {
	return &cephFS{}
}

func (h cephFS) GetType() apis.ContainerVolumeMountType {
	return apis.CONTAINER_VOLUME_MOUNT_TYPE_CEPHF_FS
}

func (h cephFS) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *models.SGuest, vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error) {
	return vm, h.ValidatePodCreateData(ctx, userCred, vm, nil)
}

func (h cephFS) ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, vm *apis.ContainerVolumeMount, input *api.ServerCreateInput) error {
	fs := vm.CephFS
	if fs == nil {
		return httperrors.NewNotEmptyError("ceph_fs is nil")
	}
	if len(fs.Id) == 0 {
		return httperrors.NewNotEmptyError("cep_fs.id is empty")
	}
	obj, err := validators.ValidateModel(ctx, userCred, models.FileSystemManager, &fs.Id)
	if err != nil {
		return err
	}
	filesystem := obj.(*models.SFileSystem)
	fs.Id = filesystem.Id
	if filesystem.Status != apis.STATUS_AVAILABLE {
		return httperrors.NewInvalidStatusError("invalid cephfs status %s", filesystem.Status)
	}
	account := filesystem.GetCloudaccount()
	if gotypes.IsNil(account) {
		return httperrors.NewInputParameterError("invalid cephfs %s", filesystem.Name)
	}
	if !db.IsAllowUpdate(ctx, rbacscope.ScopeProject, userCred, filesystem) {
		vm.ReadOnly = true
	}
	if account.Provider != api.CLOUD_PROVIDER_CEPHFS {
		return httperrors.NewInputParameterError("invalid cephfs type %s", account.Provider)
	}
	if gotypes.IsNil(account.Options) {
		return httperrors.NewInputParameterError("missing mon_host")
	}
	monHost, _ := account.Options.GetString("mon_host")
	if len(monHost) == 0 {
		return httperrors.NewInputParameterError("missing mon_host")
	}
	_, err = account.GetOptionPassword()
	return err
}
