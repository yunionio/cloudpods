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
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=dns_recordset
// +onecloud:swagger-gen-model-plural=dns_recordsets
type SDnsRecordSetManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager

	SDnsZoneResourceBaseManager
}

var DnsRecordSetManager *SDnsRecordSetManager

func init() {
	DnsRecordSetManager = &SDnsRecordSetManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SDnsRecordSet{},
			"dns_recordsets_tbl",
			"dns_recordset",
			"dns_recordsets",
		),
	}
	DnsRecordSetManager.SetVirtualObject(DnsRecordSetManager)
}

type SDnsRecordSet struct {
	db.SEnabledStatusStandaloneResourceBase
	SDnsZoneResourceBase

	DnsType    string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"domain" create:"domain_required"`
	DnsValue   string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"domain" create:"domain_required"`
	TTL        int64  `nullable:"false" list:"user" update:"domain" create:"domain_required" json:"ttl"`
	MxPriority int64  `nullable:"false" list:"user" update:"domain" create:"domain_optional"`
}

func (manager *SDnsRecordSetManager) EnableGenerateName() bool {
	return false
}

type SDnsRecordSetValidateInfo struct {
	api.SDnsRecordSet
	TrafficPolicies []api.DnsRecordPolicy
}

func validateDnsrecordPolicy(dnsType string, dnsZone *SDnsZone, trafficPolicies []api.DnsRecordPolicy) error {
	for _, policy := range trafficPolicies {
		if len(policy.Provider) == 0 {
			return httperrors.NewGeneralError(fmt.Errorf("missing traffic policy provider"))
		}
		factory, err := cloudprovider.GetProviderFactory(policy.Provider)
		if err != nil {
			return httperrors.NewGeneralError(errors.Wrapf(err, "invalid provider %s for traffic policy", policy.Provider))
		}
		_dnsTypes := factory.GetSupportedDnsTypes()
		dnsTypes, _ := _dnsTypes[cloudprovider.TDnsZoneType(dnsZone.ZoneType)]
		if ok, _ := utils.InArray(cloudprovider.TDnsType(dnsType), dnsTypes); !ok {
			return httperrors.NewNotSupportedError("%s %s not supported dns type %s", policy.Provider, dnsZone.ZoneType, dnsType)
		}
		_policyTypes := factory.GetSupportedDnsPolicyTypes()
		policyTypes, _ := _policyTypes[cloudprovider.TDnsZoneType(dnsZone.ZoneType)]
		if ok, _ := utils.InArray(cloudprovider.TDnsPolicyType(policy.PolicyType), policyTypes); !ok {
			return httperrors.NewNotSupportedError("%s %s not supported policy type %s", policy.Provider, dnsZone.ZoneType, policy.PolicyType)
		}
		_policyValues := factory.GetSupportedDnsPolicyValues()
		policyValues, _ := _policyValues[cloudprovider.TDnsPolicyType(policy.PolicyType)]
		if len(policyValues) > 0 {
			if len(policy.PolicyValue) == 0 {
				return httperrors.NewMissingParameterError(fmt.Sprintf("missing %s policy value", policy.Provider))
			}
			if isIn, _ := utils.InArray(cloudprovider.TDnsPolicyValue(policy.PolicyValue), policyValues); !isIn {
				return httperrors.NewNotSupportedError("%s %s %s not support %s", policy.Provider, dnsZone.ZoneType, policy.PolicyType, policy.PolicyValue)
			}
		}
	}
	return nil
}

