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

// +onecloud:swagger-gen-ignore
type SCloudpolicyResourceBaseManager struct {
}

type SCloudpolicyResourceBase struct {
	// 权限Id
	CloudpolicyId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" json:"cloudpolicy_id"`
}

func (manager *SCloudpolicyResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, policyCred mcclient.TokenCredential, query api.CloudpolicyResourceListInput) (*sqlchemy.SQuery, error) {
	if len(query.Cloudpolicy) > 0 {
		policy, err := CloudpolicyManager.FetchByIdOrName(ctx, nil, query.Cloudpolicy)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", query.Cloudpolicy)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("cloudpolicy_id", policy.GetId())
	}
	return q, nil
}

func (manager *SCloudpolicyResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	policyCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudpolicyResourceDetails {
	rows := make([]api.CloudpolicyResourceDetails, len(objs))
	policyIds := make([]string, len(objs))
	for i := range objs {
		var base *SCloudpolicyResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudpolicyResourceBase in %#v: %s", objs[i], err)
		} else if base != nil && len(base.CloudpolicyId) > 0 {
			policyIds[i] = base.CloudpolicyId
		}
	}
	policyMaps, err := db.FetchIdNameMap2(CloudpolicyManager, policyIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %v", err)
		return rows
	}
	for i := range rows {
		rows[i].Cloudpolicy, _ = policyMaps[policyIds[i]]
	}
	return rows
}
