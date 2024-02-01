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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func RunBatchCreateTask(
	ctx context.Context,
	items []db.IModel,
	userCred mcclient.TokenCredential,
	data []jsonutils.JSONObject,
	pendingUsage SQuota,
	pendingRegionUsage SRegionQuota,
	taskName string,
	parentTaskId string,
) error {
	taskItems := make([]db.IStandaloneModel, len(items))
	for i, t := range items {
		taskItems[i] = t.(db.IStandaloneModel)
	}
	params := jsonutils.NewDict()
	params.Set("data", jsonutils.NewArray(data...))

	task, err := taskman.TaskManager.NewParallelTask(ctx, taskName, taskItems, userCred, params, parentTaskId, "", &pendingUsage, &pendingRegionUsage)
	if err != nil {
		return errors.Wrapf(err, "NewParallelTask %s", taskName)
	}

	return task.ScheduleRun(nil)
}

func ValidateScheduleCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput, hypervisor string) (*api.ServerCreateInput, error) {
	var err error

	if input.Baremetal {
		hypervisor = api.HYPERVISOR_BAREMETAL
	}

	// base validate_create_data
	if (input.PreferHost != "") && hypervisor != api.HYPERVISOR_POD {

		bmName := input.PreferHost
		bmObj, err := HostManager.FetchByIdOrName(nil, bmName)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("Host %s not found", bmName)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		baremetal := bmObj.(*SHost)

		err = baremetal.IsAssignable(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "IsAssignable")
		}

		if !baremetal.GetEnabled() {
			return nil, httperrors.NewInvalidStatusError("Baremetal %s not enabled", bmName)
		}

		if len(hypervisor) > 0 && hypervisor != api.HOSTTYPE_HYPERVISOR[baremetal.HostType] {
			return nil, httperrors.NewInputParameterError("cannot run hypervisor %s on specified host with type %s", hypervisor, baremetal.HostType)
		}

		if len(hypervisor) == 0 {
			hypervisor = api.HOSTTYPE_HYPERVISOR[baremetal.HostType]
		}

		if len(hypervisor) == 0 {
			hypervisor = api.HYPERVISOR_DEFAULT
		}

		_, err = GetDriver(hypervisor).ValidateCreateDataOnHost(ctx, userCred, bmName, baremetal, input)
		if err != nil {
			return nil, err
		}

		defaultStorage, err := GetDriver(hypervisor).ChooseHostStorage(baremetal, nil, &api.DiskConfig{}, nil)
		if err != nil {
			return nil, errors.Wrap(err, "ChooseHostStorage")
		}
		if defaultStorage == nil {
			return nil, httperrors.NewInsufficientResourceError("no valid storage on host")
		}
		input.PreferHost = baremetal.Id
		input.DefaultStorageType = defaultStorage.StorageType

		zone, _ := baremetal.GetZone()
		input.PreferZone = zone.Id
		region, _ := zone.GetRegion()
		input.PreferRegion = region.Id
	} else {
		if len(input.Schedtags) > 0 {
			input.Schedtags, err = SchedtagManager.ValidateSchedtags(userCred, input.Schedtags)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid aggregate_strategy: %s", err)
			}
		}

		if input.PreferWire != "" {
			wireStr := input.PreferWire
			wireObj, err := WireManager.FetchByIdOrName(userCred, wireStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Wire %s not found", wireStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			wire := wireObj.(*SWire)
			input.PreferWire = wire.Id
			zone, _ := wire.GetZone()
			input.PreferZone = zone.Id
			region, _ := zone.GetRegion()
			input.PreferRegion = region.Id
		} else if input.PreferZone != "" {
			zoneStr := input.PreferZone
			zoneObj, err := ZoneManager.FetchByIdOrName(userCred, zoneStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Zone %s not found", zoneStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			zone := zoneObj.(*SZone)
			input.PreferZone = zone.Id
			region, _ := zone.GetRegion()
			input.PreferRegion = region.Id
		} else if input.PreferRegion != "" {
			regionStr := input.PreferRegion
			regionObj, err := CloudregionManager.FetchByIdOrName(userCred, regionStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Region %s not found", regionStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			region := regionObj.(*SCloudregion)
			input.PreferRegion = region.Id
		}
	}

	// default hypervisor
	if len(hypervisor) == 0 {
		hypervisor = api.HYPERVISOR_KVM
	}

	if !utils.IsInStringArray(hypervisor, api.HYPERVISORS) {
		return nil, httperrors.NewInputParameterError("Hypervisor %s not supported", hypervisor)
	}

	input.Hypervisor = hypervisor
	return input, nil
}
