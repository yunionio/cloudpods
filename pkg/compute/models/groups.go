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

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	REDIS_TYPE = "REDIS"
	RDS_TYPE   = "RDS"
)

// +onecloud:swagger-gen-model-singular=instancegroup
// +onecloud:swagger-gen-model-plural=instancegroups
type SGroupManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
	SZoneResourceBaseManager
}

var GroupManager *SGroupManager

func init() {
	// GroupManager's Keyword and KeywordPlural is instancegroup and instancegroups because group has been used by
	// keystone.
	GroupManager = &SGroupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SGroup{},
			"groups_tbl",
			"instancegroup",
			"instancegroups",
		),
	}
	GroupManager.SetVirtualObject(GroupManager)
}

type SGroup struct {
	db.SVirtualResourceBase

	SZoneResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`

	db.SEnabledResourceBase `nullable:"false" default:"true" create:"optional" list:"user" update:"user"`

	// 服务类型
	ServiceType string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	ParentId    string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`

	// 可用区Id
	// example: zone1
	// ZoneId string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`

	// 调度策略
	SchedStrategy string `width:"16" charset:"ascii" nullable:"true" default:"" list:"user" update:"user" create:"optional"`

	// the upper limit number of guests with this group in a host
	Granularity     int               `nullable:"false" list:"user" get:"user" create:"optional" update:"user" default:"1"`
	ForceDispersion tristate.TriState `list:"user" get:"user" create:"optional" update:"user" default:"true"`
	// 是否启用
	// Enabled tristate.TriState `default:"true" create:"optional" list:"user" update:"user"`
}

// 主机组列表
func (sm *SGroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.InstanceGroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = sm.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = sm.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}

	q, err = sm.SZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}

	guestFilter := input.ServerId
	if len(guestFilter) != 0 {
		guestObj, err := GuestManager.FetchByIdOrName(ctx, userCred, guestFilter)
		if err != nil {
			return nil, err
		}
		ggSub := GroupguestManager.Query("group_id").Equals("guest_id", guestObj.GetId()).SubQuery()
		q = q.Join(ggSub, sqlchemy.Equals(ggSub.Field("group_id"), q.Field("id")))
	}
	if len(input.ParentId) > 0 {
		q = q.Equals("parent_id", input.ParentId)
	}
	if len(input.ServiceType) > 0 {
		q = q.Equals("service_type", input.ServiceType)
	}
	if len(input.SchedStrategy) > 0 {
		q = q.Equals("sched_strategy", input.SchedStrategy)
	}

	return q, nil
}

func (sm *SGroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.InstanceGroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = sm.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = sm.SZoneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.OrderByExtraFields")
	}
	if db.NeedOrderQuery([]string{input.OrderByVips}) {
		gnQ := GroupnetworkManager.Query()
		gnQ = gnQ.AppendField(gnQ.Field("group_id"), sqlchemy.COUNT("vips", gnQ.Field("ip_addr")))
		gnQ = gnQ.GroupBy("group_id")
		gnSQ := gnQ.SubQuery()
		q = q.LeftJoin(gnSQ, sqlchemy.Equals(gnSQ.Field("group_id"), q.Field("id")))
		q.AppendField(q.QueryFields()...)
		q.AppendField(gnSQ.Field("vips"))
		q = db.OrderByFields(q, []string{input.OrderByVips}, []sqlchemy.IQueryField{q.Field("vips")})
	}
	if db.NeedOrderQuery([]string{input.OrderByGuestCount}) {
		ggQ := GroupguestManager.Query()
		ggQ = ggQ.AppendField(ggQ.Field("group_id"), sqlchemy.COUNT("guest_count"))
		ggQ = ggQ.GroupBy("group_id")
		ggSQ := ggQ.SubQuery()
		q = q.LeftJoin(ggSQ, sqlchemy.Equals(ggSQ.Field("group_id"), q.Field("id")))
		q.AppendField(q.QueryFields()...)
		q.AppendField(ggSQ.Field("guest_count"))
		q = db.OrderByFields(q, []string{input.OrderByGuestCount}, []sqlchemy.IQueryField{q.Field("guest_count")})
	}
	return q, nil
}

