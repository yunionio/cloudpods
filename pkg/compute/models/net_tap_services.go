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
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=tap_service
// +onecloud:swagger-gen-model-plural=tap_services
type SNetTapServiceManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var NetTapServiceManager *SNetTapServiceManager

func init() {
	NetTapServiceManager = &SNetTapServiceManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SNetTapService{},
			"net_tap_services_tbl",
			"tap_service",
			"tap_services",
		),
	}
	NetTapServiceManager.SetVirtualObject(NetTapServiceManager)
	NetTapServiceManager.TableSpec().AddIndex(false, "mac_addr", "deleted")
}

type SNetTapService struct {
	db.SEnabledStatusStandaloneResourceBase

	// 流量采集端类型，虚拟机(guest)还是宿主机(host)
	Type string `width:"10" charset:"ascii" list:"admin" create:"admin_required"`
	// 接受流量的目标ID，如果type=host，是hostId，如果type=guest，是guestId
	TargetId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
	// 接受流量的Mac地址
	MacAddr string `width:"18" charset:"ascii" list:"admin" create:"admin_optional"`
	// 网卡名称
	Ifname string `width:"16" charset:"ascii" nullable:"true" list:"admin"`
}

func (man *SNetTapServiceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetTapServiceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
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

func (man *SNetTapServiceManager) filterByHostId(q *sqlchemy.SQuery, hostId string) *sqlchemy.SQuery {
	guestIdQ := GuestManager.Query("id").Equals("host_id", hostId).SubQuery()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("type"), api.TapServiceHost),
			sqlchemy.Equals(q.Field("target_id"), hostId),
		),
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("type"), api.TapServiceGuest),
			sqlchemy.In(q.Field("target_id"), guestIdQ),
		),
	))
	return q
}

func (man *SNetTapServiceManager) getEnabledTapServiceOnHost(hostId string) ([]SNetTapService, error) {
	q := man.Query().IsTrue("enabled")
	q = man.filterByHostId(q, hostId)
	srvs := make([]SNetTapService, 0)
	err := db.FetchModelObjects(man, q, &srvs)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return srvs, nil
}