// 创建
func (manager *SDnsRecordSetManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DnsRecordSetCreateInput) (api.DnsRecordSetCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}
	input.Name = strings.ToLower(input.Name)
	rrRegexp := regexp.MustCompile(`^(([a-zA-Z0-9_][a-zA-Z0-9_-]{0,61}[a-zA-Z0-9_])|[a-zA-Z0-9_]|\*{1}){1}(\.(([a-zA-Z0-9_][a-zA-Z0-9_-]{0,61}[a-zA-Z0-9_])|[a-zA-Z0-9_]))*$|^@{1}$`)
	if !rrRegexp.MatchString(input.Name) {
		return input, httperrors.NewInputParameterError("invalid record name %s", input.Name)
	}

	if len(input.DnsZoneId) == 0 {
		return input, httperrors.NewMissingParameterError("dns_zone_id")
	}
	_dnsZone, err := DnsZoneManager.FetchByIdOrName(userCred, input.DnsZoneId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewResourceNotFoundError2("dns_zone", input.DnsZoneId)
		}
		return input, httperrors.NewGeneralError(err)
	}
	dnsZone := _dnsZone.(*SDnsZone)

	validateInfo := &SDnsRecordSetValidateInfo{}

	validateInfo.TrafficPolicies = input.TrafficPolicies

	recordset := api.SDnsRecordSet{}
	recordset.DnsZoneId = input.DnsZoneId
	recordset.DnsType = input.DnsType
	recordset.DnsValue = input.DnsValue
	recordset.TTL = input.TTL
	recordset.MxPriority = input.MxPriority

	err = recordset.ValidateDnsrecordValue()
	if err != nil {
		return input, err
	}
	err = validateDnsrecordPolicy(input.DnsType, dnsZone, input.TrafficPolicies)
	if err != nil {
		return input, err
	}

	// 处理重复的记录

	// CNAME  dnsName不能和其他类型record相同

	// 同dnsName 同dnsType重复检查
	// 检查dnsrecord 是否通过policy重复
	// simple类型不能重复，不能和其他policy重复
	// 不同类型policy不能重复
	// 同类型policy的dnsrecord重复时，需要通过policyvalue区别

	// validate name type
	q := DnsRecordSetManager.Query().Equals("dns_zone_id", input.DnsZoneId).Equals("name", input.Name)
	recordTypeQuery := q
	switch input.DnsType {
	case "CNAME":
		recordTypeQuery = recordTypeQuery.NotEquals("dns_type", "CNAME")
	default:
		recordTypeQuery = recordTypeQuery.Equals("dns_type", "CNAME")
	}

	cnt, err := recordTypeQuery.CountWithError()
	if err != nil {
		return input, httperrors.NewGeneralError(err)
	}
	if cnt > 0 {
		return input, httperrors.NewNotSupportedError("duplicated with CNAME dnsrecord name not support")
	}

	//validate policy

	policyQuery := DnsRecordSetManager.Query().Equals("dns_zone_id", input.DnsZoneId).Equals("name", input.Name).Equals("dns_type", input.DnsType)
	if len(input.TrafficPolicies) > 0 {
		for i := range input.TrafficPolicies {
			dupeDnsrecord := DnsRecordSetManager.Query().Equals("dns_zone_id", input.DnsZoneId).Equals("name", input.Name).Equals("dns_type", input.DnsType).SubQuery()
			provider := input.TrafficPolicies[i].Provider
			policyType := input.TrafficPolicies[i].PolicyType
			policyValue := input.TrafficPolicies[i].PolicyValue

			policyQuery := dupeDnsrecord.Query()
			dnsrecordTrafficPolicies := DnsRecordSetTrafficPolicyManager.Query().SubQuery()
			dnstrafficPolicy := DnsTrafficPolicyManager.Query().SubQuery()

			policyQuery = policyQuery.LeftJoin(dnsrecordTrafficPolicies, sqlchemy.Equals(dupeDnsrecord.Field("id"), dnsrecordTrafficPolicies.Field("dns_recordset_id")))
			policyQuery = policyQuery.LeftJoin(dnstrafficPolicy, sqlchemy.Equals(dnsrecordTrafficPolicies.Field("dns_traffic_policy_id"), dnstrafficPolicy.Field("id")))
			policyQuery = policyQuery.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(dnsrecordTrafficPolicies.Field("dns_traffic_policy_id")), // 无trafficPolicy的重复dnsrecord
				sqlchemy.Contains(dnstrafficPolicy.Field("name"), "Simple"),                     // 简单 trafficPolicy的重复dnsrecord
				sqlchemy.AND(
					sqlchemy.Equals(dnstrafficPolicy.Field("provider"), provider), // 只看相同provider
					sqlchemy.OR(
						sqlchemy.NotEquals(dnstrafficPolicy.Field("policy_type"), policyType), // policy类型不同的重复dnsrecord
						sqlchemy.AND(
							sqlchemy.Equals(dnstrafficPolicy.Field("policy_type"), policyType), // policy类型相同但未通过policyvalue区分的的重复dnsrecord
							sqlchemy.Equals(dnstrafficPolicy.Field("policy_value"), policyValue)),
					),
				),
			))
			cnt, err = policyQuery.CountWithError()
			if err != nil {
				return input, httperrors.NewGeneralError(err)
			}
			if cnt > 0 {
				return input, httperrors.NewNotSupportedError("duplicated dnsrecord with existed dnsrecord can not distinguish by %s policy", provider)
			}
		}
	} else {
		cnt, err = policyQuery.CountWithError()
		if err != nil {
			return input, httperrors.NewGeneralError(err)
		}
		if cnt > 0 {
			return input, httperrors.NewNotSupportedError("duplicated dnsrecord with existed dnsrecord not support")
		}
	}

	input.Status = api.DNS_RECORDSET_STATUS_AVAILABLE
	input.DnsZoneId = dnsZone.Id
	return input, nil
}