func (sm *SGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = sm.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = sm.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (sm *SGroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.InstanceGroupDetail {
	rows := make([]api.InstanceGroupDetail, len(objs))

	virtRows := sm.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := sm.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.InstanceGroupDetail{
			VirtualResourceDetails: virtRows[i],
			ZoneResourceInfo:       zoneRows[i],
		}
		rows[i].GuestCount = objs[i].(*SGroup).GetGuestCount()
		rows[i].Vips, _ = GroupnetworkManager.getVips(objs[i].(*SGroup).Id)
		net, _ := objs[i].(*SGroup).getAttachedNetwork()
		if net != nil {
			rows[i].NetworkId = net.Id
			rows[i].Network = net.Name
		}
		eip, _ := objs[i].(*SGroup).getElasticIp()
		if eip != nil {
			rows[i].VipEip = eip.IpAddr
		}
	}

	return rows
}

func (group *SGroup) GetGuestCount() int {
	q := GroupguestManager.Query().Equals("group_id", group.Id)
	count, _ := q.CountWithError()
	return count
}

func (group *SGroup) GetGuests() []SGuest {
	ggm := GroupguestManager.Query().SubQuery()
	q := GuestManager.Query()
	q = q.Join(ggm, sqlchemy.Equals(q.Field("id"), ggm.Field("guest_id")))
	q = q.Filter(sqlchemy.Equals(ggm.Field("group_id"), group.Id))

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil
	}
	return guests
}

func (group *SGroup) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	eip, err := group.getElasticIp()
	if err != nil {
		return errors.Wrap(err, "getElasticIp")
	}
	if eip != nil {
		return errors.Wrapf(httperrors.ErrNotEmpty, "group associate with eip %s", eip.IpAddr)
	}
	q := GroupguestManager.Query().Equals("group_id", group.Id)
	count, err := q.CountWithError()
	if err != nil {
		return errors.Wrapf(err, "fail to check that if there are any guest in this group %s", group.Name)
	}
	if count > 0 {
		return httperrors.NewUnsupportOperationError("please retry after unbind all guests in group")
	}
	return nil
}

func (group *SGroup) GetNetworks() ([]SGroupnetwork, error) {
	q := GroupnetworkManager.Query().Equals("group_id", group.Id)
	groupnets := make([]SGroupnetwork, 0)
	err := db.FetchModelObjects(GroupnetworkManager, q, &groupnets)
	if err != nil {
		return nil, err
	}
	return groupnets, nil
}

func (group *SGroup) PerformBindGuests(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	if group.Enabled.IsFalse() {
		return nil, httperrors.NewForbiddenError("can not bind guest from disabled guest")
	}
	guestIdSet, hostIds, err := group.checkGuests(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	groupGuests, err := GroupguestManager.FetchByGroupId(group.Id)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, group, logclient.ACT_VM_ASSOCIATE, nil, userCred, false)
		return nil, err
	}

	for i := range groupGuests {
		guestId := groupGuests[i].GuestId
		if guestIdSet.Has(guestId) {
			guestIdSet.Delete(guestId)
		}
	}

	var networkId string
	gns, err := group.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "GetNetworks")
	}
	if len(gns) > 0 {
		networkId = gns[0].NetworkId
	}

	for _, guestId := range guestIdSet.UnsortedList() {
		if len(networkId) > 0 {
			// need to check consistency of network
			gns, err := GuestnetworkManager.FetchByGuestId(guestId)
			if err != nil {
				return nil, errors.Wrap(err, "")
			}
			if len(gns) != 1 {
				return nil, errors.Wrap(httperrors.ErrNotSupported, "cannot join a guest without network or with more than one network to a group with VIP")
			}
			if gns[0].NetworkId != networkId {
				return nil, errors.Wrap(httperrors.ErrConflict, "cannot join a guest with network inconsist with VIP")
			}
		}
		_, err := GroupguestManager.Attach(ctx, group.Id, guestId)
		if err != nil {
			logclient.AddActionLogWithContext(ctx, group, logclient.ACT_VM_ASSOCIATE, nil, userCred, false)
			return nil, errors.Wrapf(err, "fail to attch guest %s to group %s", guestId, group.Id)
		}
	}

	err = group.clearSchedDescCache(hostIds)
	if err != nil {
		log.Errorf("fail to clear scheduler desc cache after binding guests successfully: %s", err.Error())
	}
	logclient.AddActionLogWithContext(ctx, group, logclient.ACT_VM_ASSOCIATE, nil, userCred, true)
	return nil, nil
}

