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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SCloudaccountResourceBaseManager struct {
}

type SCloudaccountResourceBase struct {
	// 云账号Id
	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" json:"cloudaccount_id"`
}

func (manager *SCloudaccountResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudaccountResourceListInput) (*sqlchemy.SQuery, error) {
	if len(query.CloudaccountId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, CloudaccountManager, &query.CloudaccountId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("cloudaccount_id", query.CloudaccountId)
	}
	if len(query.Provider) > 0 {
		sq := CloudaccountManager.Query().SubQuery()
		q = q.Join(sq, sqlchemy.Equals(q.Field("cloudaccount_id"), sq.Field("id"))).Filter(sqlchemy.In(sq.Field("provider"), query.Provider))
	}
	return q, nil
}

func (manager *SCloudaccountResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudaccountResourceDetails {
	rows := make([]api.CloudaccountResourceDetails, len(objs))
	accountIds := make([]string, len(objs))
	for i := range objs {
		var base *SCloudaccountResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudaccountResourceBase in %#v: %s", objs[i], err)
		} else if base != nil && len(base.CloudaccountId) > 0 {
			accountIds[i] = base.CloudaccountId
		}
	}
	accounts := make(map[string]SCloudaccount)
	err := db.FetchStandaloneObjectsByIds(CloudaccountManager, accountIds, &accounts)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %v", err)
		return rows
	}
	for i := range rows {
		if account, ok := accounts[accountIds[i]]; ok {
			rows[i].Cloudaccount = account.Name
			rows[i].Provider = account.Provider
			rows[i].Brand = account.Brand
			if len(rows[i].Brand) == 0 {
				rows[i].Brand = account.Provider
			}
			rows[i].IamLoginUrl = account.IamLoginUrl
		}
	}
	return rows
}

func (self *SCloudaccountResourceBase) GetCloudaccount() (*SCloudaccount, error) {
	account, err := CloudaccountManager.FetchById(self.CloudaccountId)
	if err != nil {
		return nil, errors.Wrap(err, "CloudaccountManager.FetchById")
	}
	return account.(*SCloudaccount), nil
}
