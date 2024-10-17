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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SHostwireManagerDeprecated struct {
	SHostJointsManager
	SWireResourceBaseManager
}

var HostwireManagerDeprecated *SHostwireManagerDeprecated

func init() {
	db.InitManager(func() {
		HostwireManagerDeprecated = &SHostwireManagerDeprecated{
			SHostJointsManager: NewHostJointsManager(
				"host_id",
				SHostwireDeprecated{},
				"hostwires_tbl",
				"hostwire",
				"hostwires",
				WireManager,
			),
		}
		HostwireManagerDeprecated.SetVirtualObject(HostwireManagerDeprecated)
	})
}

// +onecloud:model-api-gen
type SHostwireDeprecated struct {
	SHostJointsBase

	Bridge string `width:"64" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_required"`
	// 接口名称
	Interface string `width:"64" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_required"`
	// 是否是主地址
	IsMaster bool `nullable:"true" default:"false" list:"domain" update:"domain" create:"domain_optional"`
	// MAC地址
	MacAddr string `width:"18" charset:"ascii" list:"domain" update:"domain" create:"domain_required"`

	// 宿主机Id
	HostId string `width:"128" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
	// 二层网络Id
	WireId string `width:"128" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
}

func (manager *SHostwireManagerDeprecated) GetMasterFieldName() string {
	return "host_id"
}

func (manager *SHostwireManagerDeprecated) GetSlaveFieldName() string {
	return "wire_id"
}

func (manager *SHostwireManagerDeprecated) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostwireDetails {
	rows := make([]api.HostwireDetails, len(objs))

	hostRows := manager.SHostJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	wireIds := make([]string, len(rows))

	for i := range rows {
		rows[i] = api.HostwireDetails{
			HostJointResourceDetails: hostRows[i],
		}
		wireIds[i] = objs[i].(*SHostwireDeprecated).WireId
	}

	wires := make(map[string]SWire)
	err := db.FetchStandaloneObjectsByIds(WireManager, wireIds, &wires)
	if err != nil {
		log.Errorf("db.FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if wire, ok := wires[wireIds[i]]; ok {
			rows[i].Wire = wire.Name
			rows[i].Bandwidth = wire.Bandwidth
		}
	}

	return rows
}

func (hw *SHostwireDeprecated) GetWire() *SWire {
	wire, _ := WireManager.FetchById(hw.WireId)
	if wire != nil {
		return wire.(*SWire)
	}
	return nil
}

func (hw *SHostwireDeprecated) GetHost() *SHost {
	host, _ := HostManager.FetchById(hw.HostId)
	if host != nil {
		return host.(*SHost)
	}
	return nil
}

func (self *SHostwireDeprecated) GetGuestnicsCount() (int, error) {
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

func (self *SHostwireDeprecated) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	cnt, err := self.GetGuestnicsCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetGuestnicsCount fail %s", err)
	}
	if cnt > 0 {
		// check if this is the last one
		// host := self.GetHost()
		// if len(host.getHostwiresOfId(self.WireId)) == 1 {
		//	return httperrors.NewNotEmptyError("guest on the host are using networks on this wire")
		// }
	}
	return self.SHostJointsBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SHostwireDeprecated) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SHostwireDeprecated) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	host := self.GetHost()
	if host == nil {
		log.Errorf("no host found??")
		return
	}
	netif := host.GetNetInterface(self.MacAddr, 1)
	if netif == nil {
		log.Errorf("no netinterface for %s", self.MacAddr)
		return
	}
	err := host.DisableNetif(ctx, userCred, netif, false)
	if err != nil {
		log.Errorf("host.DisableNetif fail %s", err)
		return
	}
	err = netif.UnsetWire()
	if err != nil {
		log.Errorf("netif.UnsetWire fail %s", err)
		return
	}
}

func (self *SHostwireDeprecated) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SHostwireManagerDeprecated) FilterByParams(q *sqlchemy.SQuery, params jsonutils.JSONObject) *sqlchemy.SQuery {
	macStr := jsonutils.GetAnyString(params, []string{"mac", "mac_addr"})
	if len(macStr) > 0 {
		q = q.Filter(sqlchemy.Equals(q.Field("mac_addr"), macStr))
	}
	return q
}

func (manager *SHostwireManagerDeprecated) FetchByHostIdAndMac(hostId string, mac string) (*SHostwireDeprecated, error) {
	hw, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("host_id", hostId).Equals("mac_addr", mac)
	err = q.First(hw)
	if err != nil {
		return nil, err
	}
	return hw.(*SHostwireDeprecated), nil
}

func (manager *SHostwireManagerDeprecated) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostwireListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.ListItemFilter(ctx, q, userCred, query.HostJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SWireResourceBaseManager.ListItemFilter(ctx, q, userCred, query.WireFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemFilter")
	}

	if len(query.Bridge) > 0 {
		q = q.In("bridge", query.Bridge)
	}
	if len(query.Interface) > 0 {
		q = q.In("interface", query.Interface)
	}
	if query.IsMaster != nil {
		if *query.IsMaster {
			q = q.IsTrue("is_master")
		} else {
			q = q.IsFalse("is_master")
		}
	}
	if len(query.MacAddr) > 0 {
		q = q.In("mac_addr", query.MacAddr)
	}

	return q, nil
}

func (manager *SHostwireManagerDeprecated) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostwireListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.OrderByExtraFields(ctx, q, userCred, query.HostJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SWireResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.WireFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SWireResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SHostwireManagerDeprecated) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SHostJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SWireResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SWireResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (hw *SHostwireDeprecated) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	hw.SHostJointsBase.PostCreate(ctx, userCred, ownerId, query, data)
	hw.syncClassMetadata(ctx, userCred)
}

func (hw *SHostwireDeprecated) syncClassMetadata(ctx context.Context, userCred mcclient.TokenCredential) error {
	host := hw.GetHost()
	wire := hw.GetWire()
	err := db.InheritFromTo(ctx, userCred, wire, host)
	if err != nil {
		log.Errorf("Inherit class metadata from host to wire fail: %s", err)
		return errors.Wrap(err, "InheritFromTo")
	}
	return nil
}
