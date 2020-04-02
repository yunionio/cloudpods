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
	"yunion.io/x/sqlchemy"

	apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var UnhealthStatus = []string{
	apis.VM_UNKNOWN, apis.VM_SCHEDULE_FAILED, apis.VM_NETWORK_FAILED, apis.VM_DEVICE_FAILED, apis.VM_DISK_FAILED,
	apis.VM_DEPLOY_FAILED, apis.VM_READY, apis.VM_START_FAILED,
}

type sUnnormalGuest struct {
	Id            string
	ScalngGroupId string
}

func (asc *SASController) CheckInstanceHealth(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	checkEarliestTime := time.Now().Add(-10 * time.Minute)
	// Fetch all unhealth status instace
	unnormalGuests := make([]sUnnormalGuest, 0, 5)
	sgQ := models.ScalingGroupManager.Query("id").IsFalse("enabled").SubQuery()
	sggQ := models.ScalingGroupGuestManager.Query("guest_id", "scaling_group_id").In("scaling_group_id", sgQ).SubQuery()
	q := models.GuestManager.Query("id").In("stauts", UnhealthStatus).LT("created_at", checkEarliestTime)
	q = q.Join(sggQ, sqlchemy.Equals(q.Field("id"), sggQ.Field("guest_id")))
	q = q.AppendField(sggQ.Field("scaling_group_id"))
	rows, err := q.Rows()
	if err != nil {
		log.Errorf("GuestManager's SQuery.Rows: %s", err.Error())
	}
	for rows.Next() {
		var ug sUnnormalGuest
		rows.Scan(&ug)
		unnormalGuests = append(unnormalGuests, ug)
	}
	rows.Close()

	// request to detach
	removeParams := jsonutils.NewDict()
	removeParams.Set("delete_server", jsonutils.JSONTrue)
	removeParams.Set("auto", jsonutils.JSONTrue)
	session := auth.GetSession(ctx, userCred, "", "")
	for _, ug := range unnormalGuests {
		removeParams.Set("scaling_group", jsonutils.NewString(ug.ScalngGroupId))
		_, err := modules.Servers.PerformAction(session, ug.Id, "detach-scaling-group", removeParams)
		if err != nil {
			log.Errorf("Request Detach Scaling Group failed: %s", err.Error())
		}
	}
}
