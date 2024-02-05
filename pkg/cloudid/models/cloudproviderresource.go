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
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudproviderResourceBaseManager struct {
}

type SCloudproviderResourceBase struct {
	// 子订阅Id
	CloudproviderId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" json:"cloudprovider_id"`
}

func (manager *SCloudproviderResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudproviderResourceListInput) (*sqlchemy.SQuery, error) {
	if len(query.Cloudprovider) > 0 {
		provider, err := CloudproviderManager.FetchByIdOrName(ctx, nil, query.Cloudprovider)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudprovider", query.Cloudprovider)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("cloudprovider_id", provider.GetId())
	}
	return q, nil
}

func (manager *SCloudproviderResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudproviderResourceDetails {
	rows := make([]api.CloudproviderResourceDetails, len(objs))
	providerIds := make([]string, len(objs))
	for i := range objs {
		var base *SCloudproviderResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudproviderResourceBase in %#v: %s", objs[i], err)
		} else if base != nil && len(base.CloudproviderId) > 0 {
			providerIds[i] = base.CloudproviderId
		}
	}
	providerMaps, err := db.FetchIdNameMap2(CloudproviderManager, providerIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %v", err)
		return rows
	}
	for i := range rows {
		rows[i].Cloudprovider, _ = providerMaps[providerIds[i]]
	}
	return rows
}
