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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SPolicyAssignmentManager struct {
	db.SDomainLevelResourceBaseManager
	SPolicyDefinitionResourceBaseManager
}

var PolicyAssignmentManager *SPolicyAssignmentManager

func init() {
	PolicyAssignmentManager = &SPolicyAssignmentManager{
		SDomainLevelResourceBaseManager: db.NewDomainLevelResourceBaseManager(
			SPolicyAssignment{},
			"policy_assignments_tbl",
			"policy_assignment",
			"policy_assignments",
		),
	}
	PolicyAssignmentManager.SetVirtualObject(PolicyAssignmentManager)
}

type SPolicyAssignment struct {
	db.SDomainLevelResourceBase

	SPolicyDefinitionResourceBase
}

// 策略分配列表
func (manager *SPolicyAssignmentManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.PolicyAssignmentListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DomainLevelResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SPolicyDefinitionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.PolicyDefinitionResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SPolicyAssignmentManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.PolicyAssignmentDetails {
	rows := make([]api.PolicyAssignmentDetails, len(objs))
	domainRows := manager.SDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	definitionRows := manager.SPolicyDefinitionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.PolicyAssignmentDetails{
			DomainLevelResourceDetails:   domainRows[i],
			PolicyDefinitionResourceInfo: definitionRows[i],
		}
	}
	return rows
}

func (manager *SPolicyAssignmentManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.PolicyAssignmentListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SPolicyAssignmentManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.PolicyAssignmentCreateInput) (api.PolicyAssignmentCreateInput, error) {
	return input, httperrors.NewInputParameterError("not support create")
}

func (manager *SPolicyAssignmentManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SDomainLevelResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SPolicyAssignmentManager) checkAndSetAssignment(ctx context.Context, definition *SPolicyDefinition, domainId string) error {
	q := manager.Query().Equals("policydefinition_id", definition.Id).Equals("domain_id", domainId)
	count, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "CountWithError")
	}
	if count == 0 {
		return manager.newAssignment(ctx, definition, domainId)
	}
	return nil
}

func (manager *SPolicyAssignmentManager) newAssignment(ctx context.Context, definition *SPolicyDefinition, domainId string) error {
	assignment := SPolicyAssignment{}
	assignment.SetModelManager(manager, &assignment)

	assignment.Name = fmt.Sprintf("assignment for %s domain %s", definition.Name, domainId)
	assignment.DomainId = domainId
	assignment.PolicydefinitionId = definition.Id

	return manager.TableSpec().Insert(ctx, &assignment)
}