func (self *SDnsRecordSet) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := api.DnsRecordSetCreateInput{}
	data.Unmarshal(&input)
	for _, policy := range input.TrafficPolicies {
		self.setTrafficPolicy(ctx, userCred, policy.Provider, cloudprovider.TDnsPolicyType(policy.PolicyType), cloudprovider.TDnsPolicyValue(policy.PolicyValue), policy.PolicyOptions)
	}

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return
	}
	logclient.AddSimpleActionLog(dnsZone, logclient.ACT_ALLOCATE, data, userCred, true)
	notifyclient.NotifyWebhook(ctx, userCred, self, notifyclient.ActionCreate)
	dnsZone.DoSyncRecords(ctx, userCred)
}

// DNS记录列表
func (manager *SDnsRecordSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsRecordSetListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = manager.SDnsZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DnsZoneFilterListBase)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// 解析详情
func (manager *SDnsRecordSetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DnsRecordSetDetails {
	rows := make([]api.DnsRecordSetDetails, len(objs))
	enRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	recordIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.DnsRecordSetDetails{
			EnabledStatusStandaloneResourceDetails: enRows[i],
		}
		record := objs[i].(*SDnsRecordSet)
		recordIds[i] = record.Id
	}

	q := DnsRecordSetTrafficPolicyManager.Query().In("dns_recordset_id", recordIds)
	recordSetPolicies := []SDnsRecordSetTrafficPolicy{}
	err := db.FetchModelObjects(DnsRecordSetTrafficPolicyManager, q, &recordSetPolicies)
	if err != nil {
		return rows
	}
	recordMaps := map[string][]string{}
	policyIds := []string{}
	for i := range recordSetPolicies {
		if !utils.IsInStringArray(recordSetPolicies[i].DnsTrafficPolicyId, policyIds) {
			policyIds = append(policyIds, recordSetPolicies[i].DnsTrafficPolicyId)
		}
		_, ok := recordMaps[recordSetPolicies[i].DnsRecordsetId]
		if !ok {
			recordMaps[recordSetPolicies[i].DnsRecordsetId] = []string{}
		}
		recordMaps[recordSetPolicies[i].DnsRecordsetId] = append(recordMaps[recordSetPolicies[i].DnsRecordsetId], recordSetPolicies[i].DnsTrafficPolicyId)
	}
	q = DnsTrafficPolicyManager.Query().In("id", policyIds)
	policies := []SDnsTrafficPolicy{}
	err = db.FetchModelObjects(DnsTrafficPolicyManager, q, &policies)
	if err != nil {
		return rows
	}
	policyMaps := map[string]api.DnsRecordPolicy{}
	for i := range policies {
		policyMaps[policies[i].Id] = api.DnsRecordPolicy{
			Provider:      policies[i].Provider,
			PolicyType:    policies[i].PolicyType,
			PolicyValue:   policies[i].PolicyValue,
			PolicyOptions: policies[i].Options,
		}
	}
	for i := range rows {
		rows[i].TrafficPolicies = []api.DnsRecordPolicy{}
		policyIds, ok := recordMaps[recordIds[i]]
		if ok {
			for _, policyId := range policyIds {
				policy, ok := policyMaps[policyId]
				if ok {
					rows[i].TrafficPolicies = append(rows[i].TrafficPolicies, policy)
				}
			}
		}
	}

	return rows
}

