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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCNwareRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SCNwareRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SCNwareRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CNWARE
}

func (self *SCNwareRegionDriver) RequestCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, net *models.SNetwork, task taskman.ITask) error {
	return net.SetStatus(ctx, userCred, api.NETWORK_STATUS_AVAILABLE, "")
}
func (self *SCNwareRegionDriver) RequestCreateInstanceSnapshot(ctx context.Context, guest *models.SGuest, isp *models.SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ivm, err := guest.GetIVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "unable to GetIVM")
		}
		_, err = ivm.CreateInstanceSnapshot(ctx, isp.GetName(), isp.Description)
		if err != nil {
			return nil, errors.Wrap(err, "unable to CreateInstanceSnapshot")
		}
		_, err = db.Update(isp, func() error {
			isp.SetExternalId(isp.Name)
			return nil
		})
		return nil, err
	})
	return nil
}
