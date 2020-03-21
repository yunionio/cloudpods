package models

import (
	"context"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/tristate"
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
	GuestStatus    string            `width:"36" charset:"ascii" nullable:"false"`
	Manual         tristate.TriState `nullable:"false" default:"false"`
}

func (sggm *SScalingGroupGuestManager) Attach(ctx context.Context, scaligGroupId, guestId string, manual bool) error {
	sgg := &SScalingGroupGuest{
		SGuestJointsBase: SGuestJointsBase{
			GuestId:                   guestId,
		},
		ScalingGroupId:   scaligGroupId,
		GuestStatus:      compute.SG_GUEST_STATUS_READY,
	}
	if manual {
		sgg.Manual = tristate.True
	} else {
		sgg.Manual = tristate.False
	}
	return sggm.TableSpec().Insert(sgg)
}

func (sgg *SScalingGroupGuest) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, sgg)
}

func (sggm *SScalingGroupGuestManager) Fetch(scalingGroupId, networkId string) ([]SScalingGroupGuest, error) {

	sggs := make([]SScalingGroupGuest, 0)
	q := sggm.Query()
	if len(scalingGroupId) != 0 {
		q = q.Equals("scaling_group_id", scalingGroupId)
	}
	if len(networkId) != 0 {
		q = q.Equals("network_id", networkId)
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
		return nil
	})
	return err
}
