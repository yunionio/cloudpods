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

package models

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var IsolatedDeviceModelManager *SIsolatedDeviceModelManager

func init() {
	IsolatedDeviceModelManager = &SIsolatedDeviceModelManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			SIsolatedDeviceModel{},
			"isolated_device_models_tbl",
			"isolated_device_model",
			"isolated_device_models",
		),
	}
	IsolatedDeviceModelManager.SetVirtualObject(IsolatedDeviceModelManager)
}

type SIsolatedDeviceModelManager struct {
	db.SStandaloneAnonResourceBaseManager
}

type SIsolatedDeviceModel struct {
	db.SStandaloneAnonResourceBase

	Model string `width:"512" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" update:"domain"`

	VendorId string `width:"16" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" update:"domain"`
	DeviceId string `width:"16" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" update:"domain"`

	DevType string `width:"16" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
}

func (manager *SIsolatedDeviceModelManager) ValidateCreateData(ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.IsolatedDeviceModelCreateInput,
) (api.IsolatedDeviceModelCreateInput, error) {
	if utils.IsInStringArray(input.DevType, api.VALID_PASSTHROUGH_TYPES) {
		return input, httperrors.NewInputParameterError("device type %q unsupported", input.DevType)
	}

	input.VendorId = strings.ToLower(input.VendorId)
	input.DeviceId = strings.ToLower(input.DeviceId)
	deviceVendorReg := regexp.MustCompile(`[a-f0-9]{4}`)

	if !deviceVendorReg.MatchString(input.VendorId) {
		return input, httperrors.NewInputParameterError("bad vendor id %s", input.VendorId)
	}
	if !deviceVendorReg.MatchString(input.DeviceId) {
		return input, httperrors.NewInputParameterError("bad vendor id %s", input.DeviceId)
	}

	if cnt := manager.Query().Equals("vendor_id", input.VendorId).Equals("device_id", input.DeviceId).Count(); cnt > 0 {
		return input, httperrors.NewDuplicateResourceError("vendor %s device %s has been registered", input.VendorId, input.DeviceId)
	}

	if cnt := manager.Query().Equals("model", input.Model).Count(); cnt > 0 {
		return input, httperrors.NewDuplicateResourceError("model %s has been registered", input.Model)
	}

	return input, nil
}

func (self *SIsolatedDeviceModel) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := api.IsolatedDeviceModelCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		log.Errorf("!!!data.Unmarshal api.IsolatedDeviceModelCreateInput fail %s", err)
	}
	go func() {
		defer self.RemoveMetadata(ctx, api.MEAT_PROBED_HOST_COUNT, userCred)

		for i := range input.Hosts {
			iHost, err := HostManager.FetchByIdOrName(userCred, input.Hosts[i])
			if err != nil {
				log.Errorf("failed fetch host %s: %s", input.Hosts[i], err)
				continue
			}
			host := iHost.(*SHost)
			if err := host.RequestScanIsolatedDevices(ctx, userCred); err != nil {
				log.Errorf("failed scan isolated device %s", err)
			}
			self.SetMetadata(ctx, api.MEAT_PROBED_HOST_COUNT, strconv.Itoa(i+1), userCred)
		}
	}()
}

func (self *SIsolatedDeviceModel) ValidateUpdateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input api.IsolatedDeviceModelUpdateInput,
) (api.IsolatedDeviceModelUpdateInput, error) {
	input.VendorId = strings.ToLower(input.VendorId)
	input.DeviceId = strings.ToLower(input.DeviceId)
	deviceVendorReg := regexp.MustCompile(`[a-f0-9]{4}`)

	if !deviceVendorReg.MatchString(input.VendorId) {
		return input, httperrors.NewInputParameterError("bad vendor id %s", input.VendorId)
	}
	if !deviceVendorReg.MatchString(input.DeviceId) {
		return input, httperrors.NewInputParameterError("bad vendor id %s", input.DeviceId)
	}

	if self.VendorId != input.VendorId || self.DeviceId != input.DeviceId {
		if cnt := IsolatedDeviceModelManager.Query().Equals("vendor_id", input.VendorId).Equals("device_id", input.DeviceId).Count(); cnt > 0 {
			return input, httperrors.NewDuplicateResourceError("vendor %s device %s has been registered", input.VendorId, input.DeviceId)
		}
	}

	if self.Model != input.Model {
		if cnt := IsolatedDeviceModelManager.Query().Equals("model", input.Model).Count(); cnt > 0 {
			return input, httperrors.NewDuplicateResourceError("model %s has been registered", input.Model)
		}
	}

	return input, nil
}

func (manager *SIsolatedDeviceModelManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IsolatedDeviceModelListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneAnonResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}

	if len(query.DevType) > 0 {
		q = q.In("dev_type", query.DevType)
	}
	if len(query.Model) > 0 {
		q = q.In("model", query.Model)
	}

	if len(query.VendorId) > 0 {
		q = q.Equals("vendor_id", query.VendorId)
	}
	if len(query.DeviceId) > 0 {
		q = q.Equals("device_id", query.DeviceId)
	}
	return q, nil
}

func (manager *SIsolatedDeviceModelManager) GetByVendorDevice(vendorId, deviceId string) (*SIsolatedDeviceModel, error) {
	devModel := new(SIsolatedDeviceModel)
	err := manager.Query().Equals("vendor_id", vendorId).Equals("device_id", deviceId).First(devModel)
	if err != nil {
		return nil, err
	}
	devModel.SetModelManager(manager, devModel)
	return devModel, nil
}

func (manager *SIsolatedDeviceModelManager) GetByDevModel(model string) (*SIsolatedDeviceModel, error) {
	devModel := new(SIsolatedDeviceModel)
	err := manager.Query().Equals("model", model).First(devModel)
	if err != nil {
		return nil, err
	}
	devModel.SetModelManager(manager, devModel)
	return devModel, nil
}

func (manager *SIsolatedDeviceModelManager) GetByDevType(devType string) (*SIsolatedDeviceModel, error) {
	devModel := new(SIsolatedDeviceModel)
	err := manager.Query().Equals("dev_type", devType).First(devModel)
	if err != nil {
		return nil, err
	}
	devModel.SetModelManager(manager, devModel)
	return devModel, nil
}