func (group *SGroup) PerformUnbindGuests(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	if group.Enabled.IsFalse() {
		return nil, httperrors.NewForbiddenError("can not unbind guest from disabled guest")
	}
	guestIdSet, hostIds, err := group.checkGuests(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}

	groupGuests, err := GroupguestManager.FetchByGroupId(group.Id)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, group, logclient.ACT_VM_DISSOCIATE, nil, userCred, false)
		return nil, err
	}

	for i := range groupGuests {
		joint := groupGuests[i]
		if !guestIdSet.Has(joint.GuestId) {
			continue
		}
		err := joint.Detach(ctx, userCred)
		if err != nil {
			logclient.AddActionLogWithContext(ctx, group, logclient.ACT_VM_DISSOCIATE, nil, userCred, false)
			return nil, errors.Wrapf(err, "fail to detach guest %s to group %s", joint.GuestId, group.Id)
		}
	}

	err = group.clearSchedDescCache(hostIds)
	if err != nil {
		log.Errorf("fail to clear scheduler desc cache after unbinding guests successfully: %s", err.Error())
	}
	logclient.AddActionLogWithContext(ctx, group, logclient.ACT_VM_DISSOCIATE, nil, userCred, true)
	return nil, nil
}

func (group *SGroup) checkGuests(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (guestIdSet sets.String, hostIds []string, err error) {

	guestIdArr := jsonutils.GetArrayOfPrefix(data, "guest")
	if len(guestIdArr) == 0 {
		return nil, nil, httperrors.NewMissingParameterError("guest.0 guest.1 ... ")
	}

	guestIdSet = sets.NewString()
	hostIdSet := sets.NewString()
	for i := range guestIdArr {
		guestIdStr, _ := guestIdArr[i].GetString()
		model, err := GuestManager.FetchByIdOrName(ctx, userCred, guestIdStr)
		if err == sql.ErrNoRows {
			return nil, nil, httperrors.NewInputParameterError("no such model %s", guestIdStr)
		}
		if err != nil {
			return nil, nil, errors.Wrapf(err, "fail to fetch model by id or name %s", guestIdStr)
		}
		guest := model.(*SGuest)
		if guest.ProjectId != group.ProjectId {
			return nil, nil, httperrors.NewForbiddenError("guest and instance group should belong to same project")
		}
		guestIdSet.Insert(guest.GetId())
		hostIdSet.Insert(guest.HostId)
	}
	hostIds = hostIdSet.List()
	return
}

func (group *SGroup) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(group, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (group *SGroup) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(group, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (group *SGroup) ClearAllScheDescCache() error {
	guests, err := group.fetchAllGuests()
	if err != nil {
		return errors.Wrapf(err, "fail to fetch all guest of group %s", group.Id)
	}

	hostIdSet := sets.NewString()
	for i := range guests {
		hostIdSet.Insert(guests[i].HostId)
	}

	return group.clearSchedDescCache(hostIdSet.List())
}

func (group *SGroup) clearSchedDescCache(hostIds []string) error {
	var g errgroup.Group
	for i := range hostIds {
		hostId := hostIds[i]
		g.Go(func() error {
			return HostManager.ClearSchedDescCache(hostId)
		})
	}
	return g.Wait()
}

func (group *SGroup) fetchAllGuests() ([]SGuest, error) {
	ggSub := GroupguestManager.Query("guest_id").Equals("group_id", group.GetId()).SubQuery()
	guestSub := GuestManager.Query().SubQuery()
	q := guestSub.Query().Join(ggSub, sqlchemy.Equals(ggSub.Field("guest_id"), guestSub.Field("id")))
	guests := make([]SGuest, 0, 2)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, err
	}
	return guests, nil
}

func (manager *SGroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (grp *SGroup) getAttachedNetwork() (*SNetwork, error) {
	var netId string
	guests := grp.GetGuests()
	for i := range guests {
		guest := guests[i]
		nets, err := guest.GetNetworks("")
		if err != nil {
			return nil, errors.Wrapf(err, "guest.GestNetworks(%s)", guest.Name)
		}
		if len(nets) != 1 {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "guest (%s) has %d networks", guest.Name, len(nets))
		}
		if len(netId) == 0 {
			netId = nets[0].NetworkId
		} else if netId != nets[0].NetworkId {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "inconsistent networkId for member servers")
		}
	}
	if len(netId) == 0 {
		gns, err := GroupnetworkManager.FetchByGroupId(grp.Id)
		if err != nil {
			return nil, errors.Wrap(err, "GroupnetworkManager.FetchByGroupId")
		}
		for _, gn := range gns {
			netId = gn.NetworkId
		}
	}
	if len(netId) == 0 {
		return nil, nil
	}
	netObj, err := NetworkManager.FetchById(netId)
	if err != nil {
		return nil, errors.Wrapf(err, "NetworkManager.FetchById %s", netId)
	}
	return netObj.(*SNetwork), nil
}

func (net *SNetwork) GetRegionalQuotaKeys(ownerId mcclient.IIdentityProvider) (quotas.IQuotaKeys, error) {
	vpc, err := net.GetVpc()
	if err != nil {
		return nil, errors.Wrap(err, "getVpc")
	}
	provider := vpc.GetCloudprovider()
	if provider == nil && len(vpc.ManagerId) > 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid manager")
	}
	region, _ := net.GetRegion()
	if region == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid region")
	}
	return fetchRegionalQuotaKeys(rbacscope.ScopeProject, ownerId, region, provider), nil
}

