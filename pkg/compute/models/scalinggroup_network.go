package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var ScalingGroupNetworkManager *SScalingGroupNetworkManager

func init() {
	db.InitManager(func() {
		ScalingGroupNetworkManager = &SScalingGroupNetworkManager{
			SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
				SScalingGroupNetwork{},
				"scalinggroupnetworks_tbl",
				"scalinggroupnetwork",
				"scalinggroupnetworks",
				ScalingGroupManager,
				NetworkManager,
			),
		}
	})
}

type SScalingGroupNetworkManager struct {
	db.SVirtualJointResourceBaseManager
}

type SScalingGroupNetwork struct {
	db.SVirtualJointResourceBase

	ScalingGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	NetworkId      string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (sgnm *SScalingGroupNetworkManager) Attach(ctx context.Context, scalingGroupId, networkId string) error {
	sgn := &SScalingGroupNetwork{
		ScalingGroupId: scalingGroupId,
		NetworkId:      networkId,
	}
	return sgnm.TableSpec().Insert(sgn)
}

func (sgn *SScalingGroupNetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, sgn)
}