func (self *SDnsRecordSet) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	policies, err := self.GetDnsTrafficPolicies()
	if err != nil {
		return errors.Wrapf(err, "GetDnsTrafficPolicies for record %s(%s)", self.Name, self.Id)
	}
	for i := range policies {
		err = self.RemovePolicy(ctx, userCred, policies[i].Id)
		if err != nil {
			return errors.Wrapf(err, "RemovePolicy(%s)", policies[i].Id)
		}
	}
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

type sRecordUniqValues struct {
	DnsZoneId string
	DnsType   string
	DnsName   string
	DnsValue  string
}

func (self *SDnsRecordSet) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(sRecordUniqValues{
		DnsZoneId: self.DnsZoneId,
		DnsName:   self.Name,
		DnsType:   self.DnsType,
		DnsValue:  self.DnsValue,
	})
}

func (manager *SDnsRecordSetManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	values := &sRecordUniqValues{}
	data.Unmarshal(values)
	return jsonutils.Marshal(values)
}

func (manager *SDnsRecordSetManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	uniq := &sRecordUniqValues{}
	values.Unmarshal(uniq)
	if len(uniq.DnsZoneId) > 0 {
		q = q.Equals("dns_zone_id", uniq.DnsZoneId)
	}
	if len(uniq.DnsName) > 0 {
		q = q.Equals("name", uniq.DnsName)
	}
	if len(uniq.DnsType) > 0 {
		q = q.Equals("dns_type", uniq.DnsType)
	}
	if len(uniq.DnsValue) > 0 {
		q = q.Equals("dns_value", uniq.DnsValue)
	}

	return q
}

func (manager *SDnsRecordSetManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	dnsZoneId, _ := data.GetString("dns_zone_id")
	if len(dnsZoneId) > 0 {
		dnsZone, err := db.FetchById(DnsZoneManager, dnsZoneId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(DnsZoneManager, %s)", dnsZoneId)
		}
		return dnsZone.(*SDnsZone).GetOwnerId(), nil
	}
	return db.FetchDomainInfo(ctx, data)
}

func (manager *SDnsRecordSetManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	sq := DnsZoneManager.Query("id")
	sq = db.SharableManagerFilterByOwner(DnsZoneManager, sq, userCred, scope)
	return q.In("dns_zone_id", sq.SubQuery())
}

func (manager *SDnsRecordSetManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (self *SDnsRecordSet) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return false
	}
	return dnsZone.IsSharable(reqUsrId)
}

func (self *SDnsRecordSet) GetOwnerId() mcclient.IIdentityProvider {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil
	}
	return dnsZone.GetOwnerId()
}

