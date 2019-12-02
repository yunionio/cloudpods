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
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func RunBatchCreateTask(
	ctx context.Context,
	items []db.IModel,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
	pendingUsage SQuota,
	pendingRegionUsage SRegionQuota,
	taskName string,
	parentTaskId string,
) {
	taskItems := make([]db.IStandaloneModel, len(items))
	for i, t := range items {
		taskItems[i] = t.(db.IStandaloneModel)
	}
	params := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewParallelTask(ctx, taskName, taskItems, userCred, params, parentTaskId, "", &pendingUsage, &pendingRegionUsage)
	if err != nil {
		log.Errorf("%s newTask error %s", taskName, err)
	} else {
		task.ScheduleRun(nil)
	}
}

func allowAssignHost(userCred mcclient.TokenCredential) bool {
	for _, scope := range []rbacutils.TRbacScope{
		rbacutils.ScopeSystem,
		rbacutils.ScopeDomain,
	} {
		if userCred.IsAllow(scope, consts.GetServiceType(), GuestManager.KeywordPlural(), policy.PolicyActionPerform, "assign-host") {
			return true
		}
	}
	return false
}

func ValidateScheduleCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput, hypervisor string) (*api.ServerCreateInput, error) {
	var err error

	if input.Baremetal {
		hypervisor = api.HYPERVISOR_BAREMETAL
	}

	// base validate_create_data
	if (input.PreferHost != "") && hypervisor != api.HYPERVISOR_CONTAINER {

		if !allowAssignHost(userCred) {
			return nil, httperrors.NewNotSufficientPrivilegeError("Only system admin can specify preferred host")
		}
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
		if !baremetal.Enabled {
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

		defaultStorage := GetDriver(hypervisor).ChooseHostStorage(baremetal, "", nil)
		if defaultStorage == nil {
			return nil, httperrors.NewInsufficientResourceError("no valid storage on host")
		}
		input.PreferHost = baremetal.Id
		input.DefaultStorageType = defaultStorage.StorageType

		zone := baremetal.GetZone()
		input.PreferZone = zone.Id
		region := zone.GetRegion()
		input.PreferRegion = region.Id
	} else {
		schedtags := make(map[string]string)
		for _, tag := range input.Schedtags {
			schedtags[tag.Id] = tag.Strategy
		}
		if len(schedtags) > 0 {
			schedtags, err = SchedtagManager.ValidateSchedtags(userCred, schedtags)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid aggregate_strategy: %s", err)
			}
			tags := make([]*api.SchedtagConfig, 0)
			for name, strategy := range schedtags {
				tags = append(tags, &api.SchedtagConfig{Id: name, Strategy: strategy})
			}
			input.Schedtags = tags
		}

		if input.PreferWire != "" {
			wireStr := input.PreferWire
			wireObj, err := WireManager.FetchById(wireStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Wire %s not found", wireStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			wire := wireObj.(*SWire)
			input.PreferWire = wire.Id
			zone := wire.GetZone()
			input.PreferZone = zone.Id
			region := zone.GetRegion()
			input.PreferRegion = region.Id
		} else if input.PreferZone != "" {
			zoneStr := input.PreferZone
			zoneObj, err := ZoneManager.FetchById(zoneStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Zone %s not found", zoneStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			zone := zoneObj.(*SZone)
			input.PreferZone = zone.Id
			region := zone.GetRegion()
			input.PreferRegion = region.Id
		} else if input.PreferRegion != "" {
			regionStr := input.PreferRegion
			regionObj, err := CloudregionManager.FetchById(regionStr)
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
