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
)

type SGroupnetworkManager struct {
	SGroupJointsManager
}

var GroupnetworkManager *SGroupnetworkManager

func init() {
	db.InitManager(func() {
		GroupnetworkManager = &SGroupnetworkManager{
			SGroupJointsManager: NewGroupJointsManager(
				SGroupnetwork{},
				"groupnetworks_tbl",
				"groupnetwork",
				"groupnetworks",
				NetworkManager,
			),
		}
		GroupnetworkManager.SetVirtualObject(GroupnetworkManager)
	})
}

type SGroupnetwork struct {
	SGroupJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	IpAddr string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)
	// # ip6_addr = Column(VARCHAR(64, charset='ascii'), nullable=True)

	Index int8 `nullable:"false" default:"0" list:"user" list:"user" update:"user" create:"optional"` // Column(TINYINT, nullable=False, default=0)

	EipId string `width:"36" charset:"ascii" nullable:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
}

func (manager *SGroupnetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (joint *SGroupnetwork) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SGroupnetwork) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SGroupnetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.GroupnetworkDetails, error) {
	var err error
	out := api.GroupnetworkDetails{}
	out.ModelBaseDetails, err = self.SGroupJointsBase.GetExtraDetails(ctx, userCred, query, isList)
	if err != nil {
		return out, err
	}
	out.Instancegroup, out.Network = db.JointModelExtra(self)
	return out, nil
}

func (self *SGroupnetwork) GetNetwork() *SNetwork {
	obj, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return obj.(*SNetwork)
}

func (self *SGroupnetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGroupnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}