func (self *SDnsRecordSet) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	policies, err := self.GetDnsTrafficPolicies()
	if err != nil {
		return errors.Wrapf(err, "GetDnsTrafficPolicies")
	}
	for i := range policies {
		err = self.RemovePolicy(ctx, userCred, policies[i].Id)
		if err != nil {
			return errors.Wrapf(err, "RemovePolicy")
		}
	}
	return self.Delete(ctx, userCred)
}

func (self *SDnsRecordSet) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	self.SEnabledStatusStandaloneResourceBase.PreDelete(ctx, userCred)

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return
	}
	logclient.AddSimpleActionLog(dnsZone, logclient.ACT_ALLOCATE, self, userCred, true)
	notifyclient.NotifyWebhook(ctx, userCred, self, notifyclient.ActionDelete)
	dnsZone.DoSyncRecords(ctx, userCred)
}

// 更新
func (self *SDnsRecordSet) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordSetUpdateInput) (api.DnsRecordSetUpdateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetDnsZone"))
	}

	if len(input.DnsType) == 0 {
		input.DnsType = self.DnsType
	}
	if len(input.DnsValue) == 0 {
		input.DnsValue = self.DnsValue
	}
	if input.TTL == nil {
		input.TTL = &self.TTL
	}
	if input.MxPriority == nil {
		input.MxPriority = &self.MxPriority
	}
	recordset := api.SDnsRecordSet{}
	recordset.DnsType = input.DnsType
	recordset.DnsValue = input.DnsValue
	recordset.TTL = *input.TTL
	recordset.MxPriority = *input.MxPriority

	err = recordset.ValidateDnsrecordValue()
	if err != nil {
		return input, err
	}
	err = validateDnsrecordPolicy(input.DnsType, dnsZone, input.TrafficPolicies)
	if err != nil {
		return input, err
	}

	// 处理重复的记录

	// CNAME  dnsName不能和其他类型record相同

	// 同dnsName 同dnsType重复检查
	// 检查dnsrecord 是否通过policy重复
	// simple类型不能重复，不能和其他policy重复
	// 不同类型policy不能重复
	// 同类型policy的dnsrecord重复时，需要通过policyvalue区别

	// validate name type
	q := DnsRecordSetManager.Query().Equals("dns_zone_id", dnsZone.Id).NotEquals("id", self.Id).Equals("name", input.Name)
	recordTypeQuery := q
	switch input.DnsType {
	case "CNAME":
		recordTypeQuery = recordTypeQuery.NotEquals("dns_type", "CNAME")
	default:
		recordTypeQuery = recordTypeQuery.Equals("dns_type", "CNAME")
	}

	cnt, err := recordTypeQuery.CountWithError()
	if err != nil {
		return input, httperrors.NewGeneralError(err)
	}
	if cnt > 0 {
		return input, httperrors.NewNotSupportedError("duplicated with CNAME dnsrecord name not support")
	}

	//validate policy
	policies, err := self.GetDnsTrafficPolicies()
	if err != nil {
		return input, httperrors.NewGeneralError(err)
	}
	policyMap := map[string]SDnsTrafficPolicy{}
	for i := range policies {
		policyMap[policies[i].Provider] = policies[i]
	}
	for i := range input.TrafficPolicies {
		inputPolicy := input.TrafficPolicies[i]
		policyMap[inputPolicy.Provider] = SDnsTrafficPolicy{
			Provider:    inputPolicy.Provider,
			PolicyType:  inputPolicy.PolicyType,
			PolicyValue: inputPolicy.PolicyValue,
			Options:     inputPolicy.PolicyOptions,
		}
	}

	policyQuery := DnsRecordSetManager.Query().Equals("dns_zone_id", dnsZone.Id).NotEquals("id", self.Id).Equals("name", input.Name).Equals("dns_type", input.DnsType)
	if len(policyMap) > 0 {
		for i := range policyMap {
			dupeDnsrecord := DnsRecordSetManager.Query().Equals("dns_zone_id", dnsZone.Id).NotEquals("id", self.Id).Equals("name", input.Name).Equals("dns_type", input.DnsType).SubQuery()
			provider := policyMap[i].Provider
			policyType := policyMap[i].PolicyType
			policyValue := policyMap[i].PolicyValue

			policyQuery := dupeDnsrecord.Query()
			dnsrecordTrafficPolicies := DnsRecordSetTrafficPolicyManager.Query().SubQuery()
			dnstrafficPolicy := DnsTrafficPolicyManager.Query().SubQuery()

			policyQuery = policyQuery.LeftJoin(dnsrecordTrafficPolicies, sqlchemy.Equals(dupeDnsrecord.Field("id"), dnsrecordTrafficPolicies.Field("dns_recordset_id")))
			policyQuery = policyQuery.LeftJoin(dnstrafficPolicy, sqlchemy.Equals(dnsrecordTrafficPolicies.Field("dns_traffic_policy_id"), dnstrafficPolicy.Field("id")))
			policyQuery = policyQuery.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(dnsrecordTrafficPolicies.Field("dns_traffic_policy_id")), // 无trafficPolicy的重复dnsrecord
				sqlchemy.Contains(dnstrafficPolicy.Field("name"), "Simple"),                     // 简单 trafficPolicy的重复dnsrecord
				sqlchemy.AND(
					sqlchemy.Equals(dnstrafficPolicy.Field("provider"), provider), // 只看相同provider
					sqlchemy.OR(
						sqlchemy.NotEquals(dnstrafficPolicy.Field("policy_type"), policyType), // policy类型不同的重复dnsrecord
						sqlchemy.AND(
							sqlchemy.Equals(dnstrafficPolicy.Field("policy_type"), policyType), // policy类型相同但未通过policyvalue区分的的重复dnsrecord
							sqlchemy.Equals(dnstrafficPolicy.Field("policy_value"), policyValue)),
					),
				),
			))
			cnt, err = policyQuery.CountWithError()
			if err != nil {
				return input, httperrors.NewGeneralError(err)
			}
			if cnt > 0 {
				return input, httperrors.NewNotSupportedError("duplicated dnsrecord with existed dnsrecord can not distinguish by %s policy", provider)
			}
		}
	} else {
		cnt, err = policyQuery.CountWithError()
		if err != nil {
			return input, httperrors.NewGeneralError(err)
		}
		if cnt > 0 {
			return input, httperrors.NewNotSupportedError("duplicated dnsrecord with existed dnsrecord not support")
		}
	}
	return input, nil
}

