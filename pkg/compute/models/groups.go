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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

const (
	REDIS_TYPE = "REDIS"
	RDS_TYPE   = "RDS"
)

type SGroupManager struct {
	db.SVirtualResourceBaseManager
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

	ServiceType   string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`            // Column(VARCHAR(36, charset='ascii'), nullable=True)
	ParentId      string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`            // Column(VARCHAR(36, charset='ascii'), nullable=True)
	ZoneId        string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`            // Column(VARCHAR(36, charset='ascii'), nullable=True)
	SchedStrategy string `width:"16" charset:"ascii" nullable:"true" default:"" list:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True, default='')

	// the upper limit number of guests with this group in a host
	Granularity     int               `nullable:"false" list:"user" get:"user" create:"optional" update:"user" default:"1"`
	ForceDispersion tristate.TriState `list:"user" get:"user" create:"optional" update:"user" default:"true"`
	Enabled         tristate.TriState `nullable:"false" default:"true" create:"optional" list:"user" update:"user"`
}

func (sm *SGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {

	guestFilter := jsonutils.GetAnyString(query, []string{"guest", "guest_id"})
	if len(guestFilter) != 0 {
		guestObj, err := GuestManager.FetchByIdOrName(userCred, guestFilter)
		if err != nil {
			return nil, err
		}
		ggSub := GroupguestManager.Query("group_id").Equals("guest_id", guestObj.GetId()).SubQuery()
		q = q.Join(ggSub, sqlchemy.Equals(ggSub.Field("group_id"), q.Field("id")))
	}
	return q, nil
}

func (sp *SGroup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := sp.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	ret, _ := sp.getMoreDetails(ctx, userCred, extra)
	return ret
}

func (sp *SGroup) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := sp.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return sp.getMoreDetails(ctx, userCred, extra)
}

func (sp *SGroup) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	ret := query.(*jsonutils.JSONDict)
	ret.Add(jsonutils.JSONTrue, "enabled")
	q := GroupguestManager.Query().Equals("group_id", sp.Id)
	count, _ := q.CountWithError()
	ret.Add(jsonutils.NewInt(int64(count)), "guest_count")
	return ret, nil
}

func (s *SGroup) ValidateDeleteCondition(ctx context.Context) error {
	q := GroupguestManager.Query().Equals("group_id", s.Id)
	count, err := q.CountWithError()
	if err != nil {
		return errors.Wrapf(err, "fail to check that if there are any guest in this group %s", s.Name)
	}
	if count > 0 {
		return httperrors.NewUnsupportOperationError("请在解绑所有主机后重试")
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

	return group.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, group, "bind-guests") && group.Enabled.IsTrue()
}

func (group *SGroup) PerformBindGuests(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	guestIdSet, err := group.checkGuests(ctx, userCred, query, data)
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

	logclient.AddActionLogWithContext(ctx, group, logclient.ACT_VM_ASSOCIATE, nil, userCred, true)
	return nil, nil
}

func (group *SGroup) AllowPerformUnbindGuests(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {

	return group.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, group, "unbind-guests") && group.Enabled.IsTrue()
}

func (group *SGroup) PerformUnbindGuests(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guestIdSet, err := group.checkGuests(ctx, userCred, query, data)
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

	logclient.AddActionLogWithContext(ctx, group, logclient.ACT_VM_DISSOCIATE, nil, userCred, true)
	return nil, nil
}

func (group *SGroup) checkGuests(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (sets.String, error) {

	guestIdArr := jsonutils.GetArrayOfPrefix(data, "guest")
	if len(guestIdArr) == 0 {
		return nil, httperrors.NewMissingParameterError("guest.0 guest.1 ... ")
	}

	guestIdSet := sets.NewString()
	for i := range guestIdArr {
		guestIdStr, _ := guestIdArr[i].GetString()
		guest, err := GuestManager.FetchByIdOrName(userCred, guestIdStr)
		if err == sql.ErrNoRows {
			return nil, httperrors.NewInputParameterError("no such guest %s", guestIdStr)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "fail to fetch guest by id or name %s", guestIdStr)
		}
		guestIdSet.Insert(guest.GetId())
	}

	return guestIdSet, nil
}
