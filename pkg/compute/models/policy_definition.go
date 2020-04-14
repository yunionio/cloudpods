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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SPolicyDefinitionManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
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
	SManagedResourceBase

	// 参数
	Parameters jsonutils.JSONObject `nullable:"true" get:"domain" list:"domain" create:"admin_optional"`

	// 条件
	Condition string `width:"32" charset:"ascii" nullable:"false" get:"domain" list:"domain" create:"required"`
	// 类别
	Category string `width:"16" charset:"ascii" nullable:"false" get:"domain" list:"domain" create:"required"`
}

func (self *SPolicyDefinitionManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

// 策略列表
func (manager *SPolicyDefinitionManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.PolicyDefinitionListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SPolicyDefinitionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.PolicyDefinitionCreateInput) (api.PolicyDefinitionCreateInput, error) {
	input.Status = api.POLICY_DEFINITION_STATUS_READY
	domainIds := []string{}
	for _, _domain := range input.Domains {
		domain, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, _domain)
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrap(err, "FetchDomainByIdOrName"))
		}
		domainIds = append(domainIds, domain.GetId())
	}
	conditions, ok := api.POLICY_CONDITIONS[input.Category]
	if !ok {
		return input, httperrors.NewUnsupportOperationError("category %s not suppored", input.Category)
	}
	if !utils.IsInStringArray(input.Condition, conditions) {
		return input, httperrors.NewUnsupportOperationError("not support condition %s, support %s", input.Condition, conditions)
	}
	switch input.Category {
	case api.POLICY_DEFINITION_CATEGORY_CLOUDREGION:
		if len(input.Cloudregions) == 0 {
			return input, httperrors.NewMissingParameterError("cloudregions")
		}
		input.Parameters.Cloudregions = []api.SCloudregionPolicyDefinition{}
		for _, _region := range input.Cloudregions {
			region, err := CloudregionManager.FetchByIdOrName(userCred, _region)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return input, httperrors.NewResourceNotFoundError2("cloudregion", _region)
				}
				return input, httperrors.NewGeneralError(err)
			}
			input.Parameters.Cloudregions = append(input.Parameters.Cloudregions, api.SCloudregionPolicyDefinition{Id: region.GetId(), Name: region.GetName()})
		}
	case api.POLICY_DEFINITION_CATEGORY_EXPIRED:
		switch input.Condition {
		case api.POLICY_DEFINITION_CONDITION_IN_USE:
		case api.POLICY_DEFINITION_CONDITION_LE:
			if len(input.Duration) == 0 {
				return input, httperrors.NewMissingParameterError("duration")
			}
			_, err := billing.ParseBillingCycle(input.Duration)
			if err != nil {
				return input, httperrors.NewInputParameterError("invalid duration %v", err)
			}
			input.Parameters.Duration = input.Duration
		}
	case api.POLICY_DEFINITION_CATEGORY_TAG:
		if len(input.Tags) == 0 {
			return input, httperrors.NewMissingParameterError("tags")
		}
		input.Parameters.Tags = input.Tags
	case api.POLICY_DEFINITION_CATEGORY_BILLING_TYPE:
		if len(input.BillingType) == 0 {
			return input, httperrors.NewMissingParameterError("billing_type")
		}
		if !utils.IsInStringArray(input.BillingType, []string{billing_api.BILLING_TYPE_PREPAID, billing_api.BILLING_TYPE_POSTPAID}) {
			return input, httperrors.NewInputParameterError("invalid billing type %s", input.BillingType)
		}
		input.Parameters.BillingType = input.BillingType
	case api.POLICY_DEFINITION_CATEGORY_BATCH_CREATE:
		if input.Count <= 0 {
			return input, httperrors.NewInputParameterError("count must be greater than 0")
		}
		input.Parameters.Count = &input.Count
	}
	return input, nil
}

func (self *SPolicyDefinition) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := api.PolicyDefinitionCreateInput{}
	data.Unmarshal(&input)
	for _, domainId := range input.Domains {
		err := PolicyAssignmentManager.newAssignment(self, domainId)
		if err != nil {
			log.Errorf("failed to attach policy assignment for domain %s error: %v", domainId, err)
		}
	}
}

func (self *SPolicyDefinition) ValidateDeleteCondition(ctx context.Context) error {
	assignments, err := self.GetPolicyAssignments()
	if err != nil {
		return errors.Wrap(err, "GetPolicyAssignments")
	}
	if len(assignments) > 0 {
		return fmt.Errorf("%d available assignments on policy definition", len(assignments))
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
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
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SPolicyDefinition) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.PolicyDefinitionDetails, error) {
	return api.PolicyDefinitionDetails{}, nil
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

func (manager *SPolicyDefinitionManager) GetAvailablePolicyDefinitions(ctx context.Context, userCred mcclient.TokenCredential, category string) ([]SPolicyDefinition, error) {
	q := manager.Query()
	sq := PolicyAssignmentManager.Query().SubQuery()
	q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("policydefinition_id"))).Filter(
		sqlchemy.Equals(sq.Field("domain_id"), userCred.GetDomainId()),
	).Equals("status", api.POLICY_DEFINITION_STATUS_READY)
	if len(category) > 0 {
		q = q.Equals("category", category)
	}
	definitions := []SPolicyDefinition{}
	err := db.FetchModelObjects(manager, q, &definitions)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return definitions, nil
}