func (man *SNetTapServiceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetTapServiceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	if db.NeedOrderQuery([]string{query.OrderByIp}) {
		hSQ := HostManager.Query("id", "access_ip").SubQuery()
		q = q.LeftJoin(hSQ, sqlchemy.Equals(q.Field("target_id"), hSQ.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(hSQ.Field("access_ip"))
		db.OrderByFields(q, []string{query.OrderByIp}, []sqlchemy.IQueryField{q.Field("access_ip")})
	}
	if db.NeedOrderQuery([]string{query.OrderByFlowCount}) {
		ntfQ := NetTapFlowManager.Query()
		ntfQ = ntfQ.AppendField(ntfQ.Field("tap_id"), sqlchemy.COUNT("flow_count", ntfQ.Field("tap_id")))
		ntfQ = ntfQ.GroupBy("tap_id")
		ntfSQ := ntfQ.SubQuery()
		q = q.LeftJoin(ntfSQ, sqlchemy.Equals(q.Field("id"), ntfSQ.Field("tap_id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(ntfSQ.Field("flow_count"))
		db.OrderByFields(q, []string{query.OrderByFlowCount}, []sqlchemy.IQueryField{q.Field("flow_count")})
	}
	return q, nil
}

func (man *SNetTapServiceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SNetTapServiceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NetTapServiceDetails {
	rows := make([]api.NetTapServiceDetails, len(objs))
	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.NetTapServiceDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
		}
		rows[i] = objs[i].(*SNetTapService).getMoreDetails(ctx, rows[i])
	}
	return rows
}

func (srv *SNetTapService) getMoreDetails(ctx context.Context, details api.NetTapServiceDetails) api.NetTapServiceDetails {
	var err error
	switch srv.Type {
	case api.TapServiceHost:
		host := HostManager.FetchHostById(srv.TargetId)
		details.Target = host.Name
		details.TargetIps = host.AccessIp
	case api.TapServiceGuest:
		guest := GuestManager.FetchGuestById(srv.TargetId)
		details.Target = guest.Name
		ret := fetchGuestIPs([]string{srv.TargetId}, tristate.False)
		details.TargetIps = strings.Join(ret[srv.TargetId], ",")
	}
	details.FlowCount, err = srv.getFlowsCount()
	if err != nil {
		log.Errorf("getFlowsCount %s", err)
	}
	return details
}

func (srv *SNetTapService) getFlowsQuery() *sqlchemy.SQuery {
	return NetTapFlowManager.Query().Equals("tap_id", srv.Id)
}

func (srv *SNetTapService) getFlowsCount() (int, error) {
	return srv.getFlowsQuery().CountWithError()
}

func (srv *SNetTapService) getFlows() ([]SNetTapFlow, error) {
	flows := make([]SNetTapFlow, 0)
	q := srv.getFlowsQuery()
	err := db.FetchModelObjects(NetTapFlowManager, q, &flows)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return flows, nil
}

func (srv *SNetTapService) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	cnt, err := srv.getFlowsCount()
	if err != nil {
		return errors.Wrap(err, "getFlowCount")
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Tap service has associated flows")
	}
	return srv.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, info)
}

func (manager *SNetTapServiceManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.NetTapServiceCreateInput,
) (api.NetTapServiceCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(")
	}
	switch input.Type {
	case api.TapServiceHost:
		hostObj, err := HostManager.FetchByIdOrName(ctx, userCred, input.TargetId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", HostManager.Keyword(), input.TargetId)
			} else {
				return input, errors.Wrap(err, "HostManager.FetchByIdOrName")
			}
		}
		host := hostObj.(*SHost)
		if host.HostType != api.HOST_TYPE_HYPERVISOR {
			return input, errors.Wrapf(httperrors.ErrNotSupported, "host type %s not supported", host.HostType)
		}
		if len(input.MacAddr) > 0 {
			input.MacAddr = netutils.FormatMacAddr(input.MacAddr)
			nic := host.GetNetInterface(input.MacAddr, 1)
			if nic == nil {
				return input, errors.Wrap(errors.ErrNotFound, "host.GetNetInterface")
			}
			if len(nic.WireId) > 0 {
				return input, errors.Wrap(httperrors.ErrNotEmpty, "interface has been used")
			}
		}
		_, err = manager.fetchByHostIdMac(hostObj.GetId(), input.MacAddr)
		if err == nil {
			return input, errors.Wrapf(httperrors.ErrNotEmpty, "host %s(%s) has been attached to tap service", input.TargetId, input.MacAddr)
		} else if errors.Cause(err) != sql.ErrNoRows {
			return input, errors.Wrap(err, "fetchGuestById")
		}
		input.TargetId = hostObj.GetId()
	case api.TapServiceGuest:
		guestObj, err := GuestManager.FetchByIdOrName(ctx, userCred, input.TargetId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", GuestManager.Keyword(), input.TargetId)
			} else {
				return input, errors.Wrap(err, "GuestManager.FetchByIdOrName")
			}
		}
		guest := guestObj.(*SGuest)
		if guest.Hypervisor != api.HYPERVISOR_KVM {
			return input, errors.Wrapf(httperrors.ErrNotSupported, "hypervisor %s not supported", guest.Hypervisor)
		}
		// check the guest attach to tap
		_, err = manager.fetchByGuestId(guestObj.GetId())
		if err == nil {
			return input, errors.Wrapf(httperrors.ErrNotEmpty, "guest %s has been attached to tap service", input.TargetId)
		} else if errors.Cause(err) != sql.ErrNoRows {
			return input, errors.Wrap(err, "fetchGuestById")
		}
		input.TargetId = guestObj.GetId()
	default:
		return input, errors.Wrapf(httperrors.ErrNotSupported, "unsupported type %s", input.Type)
	}
	return input, nil
}

func (man *SNetTapServiceManager) fetchByGuestId(guestId string) (*SNetTapService, error) {
	return man.fetchByTargetMac(api.TapServiceGuest, guestId, "")
}

func (man *SNetTapServiceManager) fetchByHostIdMac(hostId, mac string) (*SNetTapService, error) {
	return man.fetchByTargetMac(api.TapServiceHost, hostId, mac)
}

func (man *SNetTapServiceManager) fetchByTargetMac(tapTyep, targetId, mac string) (*SNetTapService, error) {
	q := man.Query().Equals("type", api.TapServiceGuest).Equals("target_id", targetId)
	if len(mac) > 0 {
		q = q.Equals("mac_addr", mac)
	}
	tapObj, err := db.NewModelObject(man)
	if err != nil {
		return nil, errors.Wrap(err, "NewModelObject")
	}
	err = q.First(tapObj)
	if err != nil {
		return nil, errors.Wrap(err, "First")
	}
	return tapObj.(*SNetTapService), nil
}

