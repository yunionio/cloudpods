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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/utils"
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

	UID         string    `width:"128" nullable:"false" create:"required" list:"user" update:"user"`
	ContactType string    `width:"16" nullable:"false" create:"required" list:"user" update:"user"`
	Contact     string    `width:"64" nullable:"false" create:"required" list:"user" update:"user"`
	Enabled     string    `width:"5" nullable:"false" default:"1" create:"optional" list:"user" update:"user"`
	VerifiedAt  time.Time `update:"user" list:"user"`
}

func (self *SContactManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SContactManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
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

func (self *SContactManager) FetchDingtalkContacts(uid string) {
	// todo
}

func (self *SContactManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	queryDict := query.(*jsonutils.JSONDict)
	if queryDict.Contains("uid") {
		uid, _ := queryDict.GetString("uid")
		q = q.Filter(sqlchemy.Equals(q.Field("uid"), uid))
	}
	return q, nil
}

func (self *SContactManager) GetAllNotify(id, contactType string, group bool) ([]SContact, error) {
	var uids []string
	var err error

	q := self.Query()
	if !group {
		q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("uid"), id), sqlchemy.Equals(q.Field("contact_type"), contactType), sqlchemy.Equals(q.Field("status"), CONTACT_VERIFIED)))
		uids = []string{id}
	} else {
		uids, err = utils.GetUsersByGroupID(id)
		if err != nil {
			return nil, err
		}
		q.Filter(sqlchemy.AND(sqlchemy.In(q.Field("uid"), uids), sqlchemy.Equals(q.Field("contact_type"), contactType)))
	}
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

func NewSContactResponse(uid string, details string) SContactResponse {
	name, _ := utils.GetUsernameByID(uid)
	return SContactResponse{
		Id:      uid,
		Name:    name,
		Details: details,
	}
}