func (self *SDnsRecordSet) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)

	input := &api.DnsRecordSetUpdateInput{}
	data.Unmarshal(input)
	for _, policy := range input.TrafficPolicies {
		self.setTrafficPolicy(ctx, userCred, policy.Provider, cloudprovider.TDnsPolicyType(policy.PolicyType), cloudprovider.TDnsPolicyValue(policy.PolicyValue), policy.PolicyOptions)
	}

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return
	}
	logclient.AddSimpleActionLog(self, logclient.ACT_UPDATE, data, userCred, true)
	notifyclient.NotifyWebhook(ctx, userCred, self, notifyclient.ActionUpdate)
	dnsZone.DoSyncRecords(ctx, userCred)
}

func (self *SDnsRecordSet) GetDnsTrafficPolicies() ([]SDnsTrafficPolicy, error) {
	sq := DnsRecordSetTrafficPolicyManager.Query("dns_traffic_policy_id").Equals("dns_recordset_id", self.Id)
	q := DnsTrafficPolicyManager.Query().In("id", sq.SubQuery())
	policies := []SDnsTrafficPolicy{}
	err := db.FetchModelObjects(DnsTrafficPolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SDnsRecordSet) GetDnsTrafficPolicy(provider string) (*SDnsTrafficPolicy, error) {
	sq := DnsRecordSetTrafficPolicyManager.Query("dns_traffic_policy_id").Equals("dns_recordset_id", self.Id)
	q := DnsTrafficPolicyManager.Query().In("id", sq.SubQuery()).Equals("provider", provider)
	policies := []SDnsTrafficPolicy{}
	err := db.FetchModelObjects(DnsTrafficPolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(policies) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(policies) > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	return &policies[0], nil
}

func (self *SDnsRecordSet) GetDefaultDnsTrafficPolicy(provider string) (cloudprovider.TDnsPolicyType, cloudprovider.TDnsPolicyValue, *jsonutils.JSONDict, error) {
	policy, err := self.GetDnsTrafficPolicy(provider)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return cloudprovider.DnsPolicyTypeSimple, cloudprovider.DnsPolicyValueEmpty, nil, errors.Wrapf(err, "GetDnsTrafficPolicy(%s)", provider)
	}
	if policy != nil {
		return cloudprovider.TDnsPolicyType(policy.PolicyType), cloudprovider.TDnsPolicyValue(policy.PolicyValue), policy.Options, nil
	}
	return cloudprovider.DnsPolicyTypeSimple, cloudprovider.DnsPolicyValueEmpty, nil, nil
}

func (self *SDnsRecordSet) GetDnsZone() (*SDnsZone, error) {
	dnsZone, err := DnsZoneManager.FetchById(self.DnsZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "DnsZoneManager.FetchById(%s)", self.DnsZoneId)
	}
	return dnsZone.(*SDnsZone), nil
}

