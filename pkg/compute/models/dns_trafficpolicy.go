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
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDnsTrafficPolicyManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
}

var DnsTrafficPolicyManager *SDnsTrafficPolicyManager

func init() {
	DnsTrafficPolicyManager = &SDnsTrafficPolicyManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SDnsTrafficPolicy{},
			"dns_traffic_policy_tbl",
			"dns_trafficpolicy",
			"dns_trafficpolicies",
		),
	}
	DnsTrafficPolicyManager.SetVirtualObject(DnsTrafficPolicyManager)

}

type SDnsTrafficPolicy struct {
	db.SEnabledStatusInfrasResourceBase

	Provider    string              `width:"64" charset:"ascii" list:"domain" create:"domain_required"`
	PolicyType  string              `width:"32" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
	PolicyValue string              `width:"32" charset:"ascii" nullable:"false" list:"domain" create:"domain_optional"`
	Options     *jsonutils.JSONDict `get:"domain" list:"domain" create:"domain_optional"`
}

// 创建
func (manager *SDnsTrafficPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DnsTrafficPolicyCreateInput) (api.DnsTrafficPolicyCreateInput, error) {
	if len(input.Provider) == 0 {
		return input, httperrors.NewMissingParameterError("provider")
	}
	factory, err := cloudprovider.GetProviderFactory(input.Provider)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetProviderFactory(%s)", input.Provider))
	}
	types := []cloudprovider.TDnsPolicyType{}
	policyTypes := factory.GetSupportedDnsPolicyTypes()
	for _, _policyTypes := range policyTypes {
		types = append(types, _policyTypes...)
	}
	if isIn, _ := utils.InArray(cloudprovider.TDnsPolicyType(input.PolicyType), types); !isIn {
		return input, httperrors.NewUnsupportOperationError("%s not support policy type %s", input.Provider, input.PolicyType)
	}
	policyValues := factory.GetSupportedDnsPolicyValues()
	values, _ := policyValues[cloudprovider.TDnsPolicyType(input.PolicyType)]
	if len(values) > 0 {
		if isIn, _ := utils.InArray(cloudprovider.TDnsPolicyValue(input.PolicyValue), values); !isIn {
			return input, httperrors.NewUnsupportOperationError("%s %s not support policy value %s", input.Provider, input.PolicyType, input.PolicyValue)
		}
	}
	return input, nil
}

func (manager *SDnsTrafficPolicyManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, manager)
}

// 列表
func (manager *SDnsTrafficPolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsTrafficPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if len(query.PolicyType) > 0 {
		q = q.Equals("policy_type", query.PolicyType)
	}
	if len(query.Provider) > 0 {
		q = q.In("provider", query.Provider)
	}
	return q, nil
}

func (self *SDnsTrafficPolicy) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsDomainAllowUpdate(userCred, self)
}

// 详情
func (self *SDnsTrafficPolicy) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.DnsTrafficPolicyDetails, error) {
	return api.DnsTrafficPolicyDetails{}, nil
}

func (manager *SDnsTrafficPolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DnsTrafficPolicyDetails {
	rows := make([]api.DnsTrafficPolicyDetails, len(objs))
	enRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.DnsTrafficPolicyDetails{
			EnabledStatusInfrasResourceBaseDetails: enRows[i],
		}
	}
	return rows
}

func (manager *SDnsTrafficPolicyManager) Register(ctx context.Context, userCred mcclient.TokenCredential, provider string, policyType cloudprovider.TDnsPolicyType, policyValue cloudprovider.TDnsPolicyValue, options *jsonutils.JSONDict) (*SDnsTrafficPolicy, error) {
	q := manager.Query().Equals("provider", provider).Equals("policy_type", policyType).Equals("policy_value", policyValue)
	if options != nil {
		q = q.Equals("options", options.String())
	} else {
		q = q.IsNullOrEmpty("options")
	}
	policies := []SDnsTrafficPolicy{}
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(policies) > 0 {
		return &policies[0], nil
	}
	policy := &SDnsTrafficPolicy{}
	policy.SetModelManager(manager, policy)
	policy.PolicyType = string(policyType)
	policy.PolicyValue = string(policyValue)
	policy.Provider = provider
	policy.Options = options
	return func() (*SDnsTrafficPolicy, error) {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		policy.Name, err = db.GenerateName(ctx, manager, userCred, fmt.Sprintf("%s-%s", provider, policyType))
		if err != nil {
			return nil, errors.Wrapf(err, "db.GenerateName")
		}

		return policy, manager.TableSpec().Insert(ctx, policy)
	}()
}
