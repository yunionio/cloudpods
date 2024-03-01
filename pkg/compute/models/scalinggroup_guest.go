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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SScalingGroupGuestManager struct {
	SGuestJointsManager
}

var ScalingGroupGuestManager *SScalingGroupGuestManager

func init() {
	db.InitManager(func() {
		ScalingGroupGuestManager = &SScalingGroupGuestManager{
			SGuestJointsManager: NewGuestJointsManager(
				SScalingGroupGuest{},
				"scalinggroupguests_tbl",
				"scalinggroupguest",
				"scalinggroupguests",
				ScalingGroupManager,
			),
		}
		ScalingGroupGuestManager.SetVirtualObject(ScalingGroupGuestManager)
	})
}

type SScalingGroupGuest struct {
	SGuestJointsBase

	ScalingGroupId string            `width:"36" charset:"ascii" nullable:"false"`
	GuestStatus    string            `width:"36" charset:"ascii" nullable:"false" index:"true"`
	Manual         tristate.TriState `default:"false"`
}

func (sggm *SScalingGroupGuestManager) GetSlaveFieldName() string {
	return "scaling_group_id"
}

func (sggm *SScalingGroupGuestManager) Attach(ctx context.Context, scaligGroupId, guestId string, manual bool) error {
	sgg := &SScalingGroupGuest{
		SGuestJointsBase: SGuestJointsBase{
			GuestId: guestId,
		},
		ScalingGroupId: scaligGroupId,
		GuestStatus:    compute.SG_GUEST_STATUS_JOINING,
	}
	if manual {
		sgg.Manual = tristate.True
	} else {
		sgg.Manual = tristate.False
	}
	return sggm.TableSpec().Insert(ctx, sgg)
}

func (sgg *SScalingGroupGuest) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, sgg)
}

func (sggm *SScalingGroupGuestManager) Fetch(scalingGroupId, guestId string) ([]SScalingGroupGuest, error) {

	sggs := make([]SScalingGroupGuest, 0)
	q := sggm.Query()
	if len(scalingGroupId) != 0 {
		q = q.Equals("scaling_group_id", scalingGroupId)
	}
	if len(guestId) != 0 {
		q = q.Equals("guest_id", guestId)
	}
	err := db.FetchModelObjects(sggm, q, &sggs)
	return sggs, err
}

func (sgg *SScalingGroupGuest) SetGuestStatus(status string) error {
	if sgg.GuestStatus == status {
		return nil
	}
	_, err := db.Update(sgg, func() error {
		sgg.GuestStatus = status
		sgg.UpdatedAt = time.Now()
		sgg.UpdateVersion += 1
		return nil
	})
	return err
}

func (sggm *SScalingGroupGuestManager) NewQuery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, useRawQuery bool) *sqlchemy.SQuery {
	return sggm.Query()
}

func (sggm *SScalingGroupGuestManager) Query(fields ...string) *sqlchemy.SQuery {
	return sggm.SVirtualJointResourceBaseManager.Query(fields...).NotEquals("guest_status",
		compute.SG_GUEST_STATUS_PENDING_REMOVE)
}