func (self *SDnsRecordSet) syncWithCloudDnsRecord(ctx context.Context, userCred mcclient.TokenCredential, provider string, ext cloudprovider.DnsRecordSet) error {
	_, err := db.Update(self, func() error {
		self.Name = ext.DnsName
		self.Enabled = tristate.NewFromBool(ext.Enabled)
		self.Status = ext.Status
		self.TTL = ext.Ttl
		self.MxPriority = ext.MxPriority
		self.DnsType = string(ext.DnsType)
		self.DnsValue = ext.DnsValue
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "update")
	}
	return self.setTrafficPolicy(ctx, userCred, provider, ext.PolicyType, ext.PolicyValue, ext.PolicyOptions)
}

func (self *SDnsRecordSet) RemovePolicy(ctx context.Context, userCred mcclient.TokenCredential, policyId string) error {
	q := DnsRecordSetTrafficPolicyManager.Query().Equals("dns_recordset_id", self.Id).Equals("dns_traffic_policy_id", policyId)
	removed := []SDnsRecordSetTrafficPolicy{}
	err := db.FetchModelObjects(DnsRecordSetTrafficPolicyManager, q, &removed)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range removed {
		err = removed[i].Detach(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Detach")
		}
	}
	return nil
}

func (self *SDnsRecordSet) setTrafficPolicy(ctx context.Context, userCred mcclient.TokenCredential, provider string, policyType cloudprovider.TDnsPolicyType, policyValue cloudprovider.TDnsPolicyValue, policyOptions *jsonutils.JSONDict) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	policy, err := self.GetDnsTrafficPolicy(provider)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return errors.Wrapf(err, "GetDnsTrafficPolicy(%s)", provider)
	}
	if policy != nil {
		if cloudprovider.TDnsPolicyType(policy.PolicyType) == policyType && cloudprovider.TDnsPolicyValue(policy.PolicyValue) == policyValue && cloudprovider.IsPolicyOptionEquals(policy.Options, policyOptions) {
			return nil
		}
		self.RemovePolicy(ctx, userCred, policy.Id)
	}

	policy, err = DnsTrafficPolicyManager.Register(ctx, userCred, provider, policyType, policyValue, policyOptions)
	if err != nil {
		return errors.Wrapf(err, "DnsTrafficPolicyManager.Register")
	}

	recordPolicy := &SDnsRecordSetTrafficPolicy{}
	recordPolicy.SetModelManager(DnsRecordSetTrafficPolicyManager, recordPolicy)
	recordPolicy.DnsRecordsetId = self.Id
	recordPolicy.DnsTrafficPolicyId = policy.Id

	return DnsRecordSetTrafficPolicyManager.TableSpec().Insert(ctx, recordPolicy)
}

