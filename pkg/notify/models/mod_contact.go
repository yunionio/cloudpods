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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

	UID         string    `width:"128" nullable:"false" create:"required" update:"user" list:"user" get:"user"`
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
	q := self.Query()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNotNull(q.Field("updated_at")), sqlchemy.IsTrue(q.Field("deleted"))))
	n, err := q.CountWithError()
	if err != nil {
		return err
	}
	if n > 0 {
		log.Debugf("no need to init data for %s", self.TableSpec().Name())
		// no need to init data
		return nil
	}
	log.Debugf("need to init data for %s", self.TableSpec().Name())
	sql := fmt.Sprintf("update %s set updated_at=update_at, deleted=is_deleted", self.TableSpec().Name())
	q = sqlchemy.NewRawQuery(sql, "")
	q.Row()
	return nil
}

// FetchByUIDs fetch all SContancts whose uid included in uids.
// If some elements of uids are uname of users, setting param 'uname' as true will fetch correct results.
func (self *SContactManager) FetchByUIDs(ctx context.Context, uids []string, uname bool) ([]SContact, error) {
	var err error
	if uname {
		uids, err = self._UIDsFromUIDOrName(ctx, uids)
		if err != nil {
			return nil, err
		}
	}
	q := self.Query()
	q = q.Filter(sqlchemy.In(q.Field("uid"), uids))
	records := make([]SContact, 0, len(uids))
	err = db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SContactManager) _UIDsFromUIDOrName(ctx context.Context, uidStrs []string) ([]string, error) {
	users, err := utils.GetUsersWithoutRemote(ctx, uidStrs)
	if err != nil {
		return nil, err
	}
	uids := make([]string, 0, len(uidStrs))
	uidSet := sets.NewString(uidStrs...)
	var (
		uid   string
		uname string
	)
	for i := range users {
		uid = users[i].Id
		uname = users[i].Name
		if uidSet.Has(uid) {
			uids = append(uids, uid)
			uidSet.Delete(uid)
			continue
		}
		if uidSet.Has(uname) {
			uids = append(uids, uid)
			uidSet.Delete(uname)
			continue
		}
	}
	for _, uid = range uidSet.UnsortedList() {
		uids = append(uids, uid)
	}
	log.Debugf("uids %s => %s", uidStrs, uids)
	return uids, nil
}

func (self *SContactManager) FetchByUIDAndCType(uid string, contactTypes []string) ([]SContact, error) {
	q := self.Query("id", "uid", "contact_type", "contact", "enabled").Equals("uid", uid).In("contact_type", contactTypes)
	records := make([]SContact, 0, len(contactTypes))
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SContactManager) FetchByMore(uid, contact, contactType string) ([]SContact, error) {
	q := self.Query().Equals("uid", uid).Equals("contact", contact).Equals("contact_type", contactType)
	records := make([]SContact, 0, 1)
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SContact) getMoreDetail(ctx context.Context, userCred mcclient.TokenCredential,
	out api.ContactDetails) (api.ContactDetails, error) {

	uname, err := utils.GetUsernameByID(ctx, self.UID)
	if errors.Cause(err) == sql.ErrNoRows {
		uname = self.UID
		err = nil
	}
	if err != nil {
		return out, err
	}

	q := ContactManager.Query().Equals("uid", self.UID)
	contacts := make([]SContact, 0)
	err = db.FetchModelObjects(ContactManager, q, &contacts)
	if err != nil {
		return out, errors.Wrapf(err, "fetch Contacts of uid %s error", self.UID)
	}
	out.UID = self.UID
	out.Name = uname
	out.Details = jsonutils.Marshal(contacts).String()

	return out, nil
}

func (self *SContact) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, isList bool) (api.ContactDetails, error) {
	var err error
	out := api.ContactDetails{}
	out.ModelBaseDetails, err = self.SStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query, isList)
	if err != nil {
		return out, err
	}
	return self.getMoreDetail(ctx, userCred, out)
}

// 联系方式列表
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
		scopeStr = "project"
	}
	scope := rbacutils.TRbacScope(scopeStr)

	switch {
	case scope.HigherEqual(rbacutils.ScopeSystem):
	case scope.HigherEqual(rbacutils.ScopeDomain):
		uids, err := self.uidsInDomain(ctx, userCred)
		if err != nil {
			return q, err
		}
		q = q.In("uid", uids)
	default:
		q = q.Equals("uid", userCred.GetUserId())
	}

	return q, nil
}

func (self *SContactManager) uidsInDomain(ctx context.Context, userCred mcclient.TokenCredential) ([]string, error) {
	session := auth.GetSession(ctx, userCred, "", "")
	params := jsonutils.NewDict()
	params.Set("scope", jsonutils.NewString("domain"))
	ret, err := modules.UsersV3.List(session, params)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Userv3.List")
	}
	uids := make([]string, 0, len(ret.Data))
	for i := range ret.Data {
		id, _ := ret.Data[i].GetString("id")
		uids = append(uids, id)
	}
	return uids, nil
}

// Contacts query all contacts by uids and contactType
func (self *SContactManager) Contacts(uids []string, contactType string) ([]SContact, error) {
	contacts := make([]SContact, 0, len(uids))
	if contactType == WEBCONSOLE {
		for _, uid := range uids {
			contacts = append(contacts, SContact{
				UID:         uid,
				ContactType: WEBCONSOLE,
				Contact:     uid,
			})
		}
		return contacts, nil
	}

	queryCon := contactType
	if strings.Contains(contactType, ROBOT) {
		queryCon = MOBILE
	}
	q := self.Query().Equals("contact_type", queryCon).Equals("enabled", "1").In("uid", uids)
	err := db.FetchModelObjects(self, q, &contacts)
	if err != nil {
		return nil, err
	}

	// For Robot Sender, only one message of the same content is sent for multiple users,
	// so the user's contact information is a collection of all contact information
	if strings.Contains(contactType, ROBOT) {
		// hack
		contactVals := make([]string, len(contacts))
		uidVals := make([]string, len(contacts))
		for i := range contacts {
			contactVals[i] = contacts[i].Contact
			uidVals[i] = contacts[i].UID
		}
		contacts = []SContact{
			{
				UID:         strings.Join(uidVals, ","),
				ContactType: contactType,
				Contact:     strings.Join(contactVals, ","),
			},
		}
	}
	return contacts, nil
}

func (self *SContactManager) GetAllNotify(ctx context.Context, ids []string, contactType string, group bool) ([]SContact, error) {
	var uids []string
	var err error

	if !group {
		if v := ctx.Value("uname"); v != nil {
			ids, err = self._UIDsFromUIDOrName(ctx, ids)
			if err != nil {
				return nil, errors.Wrap(err, "fail to transfer array of UID or Uname to UIDs")
			}
		}
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
		uids = uid
	}
	return self.Contacts(uids, contactType)
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
