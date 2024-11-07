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

// +onecloud:swagger-gen-model-singular=tap_flow
// +onecloud:swagger-gen-model-plural=tap_flows
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
		tapObj, err := NetTapServiceManager.FetchByIdOrName(ctx, userCred, query.TapId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s not found", NetTapServiceManager.Keyword(), query.TapId)
			} else {
				return nil, errors.Wrap(err, "NetTapServiceManager.FetchByIdOrName")
			}
		}
		q = q.Equals("tap_id", tapObj.GetId())
	}
	if len(query.HostId) > 0 {
		hostObj, err := HostManager.FetchByIdOrName(ctx, userCred, query.HostId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(HostManager.Keyword(), query.HostId)
			} else {
				return nil, errors.Wrap(err, "HostManager.FetchHostById")
			}
		}
		q = man.filterByHostId(q, hostObj.GetId())
	}
	return q, nil
}

func (man *SNetTapFlowManager) filterByHostId(q *sqlchemy.SQuery, hostId string) *sqlchemy.SQuery {
	guestIdQ := GuestManager.Query("id").Equals("host_id", hostId).SubQuery()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("type"), api.TapFlowVSwitch),
			sqlchemy.Equals(q.Field("source_id"), hostId),
		),
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("type"), api.TapFlowGuestNic),
			sqlchemy.In(q.Field("source_id"), guestIdQ),
		),
	))
	return q
}

func (man *SNetTapFlowManager) getEnabledTapFlowsOfTap(tapId string) ([]SNetTapFlow, error) {
	return man.getEnabledTapFlows("", tapId)
}

func (man *SNetTapFlowManager) getEnabledTapFlowsOnHost(hostId string) ([]SNetTapFlow, error) {
	return man.getEnabledTapFlows(hostId, "")
}

