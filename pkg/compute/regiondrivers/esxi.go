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

package regiondrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SEsxiRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SEsxiRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SEsxiRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_VMWARE
}

func (self *SEsxiRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	return fmt.Errorf("%s does not support creating snapshot", self.GetProvider())
}

func (self *SEsxiRegionDriver) RequestCreateInstanceSnapshot(ctx context.Context, guest *models.SGuest, isp *models.SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ivm, err := guest.GetIVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "unable to GetIVM")
		}
		cloudSP, err := ivm.CreateInstanceSnapshot(ctx, isp.GetName(), isp.Description)
		if err != nil {
			return nil, errors.Wrap(err, "unable to CreateInstanceSnapshot")
		}
		_, err = db.Update(isp, func() error {
			isp.SetExternalId(cloudSP.GetGlobalId())
			return nil
		})
		return nil, err
	})
	return nil
}

func (self *SEsxiRegionDriver) RequestDeleteInstanceSnapshot(ctx context.Context, isp *models.SInstanceSnapshot, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		guest, err := isp.GetGuest()
		if err != nil {
			return nil, errors.Wrap(err, "GetGuest")
		}
		ivm, err := guest.GetIVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "unable to GetIVM")
		}
		id := isp.GetExternalId()
		if len(id) == 0 {
			return nil, nil
		}
		cloudSP, err := ivm.GetInstanceSnapshot(id)
		if err != nil {
			return nil, errors.Wrap(err, "unable to GetInstanceSnapshot")
		}
		err = cloudSP.Delete()
		if err != nil {
			return nil, errors.Wrap(err, "unable to delete cloud instance snapshot")
		}
		return nil, nil
	})
	return nil
}

func (self *SEsxiRegionDriver) RequestResetToInstanceSnapshot(ctx context.Context, guest *models.SGuest, isp *models.SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ivm, err := guest.GetIVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "unable to GetIVM")
		}
		err = ivm.ResetToInstanceSnapshot(ctx, isp.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "unable to ResetToInstanceSnapshot")
		}
		return nil, nil
	})
	return nil
}
