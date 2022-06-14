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
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

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
	return q, nil
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
		hostObj, err := HostManager.FetchByIdOrName(userCred, input.TargetId)
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
			nic := host.GetNetInterface(input.MacAddr)
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
		guestObj, err := GuestManager.FetchByIdOrName(userCred, input.TargetId)
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
