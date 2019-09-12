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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHostwireManager struct {
	SHostJointsManager
}

var HostwireManager *SHostwireManager

func init() {
	db.InitManager(func() {
		HostwireManager = &SHostwireManager{
			SHostJointsManager: NewHostJointsManager(
				SHostwire{},
				"hostwires_tbl",
				"hostwire",
				"hostwires",
				WireManager,
			),
		}
		HostwireManager.SetVirtualObject(HostwireManager)
	})
}

type SHostwire struct {
	SHostJointsBase

	Bridge    string `width:"16" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(16, charset='ascii'), nullable=False)
	Interface string `width:"16" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(16, charset='ascii'), nullable=False)
	IsMaster  bool   `nullable:"true" default:"false" list:"admin" update:"admin" create:"admin_optional"`             // Column(Boolean, nullable=True, default=False)
	MacAddr   string `width:"18" charset:"ascii" list:"admin" update:"admin" create:"admin_required"`                  // Column(VARCHAR(18, charset='ascii'))

	HostId string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // = Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	WireId string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
}

func (manager *SHostwireManager) GetMasterFieldName() string {
	return "host_id"
}

func (manager *SHostwireManager) GetSlaveFieldName() string {
	return "wire_id"
}

func (joint *SHostwire) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SHostwire) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SHostwire) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra)
}

func (self *SHostwire) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SHostJointsBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra), nil
}

func (hw *SHostwire) GetWire() *SWire {
	wire, _ := WireManager.FetchById(hw.WireId)
	if wire != nil {
		return wire.(*SWire)
	}
	return nil
}

func (hw *SHostwire) GetHost() *SHost {
	host, _ := HostManager.FetchById(hw.HostId)
	if host != nil {
		return host.(*SHost)
	}
	return nil
}

func (hw *SHostwire) getExtraDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	wire := hw.GetWire()
	if wire != nil {
		extra.Add(jsonutils.NewInt(int64(wire.Bandwidth)), "bandwidth")
	}
	return extra
}

func (self *SHostwire) GetGuestnicsCount() (int, error) {
	guestnics := GuestnetworkManager.Query().SubQuery()
	guests := GuestManager.Query().SubQuery()
	nets := NetworkManager.Query().SubQuery()

	q := guestnics.Query()
	q = q.Join(guests, sqlchemy.AND(sqlchemy.IsFalse(guests.Field("deleted")),
		sqlchemy.Equals(guests.Field("id"), guestnics.Field("guest_id")),
		sqlchemy.Equals(guests.Field("host_id"), self.HostId)))
	q = q.Join(nets, sqlchemy.AND(sqlchemy.IsFalse(nets.Field("deleted")),
		sqlchemy.Equals(nets.Field("id"), guestnics.Field("network_id")),
		sqlchemy.Equals(nets.Field("wire_id"), self.WireId)))

	return q.CountWithError()
}

func (self *SHostwire) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := self.GetGuestnicsCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetGuestnicsCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("guest on the host are using networks on this wire")
	}
	return self.SHostJointsBase.ValidateDeleteCondition(ctx)
}

func (self *SHostwire) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SHostwire) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SHostwireManager) FilterByParams(q *sqlchemy.SQuery, params jsonutils.JSONObject) *sqlchemy.SQuery {
	macStr := jsonutils.GetAnyString(params, []string{"mac", "mac_addr"})
	if len(macStr) > 0 {
		q = q.Filter(sqlchemy.Equals(q.Field("mac_addr"), macStr))
	}
	return q
}

func (manager *SHostwireManager) FetchByHostIdAndMac(hostId string, mac string) (*SHostwire, error) {
	hw, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("host_id", hostId).Equals("mac_addr", mac)
	err = q.First(hw)
	if err != nil {
		return nil, err
	}
	return hw.(*SHostwire), nil
}
