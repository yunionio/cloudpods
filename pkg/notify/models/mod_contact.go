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
	"fmt"
	"strings"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/utils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SContactManager struct {
	SStatusStandaloneResourceBaseManager
}

var ContactManager *SContactManager

func init() {
	ContactManager = &SContactManager{
		SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(
			SContact{},
			"notify_t_contacts",
			"contact",
			"contacts",
		),
	}
	ContactManager.SetVirtualObject(ContactManager)
}

type SContact struct {
	SStatusStandaloneResourceBase

	UID         string    `width:"128" nullable:"false" create:"required" update:"user"`
	ContactType string    `width:"16" nullable:"false" create:"required" update:"user"`
	Contact     string    `width:"64" nullable:"false" create:"required" update:"user"`
	Enabled     string    `width:"5" nullable:"false" default:"1" create:"optional" update:"user"`
	VerifiedAt  time.Time `update:"user"`
}

func (self *SContactManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SContactManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SContactManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (self *SContactManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (self *SContactManager) FetchOwnerId(ctx context.Context,
	data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {

	return db.FetchUserInfo(ctx, data)
}

func (self *SContactManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider,
	scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		if scope == rbacutils.ScopeUser {
			if len(owner.GetUserId()) > 0 {
				q = q.Equals("uid", owner.GetUserId())
			}
		}
	}
	return q
}

func (self *SContactManager) InitializeData() error {
	sql := fmt.Sprintf("update %s set updated_at=update_at, deleted=is_deleted", self.TableSpec().Name())
	q := sqlchemy.NewRawQuery(sql, "")
	q.Row()
	return nil
}

func (self *SContactManager) FetchByUIDs(uids []string) ([]SContact, error) {
	q := self.Query()
	q = q.Filter(sqlchemy.In(q.Field("uid"), uids))
	records := make([]SContact, 0, len(uids))
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SContactManager) FetchByUIDAndCType(uid string, contactTypes []string) ([]SContact, error) {
	q := self.Query("id", "uid", "contact_type", "contact", "enabled")
	q = q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("uid"), uid), sqlchemy.In(q.Field("contact_type"), contactTypes)))
	records := make([]SContact, 0, len(contactTypes))
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SContactManager) FetchByMore(uid, contact, contactType string) ([]SContact, error) {
	q := self.Query()
	q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("uid"), uid), sqlchemy.Equals(q.Field("contact"), contact), sqlchemy.Equals(q.Field("contact_type"), contactType)))
	records := make([]SContact, 0, 1)
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SContact) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) *jsonutils.JSONDict {

	ret, _ := self.getMoreDetail(ctx, userCred, query)
	return ret
}

func (self *SContact) getMoreDetail(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {

	ret := jsonutils.NewDict()
	uname, err := utils.GetUsernameByID(ctx, self.UID)
	if err != nil {
		return ret, err
	}

	q := ContactManager.Query().Equals("uid", self.UID)
	contacts := make([]SContact, 0)
	err = db.FetchModelObjects(ContactManager, q, &contacts)
	if err != nil {
		return ret, errors.Wrapf(err, "fetch Contacts of uid %s error", self.UID)
	}
	ret.Add(jsonutils.NewString(self.UID), "id")
	ret.Add(jsonutils.NewString(uname), "name")
	ret.Add(jsonutils.NewString(jsonutils.Marshal(contacts).String()), "details")

	return ret, nil
}

func (self *SContact) GetExtraDetail(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {

	return self.getMoreDetail(ctx, userCred, query)
}

func (self *SContactManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	queryDict := query.(*jsonutils.JSONDict)
	if queryDict.Contains("uid") {
		uid, _ := queryDict.GetString("uid")
		q = q.Equals("uid", uid)
	}
	// for now
	if queryDict.Contains("filter") {
		filterCon, _ := queryDict.GetString("filter")
		queryDict.Remove("filter")
		contain := "name.contains("
		index := strings.Index(filterCon, contain)
		if index < 0 {
			return q, nil
		}
		filterCon = filterCon[index+len(contain):]
		index = strings.Index(filterCon, ")")
		if index < 0 {
			return q, nil
		}
		name := filterCon[:index]
		ids, err := utils.GetUserIdsLikeName(ctx, name)
		if err != nil {
			return q, nil
		}
		q = q.In("uid", ids)
	}

	scopeStr, err := query.GetString("scope")
	if err != nil {
		scopeStr = "system"
	}
	scope := rbacutils.TRbacScope(scopeStr)

	if !scope.HigherEqual(rbacutils.ScopeSystem) {
		q = q.Equals("uid", userCred.GetUserId())
	}
	q = q.GroupBy("uid")

	return q, nil
}

func (self *SContactManager) GetAllNotify(ctx context.Context, ids []string, contactType string, group bool) ([]SContact, error) {
	var uids []string
	var err error

	q := self.Query()
	if !group {
		uids = ids
	} else {
		uid := make([]string, 0)
		for _, id := range uids {
			tmpUids, err := utils.GetUsersByGroupID(ctx, id)
			if err != nil {
				return nil, err
			}
			uid = append(uid, tmpUids...)
		}
	}
	q.Filter(sqlchemy.AND(sqlchemy.In(q.Field("uid"), uids), sqlchemy.Equals(q.Field("contact_type"),
		contactType), sqlchemy.Equals(q.Field("status"), CONTACT_VERIFIED)))

	if contactType == WEBCONSOLE {
		ret := make([]SContact, len(uids))
		for i := range uids {
			ret[i] = SContact{
				UID:         uids[i],
				ContactType: WEBCONSOLE,
				Contact:     uids[i],
			}
		}
		return ret, nil
	}
	contacts := make([]SContact, 0, 2)
	err = db.FetchModelObjects(self, q, &contacts)
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

type SContactResponse struct {
	Id      string
	Name    string
	Details string
}

func NewSContactResponse(ctx context.Context, uid string, details string) SContactResponse {
	name, _ := utils.GetUsernameByID(ctx, uid)
	return SContactResponse{
		Id:      uid,
		Name:    name,
		Details: details,
	}
}
