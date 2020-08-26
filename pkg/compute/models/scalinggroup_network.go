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
	return sgnm.TableSpec().Insert(ctx, sgn)
}

func (sgn *SScalingGroupNetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, sgn)
}