func (self *SDnsRecordSet) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordEnableInput) bool {
	return db.IsDomainAllowPerform(userCred, self, "enable")
}

// 启用
func (self *SDnsRecordSet) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordEnableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, input.PerformEnableInput)
	if err != nil {
		return nil, err
	}
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetDnsZone"))
	}
	dnsZone.DoSyncRecords(ctx, userCred)
	return nil, nil
}

func (self *SDnsRecordSet) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordDisableInput) bool {
	return db.IsDomainAllowPerform(userCred, self, "disable")
}

// 禁用
func (self *SDnsRecordSet) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordDisableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, input.PerformDisableInput)
	if err != nil {
		return nil, err
	}
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetDnsZone"))
	}
	dnsZone.DoSyncRecords(ctx, userCred)
	return nil, nil
}

func (self *SDnsRecordSet) AllowPerformSetTrafficPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordDisableInput) bool {
	return db.IsDomainAllowPerform(userCred, self, "set-traffic-policies")
}

// 设置流量策略
func (self *SDnsRecordSet) PerformSetTrafficPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordSetTrafficPoliciesInput) (jsonutils.JSONObject, error) {
	if len(input.TrafficPolicies) == 0 {
		return nil, httperrors.NewMissingParameterError("traffic_policies")
	}
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetDnsZone"))
	}
	for _, policy := range input.TrafficPolicies {
		if len(policy.Provider) == 0 {
			return nil, httperrors.NewGeneralError(fmt.Errorf("missing traffic policy provider"))
		}
		factory, err := cloudprovider.GetProviderFactory(policy.Provider)
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "invalid provider %s for traffic policy", policy.Provider))
		}
		_dnsTypes := factory.GetSupportedDnsTypes()
		dnsTypes, _ := _dnsTypes[cloudprovider.TDnsZoneType(dnsZone.ZoneType)]
		if ok, _ := utils.InArray(cloudprovider.TDnsType(self.DnsType), dnsTypes); !ok {
			return nil, httperrors.NewNotSupportedError("%s %s not supported dns type %s", policy.Provider, dnsZone.ZoneType, self.DnsType)
		}
		_policyTypes := factory.GetSupportedDnsPolicyTypes()
		policyTypes, _ := _policyTypes[cloudprovider.TDnsZoneType(dnsZone.ZoneType)]
		if ok, _ := utils.InArray(cloudprovider.TDnsPolicyType(policy.PolicyType), policyTypes); !ok {
			return nil, httperrors.NewNotSupportedError("%s %s not supported policy type %s", policy.Provider, dnsZone.ZoneType, policy.PolicyType)
		}
		_policyValues := factory.GetSupportedDnsPolicyValues()
		policyValues, _ := _policyValues[cloudprovider.TDnsPolicyType(policy.PolicyType)]
		if len(policyValues) > 0 {
			if len(policy.PolicyValue) == 0 {
				return nil, httperrors.NewMissingParameterError(fmt.Sprintf("missing %s policy value", policy.Provider))
			}
			if isIn, _ := utils.InArray(cloudprovider.TDnsPolicyValue(policy.PolicyValue), policyValues); !isIn {
				return nil, httperrors.NewNotSupportedError("%s %s %s not support %s", policy.Provider, dnsZone.ZoneType, policy.PolicyType, policy.PolicyValue)
			}
		}
		err = self.setTrafficPolicy(ctx, userCred, policy.Provider, cloudprovider.TDnsPolicyType(policy.PolicyType), cloudprovider.TDnsPolicyValue(policy.PolicyValue), policy.PolicyOptions)
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "setTrafficPolicy"))
		}
	}

	dnsZone.DoSyncRecords(ctx, userCred)
	return nil, nil
}