func (man *SNetTapFlowManager) getEnabledTapFlows(hostId string, tapId string) ([]SNetTapFlow, error) {
	q := man.Query().IsTrue("enabled")
	if len(hostId) > 0 {
		q = man.filterByHostId(q, hostId)
	}
	if len(tapId) > 0 {
		q = q.Equals("tap_id", tapId)
	}

	// filter by enabled tap
	tapQ := NetTapServiceManager.Query().IsTrue("enabled").SubQuery()
	q = q.Join(tapQ, sqlchemy.Equals(q.Field("tap_id"), tapQ.Field("id")))

	flows := make([]SNetTapFlow, 0)
	err := db.FetchModelObjects(man, q, &flows)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return flows, nil
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
		rows[i] = flow.getMoreDetails(ctx, rows[i])
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
		if host != nil {
			details.Source = host.Name
			details.SourceIps = host.AccessIp
		}
		wire := WireManager.FetchWireById(flow.NetId)
		if wire != nil {
			details.Net = wire.Name
		}
	case api.TapFlowGuestNic:
		guest := GuestManager.FetchGuestById(flow.SourceId)
		if guest != nil {
			details.Source = guest.Name
			ret := fetchGuestIPs([]string{flow.SourceId}, tristate.False)
			details.SourceIps = strings.Join(ret[flow.SourceId], ",")
		}
		netObj, _ := NetworkManager.FetchById(flow.NetId)
		if netObj != nil {
			details.Net = netObj.GetName()
		}
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
	tapObj, err := NetTapServiceManager.FetchByIdOrName(ctx, userCred, input.TapId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewResourceNotFoundError2(NetTapServiceManager.Keyword(), input.TapId)
		} else {
			return input, errors.Wrap(err, "NetTapServiceManager.FetchByIdOrName")
		}
	}
	tap := tapObj.(*SNetTapService)
	input.TapId = tap.Id
	switch input.Type {
	case api.TapFlowVSwitch:
		hostObj, err := HostManager.FetchByIdOrName(ctx, userCred, input.HostId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2(HostManager.Keyword(), input.HostId)
			} else {
				return input, errors.Wrap(err, "HostManager.FetchByIdOrName")
			}
		}
		wireObj, err := WireManager.FetchByIdOrName(ctx, userCred, input.WireId)
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
		netifs := host.getNetifsOnWire(wire.Id)
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
		if input.VlanId != nil && (*input.VlanId <= 0 || *input.VlanId > 4095) {
			return input, errors.Wrapf(httperrors.ErrInputParameter, "invalid vlan id %d", *input.VlanId)
		}
	case api.TapFlowGuestNic:
		guestObj, err := GuestManager.FetchByIdOrName(ctx, userCred, input.GuestId)
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
		input.VlanId = nil
		// check loop
		if tap.Type == api.TapServiceGuest && tap.TargetId == input.SourceId {
			return input, errors.Wrap(httperrors.ErrInputParameter, "cannot tap trafic from guest itself")
		}
	default:
		return input, errors.Wrapf(httperrors.ErrInputParameter, "invalid flow type %s", input.Type)
	}
	// check duplicity
	dupCnt, err := manager.Query().Equals("tap_id", tap.Id).Equals("type", input.Type).Equals("source_id", input.SourceId).Equals("net_id", input.NetId).CountWithError()
	if err != nil {
		return input, errors.Wrap(err, "query duplicity")
	}
	if dupCnt > 0 {
		return input, errors.Wrap(httperrors.ErrConflict, "this source has been added")
	}
	if len(input.Direction) == 0 {
		input.Direction = api.TapFlowDirectionBoth
	}
	if !utils.IsInStringArray(input.Direction, api.TapFlowDirections) {
		return input, errors.Wrapf(httperrors.ErrNotSupported, "unsupported direction %s", input.Direction)
	}
	if input.Enabled == nil {
		trueVal := true
		input.Enabled = &trueVal
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

func (flow *SNetTapFlow) getTap() *SNetTapService {
	srvObj, _ := NetTapServiceManager.FetchById(flow.TapId)
	return srvObj.(*SNetTapService)
}

func (flow *SNetTapFlow) getTapHostIp() string {
	return flow.getTap().getTapHostIp()
}

func (flow *SNetTapFlow) getMirrorConfig(needTapHostIp bool) (api.SMirrorConfig, error) {
	ret := api.SMirrorConfig{}

	if needTapHostIp {
		ret.TapHostIp = flow.getTapHostIp()
	}

	var hostId, wireId string
	switch flow.Type {
	case api.TapFlowVSwitch:
		hostId = flow.SourceId
		wireId = flow.NetId
	case api.TapFlowGuestNic:
		guest := GuestManager.FetchGuestById(flow.SourceId)
		if guest == nil {
			// guest has been deleted?
			return ret, errors.Wrap(errors.ErrNotFound, "source not found")
		}
		gn, err := guest.GetGuestnetworkByMac(flow.MacAddr)
		if err != nil {
			return ret, errors.Wrap(err, "GetGuestnetworkByMac")
		}
		ret.Port = gn.Ifname
		hostId = guest.HostId
		net, err := gn.GetNetwork()
		if err != nil {
			return ret, errors.Wrapf(err, "GetNetwork")
		}
		if net.IsClassic() {
			wireId = net.WireId
		} else {
			ret.Bridge = api.HostVpcBridge
		}
	}
	host := HostManager.FetchHostById(hostId)
	if len(wireId) > 0 {
		// classic network
		netifs := host.getNetifsOnWire(wireId)
		if len(netifs) == 0 {
			return ret, errors.Error("invalid flow? no valid hostwire")
		}
		if len(netifs) > 1 {
			return ret, errors.Error("invalid flow? host and wire have multiple hostwires")
		}
		ret.Bridge = netifs[0].Bridge
	}
	ret.HostIp = host.AccessIp
	ret.FlowId = flow.FlowId
	ret.VlanId = flow.VlanId
	ret.Direction = flow.Direction
	return ret, nil
}

func (manager *SNetTapFlowManager) removeTapFlowsByGuestId(ctx context.Context, userCred mcclient.TokenCredential, sourceId string) error {
	return manager.removeTapFlows(ctx, userCred, api.TapFlowGuestNic, sourceId)
}

func (manager *SNetTapFlowManager) removeTapFlowsByHostId(ctx context.Context, userCred mcclient.TokenCredential, sourceId string) error {
	return manager.removeTapFlows(ctx, userCred, api.TapFlowVSwitch, sourceId)
}

func (manager *SNetTapFlowManager) removeTapFlows(ctx context.Context, userCred mcclient.TokenCredential, srvType string, targetId string) error {
	srvs, err := manager.getTapFlows(srvType, targetId)
	if err != nil {
		return errors.Wrap(err, "getTapServicesByHostId")
	}
	for i := range srvs {
		err := srvs[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

func (manager *SNetTapFlowManager) getTapFlows(srvType string, sourceId string) ([]SNetTapFlow, error) {
	q := manager.Query()
	q = q.Equals("type", srvType)
	q = q.Equals("source_id", sourceId)
	ret := make([]SNetTapFlow, 0)
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return ret, nil
}
