package models

import (
	"context"
	"database/sql"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func RunBatchCreateTask(
	ctx context.Context,
	items []db.IModel,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
	pendingUsage SQuota,
	taskName string,
) {
	taskItems := make([]db.IStandaloneModel, len(items))
	for i, t := range items {
		taskItems[i] = t.(db.IStandaloneModel)
	}
	params := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewParallelTask(ctx, taskName, taskItems, userCred, params, "", "", &pendingUsage)
	if err != nil {
		log.Errorf("%s newTask error %s", taskName, err)
	} else {
		task.ScheduleRun(nil)
	}
}

func ValidateScheduleCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, hypervisor string) (*jsonutils.JSONDict, error) {
	var err error

	if jsonutils.QueryBoolean(data, "baremetal", false) {
		hypervisor = HYPERVISOR_BAREMETAL
	}

	// base validate_create_data
	if (data.Contains("prefer_baremetal") || data.Contains("prefer_host")) && hypervisor != HYPERVISOR_CONTAINER {

		if !userCred.IsAdminAllow(consts.GetServiceType(), GuestManager.KeywordPlural(), policy.PolicyActionPerform, "assign-host") {
			return nil, httperrors.NewNotSufficientPrivilegeError("Only system admin can specify preferred host")
		}
		bmName, _ := data.GetString("prefer_host")
		if len(bmName) == 0 {
			bmName, _ = data.GetString("prefer_baremetal")
		}
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

		if len(hypervisor) > 0 && hypervisor != HOSTTYPE_HYPERVISOR[baremetal.HostType] {
			return nil, httperrors.NewInputParameterError("cannot run hypervisor %s on specified host with type %s", hypervisor, baremetal.HostType)
		}

		if len(hypervisor) == 0 {
			hypervisor = HOSTTYPE_HYPERVISOR[baremetal.HostType]
		}

		if len(hypervisor) == 0 {
			hypervisor = HYPERVISOR_DEFAULT
		}

		_, err = GetDriver(hypervisor).ValidateCreateHostData(ctx, userCred, bmName, baremetal, data)
		if err != nil {
			return nil, err
		}

		data.Set("prefer_baremetal_id", jsonutils.NewString(baremetal.Id))
		data.Set("prefer_host_id", jsonutils.NewString(baremetal.Id))
		zone := baremetal.GetZone()
		data.Set("prefer_zone_id", jsonutils.NewString(zone.Id))
		region := zone.GetRegion()
		data.Set("prefer_region_id", jsonutils.NewString(region.Id))
	} else {
		schedtags := make(map[string]string)
		if data.Contains("aggregate_strategy") {
			err = data.Unmarshal(&schedtags, "aggregate_strategy")
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid aggregate_strategy")
			}
		}
		aggArray := jsonutils.GetArrayOfPrefix(data, "schedtag")
		for idx := 0; idx < len(aggArray); idx += 1 { // data.Contains(fmt.Sprintf("schedtag.%d", idx)); idx += 1 {
			aggStr, _ := aggArray[idx].GetString() // .GetString(fmt.Sprintf("schedtag.%d", idx))
			if len(aggStr) > 0 {
				parts := strings.Split(aggStr, ":")
				if len(parts) >= 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
					schedtags[parts[0]] = parts[1]
				}
			}
		}
		if len(schedtags) > 0 {
			schedtags, err = SchedtagManager.ValidateSchedtags(userCred, schedtags)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid aggregate_strategy: %s", err)
			}
			data.Add(jsonutils.Marshal(schedtags), "aggregate_strategy")
		}

		if data.Contains("prefer_wire") {
			wireStr, _ := data.GetString("prefer_wire")
			wireObj, err := WireManager.FetchById(wireStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Wire %s not found", wireStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			wire := wireObj.(*SWire)
			data.Set("prefer_wire_id", jsonutils.NewString(wire.Id))
			zone := wire.GetZone()
			data.Set("prefer_zone_id", jsonutils.NewString(zone.Id))
			region := zone.GetRegion()
			data.Set("prefer_region_id", jsonutils.NewString(region.Id))
		} else if data.Contains("prefer_zone") {
			zoneStr, _ := data.GetString("prefer_zone")
			zoneObj, err := ZoneManager.FetchById(zoneStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Zone %s not found", zoneStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			zone := zoneObj.(*SZone)
			data.Set("prefer_zone_id", jsonutils.NewString(zone.Id))
			region := zone.GetRegion()
			data.Set("prefer_region_id", jsonutils.NewString(region.Id))
		} else if data.Contains("prefer_region") {
			regionStr, _ := data.GetString("prefer_region")
			regionObj, err := CloudregionManager.FetchById(regionStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Region %s not found", regionStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			region := regionObj.(*SCloudregion)
			data.Set("prefer_region_id", jsonutils.NewString(region.Id))
		}
	}

	// default hypervisor
	if len(hypervisor) == 0 {
		hypervisor = HYPERVISOR_KVM
	}

	if !utils.IsInStringArray(hypervisor, HYPERVISORS) {
		return nil, httperrors.NewInputParameterError("Hypervisor %s not supported", hypervisor)
	}

	data.Add(jsonutils.NewString(hypervisor), "hypervisor")
	return data, nil
}