func (tap *SNetTapService) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	input := api.NetTapServiceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return errors.Wrap(err, "Unmarshal NetTapServiceCreateInput")
	}
	switch input.Type {
	case api.TapServiceGuest:
		err := func() error {
			lockman.LockClass(ctx, NetTapServiceManager, "")
			defer lockman.ReleaseClass(ctx, NetTapServiceManager, "")
			// generate mac
			mac, err := NetTapServiceManager.GenerateMac(input.MacAddr)
			if err != nil {
				return errors.Wrap(err, "GenerateMac")
			}
			ifname, err := NetTapServiceManager.generateIfname(mac)
			if err != nil {
				return errors.Wrap(err, "generateIfname")
			}
			tap.Ifname = ifname
			tap.MacAddr = mac
			return nil
		}()
		if err != nil {
			return errors.Wrap(err, "generate mac and ifname")
		}
	}
	return tap.SEnabledStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (manager *SNetTapServiceManager) generateIfname(seed string) (string, error) {
	for tried := 0; tried < maxMacTries; tried++ {
		ifname := "tap" + seclib2.HashId(seed, byte(tried), 6)
		cnt, err := manager.Query().Equals("ifname", ifname).CountWithError()
		if err != nil {
			return "", errors.Wrap(err, "CountWithError")
		}
		if cnt == 0 {
			return ifname, nil
		}
	}
	return "", errors.Wrap(httperrors.ErrTooManyAttempts, "maximal retry reached")
}

func (manager *SNetTapServiceManager) GenerateMac(suggestion string) (string, error) {
	return generateMac(suggestion)
}

func (manager *SNetTapServiceManager) FilterByMac(mac string) *sqlchemy.SQuery {
	return manager.Query().Equals("mac_addr", mac)
}

func (srv *SNetTapService) getTapHostIp() string {
	var hostId string
	if srv.Type == api.TapServiceGuest {
		guest := GuestManager.FetchGuestById(srv.TargetId)
		hostId = guest.HostId
	} else {
		hostId = srv.TargetId
	}
	host := HostManager.FetchHostById(hostId)
	return host.AccessIp
}

func (srv *SNetTapService) getConfig() (api.STapServiceConfig, error) {
	conf := api.STapServiceConfig{}

	flows, err := NetTapFlowManager.getEnabledTapFlowsOfTap(srv.Id)
	if err != nil {
		return conf, errors.Wrap(err, "NetTapFlowManager.getEnabledTapFlows")
	}
	mirrors := make([]api.SMirrorConfig, 0)
	for _, flow := range flows {
		mc, err := flow.getMirrorConfig(false)
		if err != nil {
			log.Errorf("getMirrorConfig fail: %s", err)
		} else {
			mirrors = append(mirrors, mc)
		}
	}
	sort.Sort(sMirrorConfigs(mirrors))
	conf.Mirrors = mirrors // groupMirrorConfig(mirrors)

	conf.TapHostIp = srv.getTapHostIp()
	conf.MacAddr = srv.MacAddr
	conf.Ifname = srv.Ifname
	return conf, nil
}

type sMirrorConfigs []api.SMirrorConfig

func (a sMirrorConfigs) Len() int { return len(a) }

func (a sMirrorConfigs) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a sMirrorConfigs) Less(i, j int) bool {
	if a[i].TapHostIp != a[j].TapHostIp {
		return a[i].TapHostIp < a[j].TapHostIp
	}
	if a[i].HostIp != a[j].HostIp {
		return a[i].HostIp < a[j].HostIp
	}
	if a[i].Bridge != a[j].Bridge {
		return a[i].Bridge < a[j].Bridge
	}
	if a[i].Direction != a[j].Direction {
		return a[i].Direction < a[j].Direction
	}
	if a[i].VlanId != a[j].VlanId {
		return a[i].VlanId < a[j].VlanId
	}
	if a[i].Port != a[j].Port {
		return a[i].Port < a[j].Port
	}
	return a[i].FlowId < a[j].FlowId
}

