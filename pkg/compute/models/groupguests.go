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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGroupguestManager struct {
	SGroupJointsManager
}

var GroupguestManager *SGroupguestManager

func init() {
	db.InitManager(func() {
		GroupguestManager = &SGroupguestManager{
			SGroupJointsManager: NewGroupJointsManager(
				SGroupguest{},
				"guestgroups_tbl",
				"groupguest",
				"groupguests",
				GuestManager,
			),
		}
		GroupguestManager.SetVirtualObject(GroupguestManager)
	})
}

type SGroupguest struct {
	SGroupJointsBase

	Tag     string `width:"256" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(256, charset='ascii'), nullable=True)
	GuestId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`               // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SGroupguestManager) GetSlaveFieldName() string {
	return "guest_id"
}

func (joint *SGroupguest) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SGroupguest) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SGroupguest) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SGroupJointsBase.GetCustomizeColumns(ctx, userCred, query)
	return db.JointModelExtra(self, extra)
}

func (self *SGroupguest) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SGroupJointsBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return db.JointModelExtra(self, extra), nil
}

func (self *SGroupguest) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGroupguest) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (self *SGroupguestManager) FetchByGuestId(guestId string) ([]SGroupguest, error) {
	q := self.Query().Equals("guest_id", guestId)
	joints := make([]SGroupguest, 0, 1)
	err := db.FetchModelObjects(self, q, &joints)
	if err != nil {
		return nil, err
	}
	return joints, err
}

func (self *SGroupguestManager) Attach(ctx context.Context, groupId, guestId string) (*SGroupguest, error) {

	joint := &SGroupguest{}
	joint.GuestId = guestId
	joint.GroupId = groupId

	err := self.TableSpec().Insert(joint)
	if err != nil {
		return nil, err
	}
	joint.SetModelManager(self, joint)
	return joint, nil
}