func (grp *SGroup) PerformDetachnetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.GroupDetachNetworkInput) (*api.SGroup, error) {
	net, err := grp.getAttachedNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "getAttachedNetwork")
	}
	if net == nil {
		// no guest and no attached network
		return nil, nil
	}
	gns, err := GroupnetworkManager.FetchByGroupId(grp.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GroupnetworkManager.FetchByGroupId")
	}
	if len(gns) == 0 {
		return nil, nil
	}
	for _, gn := range gns {
		if len(input.IpAddr) == 0 || gn.IpAddr == input.IpAddr || gn.Ip6Addr == input.IpAddr {
			if len(gn.EipId) > 0 {
				logclient.AddSimpleActionLog(grp, logclient.ACT_DETACH_NETWORK, "eip associated", userCred, false)
				return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot detach network with eip")
			}
			// delete
			notes := gn.GetShortDesc(ctx)
			err := gn.Detach(ctx, userCred)
			if err != nil {
				logclient.AddSimpleActionLog(grp, logclient.ACT_DETACH_NETWORK, notes, userCred, false)
				return nil, errors.Wrap(err, "Detach")
			}
			logclient.AddSimpleActionLog(grp, logclient.ACT_DETACH_NETWORK, notes, userCred, true)
		}
	}
	return nil, nil
}

