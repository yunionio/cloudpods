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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGuestTemplateResourceBase struct {
	// 主机模板ID
	GuestTemplateId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

type SGuestTemplateResourceBaseManager struct{}

func (self *SGuestTemplateResourceBase) GetGuestTemplate() *SGuestTemplate {
	obj, _ := GuestTemplateManager.FetchById(self.GuestTemplateId)
	if obj != nil {
		return obj.(*SGuestTemplate)
	}
	return nil
}

func (manager *SGuestTemplateResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GuestTemplateResourceInfo {
	rows := make([]api.GuestTemplateResourceInfo, len(objs))
	guestTemplateIds := make([]string, len(objs))
	for i := range objs {
		var base *SGuestTemplateResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SGuestTemplateResourceBase in object %s", objs[i])
			continue
		}
		guestTemplateIds[i] = base.GuestTemplateId
	}

	for i := range guestTemplateIds {
		rows[i].GuestTemplateId = guestTemplateIds[i]
	}
	guestTemplateNames, err := db.FetchIdNameMap2(GuestTemplateManager, guestTemplateIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}
	for i := range rows {
		if name, ok := guestTemplateNames[guestTemplateIds[i]]; ok {
			rows[i].GuestTemplate = name
		}
	}
	return rows
}

func (manager *SGuestTemplateResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestTemplateFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.GuestTemplateId) > 0 {
		guestTemplateObj, err := GuestTemplateManager.FetchByIdOrName(ctx, userCred, query.GuestTemplateId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GuestTemplateManager.Keyword(), query.GuestTemplateId)
			} else {
				return nil, errors.Wrap(err, "GuestTemplateManager.FetchByIdOrName")
			}
		}
		q = q.Equals("guest_template_id", guestTemplateObj.GetId())
	}
	return q, nil
}

func (manager *SGuestTemplateResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "guest_template":
		guestTemplateQuery := GuestTemplateManager.Query("name", "id").SubQuery()
		q = q.AppendField(guestTemplateQuery.Field("name", field)).Distinct()
		q = q.Join(guestTemplateQuery, sqlchemy.Equals(q.Field("guest_template_id"), guestTemplateQuery.Field("id")))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SGuestTemplateResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := GuestTemplateManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("guest_template_id"), subq.Field("id")))
		if keys.Contains("guest_template") {
			q = q.AppendField(subq.Field("name", "guest_template"))
		}
	}
	return q, nil
}

func (manager *SGuestTemplateResourceBaseManager) GetExportKeys() []string {
	return []string{"guest_template"}
}
