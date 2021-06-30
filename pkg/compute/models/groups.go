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
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	REDIS_TYPE = "REDIS"
	RDS_TYPE   = "RDS"
)

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
	// Enabled tristate.TriState `nullable:"false" default:"true" create:"optional" list:"user" update:"user"`
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
		guestObj, err := GuestManager.FetchByIdOrName(userCred, guestFilter)
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
	}

	return rows
}

func (group *SGroup) GetGuestCount() int {
	q := GroupguestManager.Query().Equals("group_id", group.Id)
	count, _ := q.CountWithError()
	return count
}

func (group *SGroup) ValidateDeleteCondition(ctx context.Context) error {
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

func (group *SGroup) AllowPerformBindGuests(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {

	return group.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, group, "bind-guests")
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

	for _, guestId := range guestIdSet.UnsortedList() {
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

func (group *SGroup) AllowPerformUnbindGuests(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {

	return group.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, group, "unbind-guests")
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
		model, err := GuestManager.FetchByIdOrName(userCred, guestIdStr)
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

func (group *SGroup) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) bool {
	return group.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, group, "enable")
}

func (group *SGroup) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(group, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (group *SGroup) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) bool {
	return group.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, group, "disable")
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
	for _, hostId := range hostIds {
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