func (grp *SGroup) PerformAttachnetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.GroupAttachNetworkInput) (*api.SGroup, error) {
	net, err := grp.getAttachedNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "getAttachedNetwork")
	}

	if len(input.NetworkId) > 0 {
		netObj, err := NetworkManager.FetchByIdOrName(ctx, userCred, input.NetworkId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(NetworkManager.Keyword(), input.NetworkId)
			} else {
				return nil, errors.Wrap(err, "NetworkManager.FetchByIdOrName")
			}
		}
		if net != nil {
			if net.Id != netObj.GetId() {
				return nil, errors.Wrap(httperrors.ErrConflict, "provided network inconsistent with existing network")
			}
		} else {
			net = netObj.(*SNetwork)
		}
	} else if net == nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "please specify network")
	}

	if len(input.IpAddr) > 0 {
		addr, err := netutils.NewIPV4Addr(input.IpAddr)
		if err != nil {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid ip_addr %s", input.IpAddr)
		}
		if !net.GetIPRange().Contains(addr) {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "ip_addr %s not in range", input.IpAddr)
		}
	}
	if (len(input.Ip6Addr) > 0 || input.RequireIPv6) && !net.IsSupportIPv6() {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "network is not ipv6 enabled")
	}

	if len(input.Ip6Addr) > 0 {
		addr6, err := netutils.NewIPV6Addr(input.Ip6Addr)
		if err != nil {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid ip6_addr %s", input.Ip6Addr)
		}
		if !net.getIPRange6().Contains(addr6) {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "ip6_addr %s not in range", input.Ip6Addr)
		}
		input.Ip6Addr = addr6.String()
	}

	// check quota
	var inicCnt, enicCnt int
	var saveQuota bool
	if net.IsExitNetwork() {
		enicCnt = 1
	} else {
		inicCnt = 1
	}
	pendingUsage := &SRegionQuota{
		Port:  inicCnt,
		Eport: enicCnt,
	}
	keys, err := net.GetRegionalQuotaKeys(grp.GetOwnerId())
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionalQuotaKeys")
	}
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("%v", err)
	}
	defer quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, saveQuota)

	lockman.LockObject(ctx, net)
	defer lockman.ReleaseObject(ctx, net)

	ipAddr, err := net.GetFreeIP(ctx, userCred, nil, nil, input.IpAddr, input.AllocDir, input.Reserved != nil && *input.Reserved, api.AddressTypeIPv4)
	if err != nil {
		return nil, errors.Wrap(err, "GetFreeIPv4")
	}
	if len(input.IpAddr) > 0 && ipAddr != input.IpAddr && input.RequireDesignatedIp != nil && *input.RequireDesignatedIp {
		return nil, errors.Wrapf(httperrors.ErrConflict, "candidate ip %s is occupied!", input.IpAddr)
	}

	var ipAddr6 string
	if len(input.Ip6Addr) > 0 || input.RequireIPv6 {
		ipAddr6, err = net.GetFreeIP(ctx, userCred, nil, nil, input.Ip6Addr, input.AllocDir, input.Reserved != nil && *input.Reserved, api.AddressTypeIPv6)
		if err != nil {
			return nil, errors.Wrap(err, "GetFreeIPv6")
		}
		if len(input.Ip6Addr) > 0 && ipAddr6 != input.Ip6Addr && input.RequireDesignatedIp != nil && *input.RequireDesignatedIp {
			return nil, errors.Wrapf(httperrors.ErrConflict, "candidate v6 ip %s is occupied!", input.Ip6Addr)
		}
	}

	gn := SGroupnetwork{}
	gn.NetworkId = net.Id
	gn.GroupId = grp.Id
	gn.IpAddr = ipAddr
	gn.Ip6Addr = ipAddr6

	gn.SetModelManager(GroupnetworkManager, &gn)

	err = GroupnetworkManager.TableSpec().Insert(ctx, &gn)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	notes := gn.GetShortDesc(ctx)
	db.OpsLog.LogAttachEvent(ctx, grp, net, userCred, notes)
	logclient.AddActionLogWithContext(ctx, grp, logclient.ACT_ATTACH_NETWORK, notes, userCred, true)

	saveQuota = true

	guests := grp.GetGuests()
	for _, g := range guests {
		host, _ := g.GetHost()
		host.ClearSchedDescCache()
		g.StartSyncTask(ctx, userCred, false, "")
	}

	return nil, nil
}

func (grp *SGroup) GetVpc() (*SVpc, error) {
	net, err := grp.getAttachedNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "getAttachedNetwork")
	}
	return net.GetVpc()
}

func (grp *SGroup) isEipAssociable() (*SNetwork, error) {
	err := ValidateAssociateEip(grp)
	if err != nil {
		return nil, err
	}

	net, err := grp.getAttachedNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "getAttachedNetwork")
	}
	if net == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "group no attached network")
	}
	if !IsOneCloudVpcResource(net) {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "group network is not a VPC network")
	}

	gns, err := grp.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "GetNetworks")
	}
	if len(gns) == 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "group no vips")
	}

	return net, nil
}

