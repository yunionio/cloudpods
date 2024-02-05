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
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SPolicyDefinitionResourceBase struct {
	// 策略Id
	PolicydefinitionId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

type SPolicyDefinitionResourceBaseManager struct {
}

func (manager *SPolicyDefinitionResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.PolicyDefinitionResourceListInput) (*sqlchemy.SQuery, error) {
	if len(query.Policydefinition) > 0 {
		definition, err := PolicyDefinitionManager.FetchByIdOrName(ctx, userCred, query.Policydefinition)
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return nil, httperrors.NewGeneralError(err)
			}
			return nil, httperrors.NewResourceNotFoundError2("policy_definition", query.Policydefinition)
		}
		q = q.Equals("policydefinition_id", definition.GetId())
	}
	return q, nil
}

func (manager *SPolicyDefinitionResourceBaseManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool,
) []api.PolicyDefinitionResourceInfo {
	rows := make([]api.PolicyDefinitionResourceInfo, len(objs))
	definitionIds := make([]string, len(objs))
	for i := range objs {
		definitionIds[i] = objs[i].(*SPolicyAssignment).PolicydefinitionId
	}

	idMaps, err := db.FetchIdNameMap2(PolicyDefinitionManager, definitionIds)
	if err != nil {
		return rows
	}
	for i := range objs {
		rows[i].Policydefinition, _ = idMaps[definitionIds[i]]
	}
	return rows
}
