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
	"database/sql"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNetTapFlowManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var NetTapFlowManager *SNetTapFlowManager

func init() {
	NetTapFlowManager = &SNetTapFlowManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SNetTapFlow{},
			"net_tap_flows_tbl",
			"tap_flow",
			"tap_flows",
		),
	}
	NetTapFlowManager.SetVirtualObject(NetTapFlowManager)
}

type SNetTapFlow struct {
	db.SEnabledStatusStandaloneResourceBase

	TapId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`

	Type string `width:"10" charset:"ascii" list:"admin" create:"admin_required"`

	SourceId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`

	NetId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`

	MacAddr string `width:"18" charset:"ascii" list:"admin" create:"admin_optional"`

	VlanId int `nullable:"true" list:"admin" create:"admin_optional"`

	Direction string `width:"6" charset:"ascii" list:"admin" create:"admin_required" default:"BOTH"`

	FlowId uint16 `nullable:"false" list:"admin"`
}

func (man *SNetTapFlowManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetTapFlowListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}
	if len(query.TapId) > 0 {
		tapObj, err := NetTapServiceManager.FetchByIdOrName(userCred, query.TapId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s not found", NetTapServiceManager.Keyword(), query.TapId)
			} else {
				return nil, errors.Wrap(err, "NetTapServiceManager.FetchByIdOrName")
			}
		}
		q = q.Equals("tap_id", tapObj.GetId())
	}
	return q, nil
}

func (man *SNetTapFlowManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetTapFlowListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SNetTapFlowManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SNetTapFlowManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NetTapFlowDetails {
	rows := make([]api.NetTapFlowDetails, len(objs))
	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	tapIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.NetTapFlowDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
		}
		flow := objs[i].(*SNetTapFlow)
		tapIds[i] = flow.TapId
	}
	tapIdMap, err := db.FetchIdNameMap2(NetTapServiceManager, tapIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail: %s", err)
		return rows
	}
	for i := range rows {
		if name, ok := tapIdMap[tapIds[i]]; ok {
			rows[i].Tap = name
		}
	}
	return rows
}

func (flow *SNetTapFlow) getMoreDetails(ctx context.Context, details api.NetTapFlowDetails) api.NetTapFlowDetails {
	switch flow.Type {
	case api.TapFlowVSwitch:
		host := HostManager.FetchHostById(flow.SourceId)
		details.Source = host.Name
		details.SourceIps = host.AccessIp
		wire := WireManager.FetchWireById(flow.NetId)
		details.Net = wire.Name
	case api.TapFlowGuestNic:
		guest := GuestManager.FetchGuestById(flow.SourceId)
		details.Source = guest.Name
		ret := fetchGuestIPs([]string{flow.SourceId}, tristate.False)
		details.SourceIps = strings.Join(ret[flow.SourceId], ",")
		netObj, _ := NetworkManager.FetchById(flow.NetId)
		details.Net = netObj.GetName()
	}
	return details
}

func (manager *SNetTapFlowManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.NetTapFlowCreateInput,
) (api.NetTapFlowCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(")
	}
	tapObj, err := NetTapServiceManager.FetchByIdOrName(userCred, input.TapId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewResourceNotFoundError2(NetTapServiceManager.Keyword(), input.TapId)
		} else {
			return input, errors.Wrap(err, "NetTapServiceManager.FetchByIdOrName")
		}
	}
	input.TapId = tapObj.GetId()
	switch input.Type {
	case api.TapFlowVSwitch:
		hostObj, err := HostManager.FetchByIdOrName(userCred, input.HostId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2(HostManager.Keyword(), input.HostId)
			} else {
				return input, errors.Wrap(err, "HostManager.FetchByIdOrName")
			}
		}
		wireObj, err := WireManager.FetchByIdOrName(userCred, input.WireId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2(WireManager.Keyword(), input.WireId)
			} else {
				return input, errors.Wrap(err, "WireManager.FetchByIdOrName")
			}
		}
		host := hostObj.(*SHost)
		if host.HostType != api.HOST_TYPE_HYPERVISOR {
			return input, errors.Wrapf(httperrors.ErrNotSupported, "host type %s not supported", host.HostType)
		}
		wire := wireObj.(*SWire)
		netifs := host.GetNetifsOnWire(wire)
		if len(netifs) == 0 {
			return input, errors.Wrapf(httperrors.ErrInvalidStatus, "host %s and wire %s not attached", input.HostId, input.WireId)
		}
		ipmiCnt := 0
		nicCnt := 0
		for _, netif := range netifs {
			if netif.NicType == api.NIC_TYPE_IPMI {
				ipmiCnt++
			}
			nicCnt++
		}
		if ipmiCnt == nicCnt {
			return input, errors.Wrapf(httperrors.ErrInvalidStatus, "host %s and wire %s attached with IPMI links", input.HostId, input.WireId)
		}
		input.SourceId = host.Id
		input.MacAddr = ""
		input.NetId = wire.Id
		if input.VlanId <= 0 || input.VlanId > 4095 {
			return input, errors.Wrapf(httperrors.ErrInputParameter, "invalid vlan id %d", input.VlanId)
		}
	case api.TapFlowGuestNic:
		guestObj, err := GuestManager.FetchByIdOrName(userCred, input.GuestId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2(GuestManager.Keyword(), input.GuestId)
			} else {
				return input, errors.Wrap(err, "GuestManager.FetchByIdOrName")
			}
		}
		guest := guestObj.(*SGuest)
		if guest.Hypervisor != api.HYPERVISOR_KVM {
			return input, errors.Wrapf(httperrors.ErrInvalidStatus, "hypervisor %s not supported", guest.Hypervisor)
		}
		gns, err := GuestnetworkManager.FetchByGuestId(guest.Id)
		if err != nil {
			return input, errors.Wrap(err, "GuestnetworkManager.FetchByGuestId")
		}
		var gn *SGuestnetwork
		if len(input.IpAddr) == 0 && len(input.MacAddr) == 0 {
			if len(gns) == 1 {
				gn = &gns[0]
			} else {
				return input, errors.Wrap(httperrors.ErrInputParameter, "either ip_addr or mac_addr should be specified")
			}
		} else {
			for i := range gns {
				if (len(input.IpAddr) > 0 && input.IpAddr == gns[i].IpAddr) || (len(input.MacAddr) > 0 && input.MacAddr == gns[i].MacAddr) {
					gn = &gns[i]
					break
				}
			}
			if gn == nil {
				return input, errors.Wrap(httperrors.ErrNotFound, "Guest network not found")
			}
		}
		input.SourceId = guest.Id
		input.MacAddr = gn.MacAddr
		input.NetId = gn.NetworkId
		input.VlanId = 0
	default:
		return input, errors.Wrapf(httperrors.ErrInputParameter, "invalid flow type %s", input.Type)
	}
	if len(input.Direction) == 0 {
		input.Direction = api.TapFlowDirectionBoth
	}
	if !utils.IsInStringArray(input.Direction, api.TapFlowDirections) {
		return input, errors.Wrapf(httperrors.ErrNotSupported, "unsupported direction %s", input.Direction)
	}
	return input, nil
}

func (tap *SNetTapFlow) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	// generate flowId
	err := func() error {
		lockman.LockClass(ctx, NetTapFlowManager, "")
		defer lockman.ReleaseClass(ctx, NetTapFlowManager, "")

		flowId, err := NetTapFlowManager.getFreeFlowId()
		if err != nil {
			return errors.Wrap(err, "getFreeFlowId")
		}
		tap.FlowId = flowId
		return nil
	}()
	if err != nil {
		return errors.Wrap(err, "generate flow id")
	}
	return tap.SEnabledStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (manager *SNetTapFlowManager) getFreeFlowId() (uint16, error) {
	flowIds := make([]struct {
		FlowId uint16 `json:"flow_id"`
	}, 0)
	q := manager.Query("flow_id").Asc("flow_id")
	err := q.All(&flowIds)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return 0, errors.Wrap(err, "queryAll")
	}
	if len(flowIds) == 0 {
		return api.TapFlowIdMin, nil
	}
	if flowIds[0].FlowId > api.TapFlowIdMin {
		return flowIds[0].FlowId - 1, nil
	}
	if flowIds[len(flowIds)-1].FlowId < api.TapFlowIdMax {
		return flowIds[len(flowIds)-1].FlowId + 1, nil
	}
	for i := 0; i < len(flowIds)-1; i++ {
		if flowIds[i].FlowId+1 < flowIds[i+1].FlowId {
			return flowIds[i].FlowId + 1, nil
		}
	}
	return 0, errors.Wrap(httperrors.ErrOutOfResource, "run out of flow id!!!")
}