func (manager *SPolicyDefinitionManager) SyncPolicyDefinitions(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, provider *SCloudprovider, iDefinitions []cloudprovider.ICloudPolicyDefinition) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}

	dbDefinitions, err := manager.getPolicyDefinitionsByManagerId(provider.Id)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SPolicyDefinition, 0)
	commondb := make([]SPolicyDefinition, 0)
	commonext := make([]cloudprovider.ICloudPolicyDefinition, 0)
	added := make([]cloudprovider.ICloudPolicyDefinition, 0)

	err = compare.CompareSets(dbDefinitions, iDefinitions, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].purge(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
			continue
		}
		syncResult.Delete()
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudPolicyDefinition(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncResult.Update()
	}
	for i := 0; i < len(added); i += 1 {
		err = manager.newFromCloudPolicyDefinition(ctx, userCred, added[i], provider)
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncResult.Add()
	}
	return syncResult
}

func (self *SPolicyDefinition) constructParameters(ctx context.Context, userCred mcclient.TokenCredential, extDefinition cloudprovider.ICloudPolicyDefinition) error {
	self.Category = extDefinition.GetCategory()
	self.Condition = extDefinition.GetCondition()
	switch self.Category {
	case api.POLICY_DEFINITION_CATEGORY_CLOUDREGION:
		if !utils.IsInStringArray(self.Condition, []string{api.POLICY_DEFINITION_CONDITION_NOT_IN, api.POLICY_DEFINITION_CONDITION_IN}) {
			return fmt.Errorf("not support category %s condition %s", self.Category, self.Condition)
		}
		parameters := extDefinition.GetParameters()
		if parameters == nil {
			return fmt.Errorf("invalid parameters")
		}
		cloudregions := []string{}
		err := parameters.Unmarshal(&cloudregions, "cloudregions")
		if err != nil {
			return errors.Wrap(err, "parameters.Unmarshal")
		}
		regions := api.SCloudregionPolicyDefinitions{Cloudregions: []api.SCloudregionPolicyDefinition{}}
		for _, cloudregion := range cloudregions {
			region, err := db.FetchByExternalId(CloudregionManager, cloudregion)
			if err != nil {
				return errors.Wrapf(err, "db.FetchByExternalId(%s)", cloudregion)
			}
			regionPolicyDefinition := api.SCloudregionPolicyDefinition{
				Id:   region.GetId(),
				Name: region.GetName(),
			}
			regions.Cloudregions = append(regions.Cloudregions, regionPolicyDefinition)
		}
		self.Parameters = jsonutils.Marshal(regions).(*jsonutils.JSONDict)
	case api.POLICY_DEFINITION_CATEGORY_TAG:
		self.Parameters = extDefinition.GetParameters()
	default:
		return fmt.Errorf("not support category %s", self.Category)
	}
	self.Status = api.POLICY_DEFINITION_STATUS_READY
	return nil
}

func (manager *SPolicyDefinitionManager) newFromCloudPolicyDefinition(ctx context.Context, userCred mcclient.TokenCredential, extDefinition cloudprovider.ICloudPolicyDefinition, provider *SCloudprovider) error {
	definition := SPolicyDefinition{}
	definition.SetModelManager(manager, &definition)

	newName, err := db.GenerateName(manager, userCred, extDefinition.GetName())
	if err != nil {
		return errors.Wrap(err, "db.GenerateName")
	}

	definition.Name = newName
	definition.ManagerId = provider.Id
	definition.Status = api.POLICY_DEFINITION_STATUS_READY
	definition.ExternalId = extDefinition.GetGlobalId()
	definition.constructParameters(ctx, userCred, extDefinition)

	err = manager.TableSpec().Insert(&definition)
	if err != nil {
		return errors.Wrap(err, "Insert")
	}

	return PolicyAssignmentManager.newAssignment(&definition, provider.DomainId)
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

func (self *SPolicyDefinition) SyncWithCloudPolicyDefinition(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extDefinition cloudprovider.ICloudPolicyDefinition) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		return self.constructParameters(ctx, userCred, extDefinition)
	})
	if err != nil {
		return errors.Wrap(err, "db.UpdateWithLock")
	}
	return PolicyAssignmentManager.checkAndSetAssignment(self, provider.DomainId)
}

func (self *SPolicyDefinition) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.HasSystemAdminPrivilege()
}

// 同步策略状态
func (self *SPolicyDefinition) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PolicyDefinitionSyncstatusInput) (jsonutils.JSONObject, error) {
	if len(self.ManagerId) == 0 {
		return nil, nil
	}
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "PolicyDefinitionSyncstatusTask", "")
}
