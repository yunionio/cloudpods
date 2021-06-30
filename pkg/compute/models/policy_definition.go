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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SPolicyDefinitionManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var PolicyDefinitionManager *SPolicyDefinitionManager

func init() {
	PolicyDefinitionManager = &SPolicyDefinitionManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SPolicyDefinition{},
			"policy_definitions_tbl",
			"policy_definition",
			"policy_definitions",
		),
	}
	PolicyDefinitionManager.SetVirtualObject(PolicyDefinitionManager)
}

type SPolicyDefinition struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	// 参数
	Parameters *jsonutils.JSONDict `get:"domain" list:"domain" create:"admin_optional"`

	// 条件
	Condition string `width:"32" charset:"ascii" nullable:"false" get:"domain" list:"domain" create:"required"`
	// 类别
	Category string `width:"16" charset:"ascii" nullable:"false" get:"domain" list:"domain" create:"required"`
}

// 策略列表
func (manager *SPolicyDefinitionManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.PolicyDefinitionListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SPolicyDefinitionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.PolicyDefinitionCreateInput) (api.PolicyDefinitionCreateInput, error) {
	return input, httperrors.NewUnsupportOperationError("not support create definition")
}

func (manager *SPolicyDefinitionManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.PolicyDefinitionListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SPolicyDefinitionManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SPolicyDefinitionManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.PolicyDefinitionDetails {
	rows := make([]api.PolicyDefinitionDetails, len(objs))
	statusRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.PolicyDefinitionDetails{
			StatusStandaloneResourceDetails: statusRows[i],
		}
	}
	return rows
}

func (manager *SPolicyDefinitionManager) getPolicyDefinitionsByManagerId(providerId string) ([]SPolicyDefinition, error) {
	definitions := []SPolicyDefinition{}
	err := fetchByManagerId(manager, providerId, &definitions)
	if err != nil {
		return nil, errors.Wrap(err, "fetchByManagerId")
	}
	return definitions, nil
}

func (manager *SPolicyDefinitionManager) GetAvailablePolicyDefinitions(ctx context.Context, userCred mcclient.TokenCredential) ([]SPolicyDefinition, error) {
	q := manager.Query()
	sq := PolicyAssignmentManager.Query().SubQuery()
	q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("policydefinition_id"))).Filter(
		sqlchemy.Equals(sq.Field("domain_id"), userCred.GetDomainId()),
	).Equals("status", api.POLICY_DEFINITION_STATUS_READY)
	definitions := []SPolicyDefinition{}
	err := db.FetchModelObjects(manager, q, &definitions)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return definitions, nil
}

func (self *SPolicyDefinition) GetPolicyAssignments() ([]SPolicyAssignment, error) {
	assignments := []SPolicyAssignment{}
	q := PolicyAssignmentManager.Query().Equals("policydefinition_id", self.Id)
	err := db.FetchModelObjects(PolicyAssignmentManager, q, &assignments)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return assignments, nil
}
