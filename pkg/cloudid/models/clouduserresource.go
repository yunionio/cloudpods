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

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SClouduserResourceBaseManager struct {
}

// +onecloud:swagger-gen-ignore
type SClouduserResourceBase struct {
	ClouduserId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (self *SClouduserJointsBase) GetClouduser() (*SClouduser, error) {
	user, err := ClouduserManager.FetchById(self.ClouduserId)
	if err != nil {
		return nil, errors.Wrap(err, "FetchById")
	}
	return user.(*SClouduser), nil
}

func (manager *SClouduserResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ClouduserResourceListInput) (*sqlchemy.SQuery, error) {
	if len(query.Clouduser) > 0 {
		user, err := ClouduserManager.FetchByIdOrName(ctx, nil, query.Clouduser)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("clouduser", query.Clouduser)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("clouduser_id", user.GetId())
	}
	return q, nil
}

func (manager *SClouduserResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ClouduserResourceDetails {
	rows := make([]api.ClouduserResourceDetails, len(objs))
	userIds := make([]string, len(objs))
	for i := range objs {
		var base *SClouduserResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SClouduserResourceBase in %#v: %s", objs[i], err)
		} else if base != nil && len(base.ClouduserId) > 0 {
			userIds[i] = base.ClouduserId
		}
	}

	users := make(map[string]SClouduser)
	err := db.FetchStandaloneObjectsByIds(ClouduserManager, userIds, &users)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %v", err)
		return rows
	}
	accountIds := make([]string, len(objs))
	providerIds := make([]string, len(objs))
	for i := range rows {
		if user, ok := users[userIds[i]]; ok {
			rows[i].Clouduser = user.Name
			accountIds[i] = user.CloudaccountId
		}
	}
	accountMaps, err := db.FetchIdNameMap2(CloudaccountManager, accountIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %v", err)
		return rows
	}

	providerMaps, err := db.FetchIdNameMap2(CloudproviderManager, providerIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %v", err)
		return rows
	}

	for i := range rows {
		rows[i].Cloudaccount, _ = accountMaps[accountIds[i]]
		rows[i].Cloudprovider, _ = providerMaps[providerIds[i]]
	}
	return rows
}