/*
func groupMirrorConfig(mirrors []api.SMirrorConfig) []api.SHostBridgeMirrorConfig {
	sort.Sort(sMirrorConfigs(mirrors))
	ret := make([]api.SHostBridgeMirrorConfig, 0)
	var mc *api.SHostBridgeMirrorConfig
	for _, m := range mirrors {
		if mc != nil && (mc.TapHostIp != m.TapHostIp || mc.HostIp != m.HostIp || mc.Bridge != m.Bridge || mc.Direction != m.Direction) {
			ret = append(ret, *mc)
			mc = nil
		}
		if mc == nil {
			mc = &api.SHostBridgeMirrorConfig{
				TapHostIp: m.TapHostIp,
				HostIp:    m.HostIp,
				Bridge:    m.Bridge,
				Direction: m.Direction,
				FlowId:    m.FlowId,
			}
		}
		if m.VlanId > 0 {
			mc.VlanId = append(mc.VlanId, m.VlanId)
		}
		if len(m.Port) > 0 {
			mc.Port = append(mc.Port, m.Port)
		}
	}
	if mc != nil {
		ret = append(ret, *mc)
	}
	return ret
}
*/

func (manager *SNetTapServiceManager) getEnabledTapServiceByGuestId(guestId string) *SNetTapService {
	srvs, err := manager.getTapServicesByGuestId(guestId, true)
	if err != nil {
		log.Errorf("getTapServicesByGuestId fail %s", err)
		return nil
	}
	if len(srvs) == 0 {
		return nil
	}
	return &srvs[0]
}

func (manager *SNetTapServiceManager) getTapServicesByGuestId(guestId string, enabled bool) ([]SNetTapService, error) {
	return manager.getTapServices(api.TapServiceGuest, guestId, enabled)
}

func (manager *SNetTapServiceManager) getTapServicesByHostId(hostId string, enabled bool) ([]SNetTapService, error) {
	return manager.getTapServices(api.TapServiceHost, hostId, enabled)
}

func (manager *SNetTapServiceManager) getTapServices(typeStr string, guestId string, enabled bool) ([]SNetTapService, error) {
	q := manager.Query().Equals("type", typeStr).Equals("target_id", guestId)
	if enabled {
		q = q.IsTrue("enabled")
	}
	srvs := make([]SNetTapService, 0)
	err := db.FetchModelObjects(manager, q, &srvs)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return srvs, nil
}

func (srv *SNetTapService) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformEnableInput,
) (jsonutils.JSONObject, error) {
	ret, err := srv.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.PerformEnable")
	}
	if srv.Type == api.TapServiceGuest {
		// need to sync config
		guest := GuestManager.FetchGuestById(srv.TargetId)
		guest.StartSyncTask(ctx, userCred, false, "")
	}
	return ret, nil
}

func (srv *SNetTapService) PerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformDisableInput,
) (jsonutils.JSONObject, error) {
	ret, err := srv.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.PerformEnable")
	}
	if srv.Type == api.TapServiceGuest {
		// need to sync config
		guest := GuestManager.FetchGuestById(srv.TargetId)
		guest.StartSyncTask(ctx, userCred, false, "")
	}
	return ret, nil
}

func (srv *SNetTapService) cleanup(ctx context.Context, userCred mcclient.TokenCredential) error {
	flows, err := srv.getFlows()
	if err != nil {
		return errors.Wrap(err, "getFlows")
	}
	for _, flow := range flows {
		err := flow.Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "flow.Delete")
		}
	}
	err = srv.Delete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "srv.Delete")
	}
	return nil
}

func (manager *SNetTapServiceManager) removeTapServicesByGuestId(ctx context.Context, userCred mcclient.TokenCredential, targetId string) error {
	return manager.removeTapServices(ctx, userCred, api.TapServiceGuest, targetId)
}

func (manager *SNetTapServiceManager) removeTapServicesByHostId(ctx context.Context, userCred mcclient.TokenCredential, targetId string) error {
	return manager.removeTapServices(ctx, userCred, api.TapServiceHost, targetId)
}

func (manager *SNetTapServiceManager) removeTapServices(ctx context.Context, userCred mcclient.TokenCredential, srvType string, targetId string) error {
	srvs, err := manager.getTapServices(srvType, targetId, false)
	if err != nil {
		return errors.Wrap(err, "getTapServicesByHostId")
	}
	for i := range srvs {
		err := srvs[i].cleanup(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "cleanup")
		}
	}
	return nil
}
