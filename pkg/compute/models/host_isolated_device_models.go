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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SHostIsolatedDeviceModelManager struct {
	SHostJointsManager
}

var HostIsolatedDeviceModelManager *SHostIsolatedDeviceModelManager

func init() {
	db.InitManager(func() {
		HostIsolatedDeviceModelManager = &SHostIsolatedDeviceModelManager{
			SHostJointsManager: NewHostJointsManager(
				"host_id",
				SHostIsolatedDeviceModel{},
				"host_isolated_device_models_tbl",
				"host_isolated_device_model",
				"host_isolated_device_models",
				IsolatedDeviceModelManager,
			),
		}
		HostIsolatedDeviceModelManager.SetVirtualObject(HostIsolatedDeviceModelManager)
		HostIsolatedDeviceModelManager.TableSpec().AddIndex(false, "host_id", "isolated_device_model_id")
	})
}

// +onecloud:swagger-gen-ignore
type SHostIsolatedDeviceModel struct {
	SHostJointsBase

	// 宿主机Id
	HostId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"required" json:"host_id"`
	// 设备类型Id
	IsolatedDeviceModelId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"required" json:"isolated_device_model_id" index:"true"`
}

func (manager *SHostIsolatedDeviceModelManager) GetMasterFieldName() string {
	return "host_id"
}

func (manager *SHostIsolatedDeviceModelManager) GetSlaveFieldName() string {
	return "isolated_device_model_id"
}

func (self *SHostIsolatedDeviceModel) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SHostIsolatedDeviceModel) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SHostIsolatedDeviceModelManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostIsolatedDeviceModelDetails {
	rows := make([]api.HostIsolatedDeviceModelDetails, len(objs))
	hostRows := manager.SHostJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	devModelIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.HostIsolatedDeviceModelDetails{
			HostJointResourceDetails: hostRows[i],
		}
		devModelIds[i] = objs[i].(*SHostIsolatedDeviceModel).IsolatedDeviceModelId
	}

	devModels := make(map[string]SIsolatedDeviceModel)
	err := db.FetchStandaloneObjectsByIds(IsolatedDeviceModelManager, devModelIds, &devModels)
	if err != nil {
		log.Errorf("db.FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if devModel, ok := devModels[devModelIds[i]]; ok {
			rows[i] = objs[i].(*SHostIsolatedDeviceModel).getExtraDetails(devModel, rows[i])
		}
	}

	return rows
}

func (self *SHostIsolatedDeviceModel) getExtraDetails(devModel SIsolatedDeviceModel, out api.HostIsolatedDeviceModelDetails) api.HostIsolatedDeviceModelDetails {
	out.Model = devModel.Model
	out.VendorId = devModel.VendorId
	out.DeviceId = devModel.DeviceId
	out.DevType = devModel.DevType
	out.HotPluggable = devModel.HotPluggable.Bool()
	out.DisableAutoDetect = devModel.DisableAutoDetect.Bool()
	return out
}

func (self *SHostIsolatedDeviceModel) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SHostJointsBase.PostCreate(ctx, userCred, ownerId, query, data)

	iHost, err := HostManager.FetchByIdOrName(ctx, userCred, self.HostId)
	if err != nil {
		log.Errorf("failed fetch host %s: %s", self.HostId, err)
		return
	}
	host := iHost.(*SHost)
	log.Infof("start request host %s scan isolated devices", host.GetName())
	if err := host.RequestScanIsolatedDevices(ctx, userCred); err != nil {
		log.Errorf("failed scan isolated device %s", err)
	}
}
