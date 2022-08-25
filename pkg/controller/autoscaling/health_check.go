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

package autoscaling

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

var UnhealthStatus = []string{
	apis.VM_UNKNOWN, apis.VM_SCHEDULE_FAILED, apis.VM_NETWORK_FAILED, apis.VM_DEVICE_FAILED, apis.VM_DISK_FAILED,
	apis.VM_DEPLOY_FAILED, apis.VM_READY, apis.VM_START_FAILED,
}

type sUnnormalGuest struct {
	Id                 string    `json:"id"`
	Status             string    `json:"status"`
	ScalngGroupId      string    `json:"scaling_group_id"`
	CreateCompleteTime time.Time `json:"create_complete_time"`
}

func (asc *SASController) HealthCheckSql() *sqlchemy.SQuery {
	now := time.Now()
	sgSubQ := models.ScalingGroupManager.Query("id").IsTrue("enabled").LT("next_check_time", now).SubQuery()
	sggQ := models.ScalingGroupGuestManager.Query("guest_id", "scaling_group_id", "updated_at").Equals("guest_status", apis.SG_GUEST_STATUS_READY)
	sggSubQ := sggQ.Join(sgSubQ, sqlchemy.Equals(sgSubQ.Field("id"), sggQ.Field("scaling_group_id"))).SubQuery()
	q := models.GuestManager.Query("id", "status").In("status", UnhealthStatus)
	q = q.Join(sggSubQ, sqlchemy.Equals(q.Field("id"), sggSubQ.Field("guest_id")))
	q = q.AppendField(sggSubQ.Field("scaling_group_id"), sggSubQ.Field("updated_at", "create_complete_time"))
	return q
}

func (asc *SASController) CheckInstanceHealth(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	// Fetch all unhealth status instace
	unnormalGuests := make([]sUnnormalGuest, 0, 5)
	scalingGroupIdSet := sets.NewString()
	rows, err := asc.HealthCheckSql().Rows()
	if err != nil {
		log.Errorf("GuestManager's SQuery.Rows: %s", err.Error())
	}
	for rows.Next() {
		var ug sUnnormalGuest
		rows.Scan(&ug.Id, &ug.Status, &ug.ScalngGroupId, &ug.CreateCompleteTime)
		scalingGroupIdSet.Insert(ug.ScalngGroupId)
		unnormalGuests = append(unnormalGuests, ug)
	}
	rows.Close()

	// fetch all ScalingGroup
	scalingGroups := make([]models.SScalingGroup, 0, scalingGroupIdSet.Len())
	q := models.ScalingGroupManager.Query().In("id", scalingGroupIdSet.UnsortedList())
	err = db.FetchModelObjects(models.ScalingGroupManager, q, &scalingGroups)
	if err != nil {
		log.Errorf("unable to fetch ScalingGroup")
		return
	}
	scalingGroupMap := make(map[string]*models.SScalingGroup, len(scalingGroups))
	for i := range scalingGroups {
		scalingGroupMap[scalingGroups[i].GetId()] = &scalingGroups[i]
	}

	// update NextCheckTime for ScalingGroup
	now := time.Now()
	for i := range scalingGroups {
		sg := &scalingGroups[i]
		_, err := db.Update(sg, func() error {
			sg.NextCheckTime = now.Add(time.Duration(sg.HealthCheckCycle) * time.Second)
			return nil
		})
		if err != nil {
			log.Errorf("unable to update NextCheckTime for ScalingGroup '%s'", sg.GetId())
		}
	}

	// request to detach
	readyGuestList := make([]string, 0, 5)
	readyGuestMap := make(map[string]string, 5)

	removeParams := jsonutils.NewDict()
	removeParams.Set("delete_server", jsonutils.JSONTrue)
	removeParams.Set("auto", jsonutils.JSONTrue)
	session := auth.GetSession(ctx, userCred, "")
	for i := range unnormalGuests {
		ug := unnormalGuests[i]
		if ug.CreateCompleteTime.Add(time.Duration(scalingGroupMap[ug.ScalngGroupId].HealthCheckGov) * time.Second).After(now) {
			continue
		}
		if ug.Status == apis.VM_READY {
			readyGuestList = append(readyGuestList, ug.Id)
			readyGuestMap[ug.Id] = ug.ScalngGroupId
			continue
		}
		removeParams.Set("scaling_group", jsonutils.NewString(ug.ScalngGroupId))
		_, err := compute.Servers.PerformAction(session, ug.Id, "detach-scaling-group", removeParams)
		if err != nil {
			log.Errorf("Request Detach Scaling Group failed: %s", err.Error())
		}
	}

	// check NextCheckTime for ScalngGroup

	if len(readyGuestList) > 0 {
		go func() {
			time.Sleep(2 * time.Minute)
			q := models.GuestManager.Query("id").In("id", readyGuestList).Equals("status", apis.VM_READY)
			rows, err := q.Rows()
			if err != nil {
				log.Errorf("GuestManager's SQuery.Rows: %s", err.Error())
			}
			removeGuestList := make([]string, 0, len(readyGuestList)/2)
			for rows.Next() {
				var g string
				rows.Scan(&g)
				removeGuestList = append(removeGuestList, g)
			}
			rows.Close()

			for _, id := range removeGuestList {
				removeParams.Set("scaling_group", jsonutils.NewString(readyGuestMap[id]))
				_, err := compute.Servers.PerformAction(session, id, "detach-scaling-group", removeParams)
				if err != nil {
					log.Errorf("Request Detach Scaling Group failed: %s", err.Error())
				}
			}
		}()
	}
}