func (grp *SGroup) PerformAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerAssociateEipInput) (jsonutils.JSONObject, error) {
	net, err := grp.isEipAssociable()
	if err != nil {
		return nil, errors.Wrap(err, "grp.isEipAssociable")
	}

	eipStr := input.EipId
	if len(eipStr) == 0 {
		return nil, httperrors.NewMissingParameterError("eip_id")
	}
	eipObj, err := ElasticipManager.FetchByIdOrName(ctx, userCred, eipStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("eip %s not found", eipStr)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	eip := eipObj.(*SElasticip)

	if eip.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot be associated")
	}

	if eip.IsAssociated() {
		return nil, httperrors.NewConflictError("eip has been associated")
	}

	if net.Id == eip.NetworkId {
		return nil, httperrors.NewInputParameterError("cannot associate eip with same network")
	}

	eipZone, _ := eip.GetZone()
	if eipZone != nil {
		insZone, _ := net.GetZone()
		if eipZone.Id != insZone.Id {
			return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different zone")
		}
	}

	grp.SetStatus(ctx, userCred, api.INSTANCE_ASSOCIATE_EIP, "associate eip")

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(grp.Id), "instance_id")
	params.Add(jsonutils.NewString(api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP), "instance_type")
	if len(input.IpAddr) > 0 {
		params.Add(jsonutils.NewString(input.IpAddr), "ip_addr")
	}

	err = eip.StartEipAssociateTask(ctx, userCred, params, "")

	return nil, err
}

func (grp *SGroup) PerformCreateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerCreateEipInput) (jsonutils.JSONObject, error) {
	net, err := grp.isEipAssociable()
	if err != nil {
		return nil, errors.Wrap(err, "grp.isEipAssociable")
	}

	var (
		region, _    = net.GetRegion()
		regionDriver = region.GetDriver()

		bw            = input.Bandwidth
		chargeType    = input.ChargeType
		bgpType       = input.BgpType
		autoDellocate = (input.AutoDellocate != nil && *input.AutoDellocate)
	)

	if chargeType == "" {
		chargeType = regionDriver.GetEipDefaultChargeType()
	}

	if chargeType == api.EIP_CHARGE_TYPE_BY_BANDWIDTH {
		if bw == 0 {
			return nil, httperrors.NewMissingParameterError("bandwidth")
		}
	}

	eipPendingUsage := &SRegionQuota{Eip: 1}
	keys, err := net.GetRegionalQuotaKeys(grp.GetOwnerId())
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	eipPendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, eipPendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("Out of eip quota: %s", err)
	}

	eip, err := ElasticipManager.NewEipForVMOnHost(ctx, userCred, &NewEipForVMOnHostArgs{
		Bandwidth:     int(bw),
		BgpType:       bgpType,
		ChargeType:    chargeType,
		AutoDellocate: autoDellocate,

		Group:        grp,
		PendingUsage: eipPendingUsage,
	})
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, eipPendingUsage, eipPendingUsage, false)
		return nil, httperrors.NewGeneralError(err)
	}

	opts := api.ElasticipAssociateInput{
		InstanceId:   grp.Id,
		InstanceType: api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP,
		IpAddr:       input.IpAddr,
	}

	err = eip.AllocateAndAssociateInstance(ctx, userCred, grp, opts, "")
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil
}

func (grp *SGroup) getElasticIp() (*SElasticip, error) {
	return ElasticipManager.getEip(api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP, grp.Id, api.EIP_MODE_STANDALONE_EIP)
}

func (grp *SGroup) PerformDissociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerDissociateEipInput) (jsonutils.JSONObject, error) {
	eip, err := grp.getElasticIp()
	if err != nil {
		log.Errorf("Fail to get Eip %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	if eip == nil {
		return nil, httperrors.NewInvalidStatusError("No eip to dissociate")
	}

	err = db.IsObjectRbacAllowed(ctx, eip, userCred, policy.PolicyActionGet)
	if err != nil {
		return nil, errors.Wrap(err, "eip is not accessible")
	}

	grp.SetStatus(ctx, userCred, api.INSTANCE_DISSOCIATE_EIP, "associate eip")

	autoDelete := (input.AudoDelete != nil && *input.AudoDelete)

	err = eip.StartEipDissociateTask(ctx, userCred, autoDelete, "")
	if err != nil {
		log.Errorf("fail to start dissociate task %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, nil
}

func (grp *SGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// cleanup groupnetwork
	grpnets, err := grp.GetNetworks()
	if err != nil {
		return errors.Wrap(err, "GetNetworks")
	}
	for i := range grpnets {
		err := grpnets[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "groupnetwork.Delete")
		}
	}
	return grp.SVirtualResourceBase.Delete(ctx, userCred)
}
