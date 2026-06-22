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

package aws

import (
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	SCOPE_REGIONAL   = "REGIONAL"
	SCOPE_CLOUDFRONT = "CLOUDFRONT"
)

var (
	WAF_SCOPES = []string{
		SCOPE_REGIONAL,
		SCOPE_CLOUDFRONT,
	}
)

type SWafRule struct {
	Action struct {
		Block struct {
		} `json:"Block"`
	} `json:"Action"`
	Name string `json:"Name"`
}

type SVisibilityConfig struct {
	CloudWatchMetricsEnabled bool
	MetricName               string
	SampledRequestsEnabled   bool
}

type SWebAcl struct {
	multicloud.SResourceBase
	AwsTags
	region *SRegion
	*sWafWebACL

	scope     string
	LockToken string
}

func (self *SWebAcl) GetIsAccessProduct() bool {
	return false
}

func (self *SWebAcl) GetAccessHeaders() []string {
	return []string{}
}

func (self *SWebAcl) GetHttpPorts() []int {
	return []int{}
}

func (self *SWebAcl) GetHttpsPorts() []int {
	return []int{}
}

func (self *SWebAcl) GetCname() string {
	return ""
}

func (self *SWebAcl) GetCertId() string {
	return ""
}

func (self *SWebAcl) GetCertName() string {
	return ""
}

func (self *SWebAcl) GetUpstreamScheme() string {
	return ""
}

func (self *SWebAcl) GetUpstreamPort() int {
	return 0
}

func (self *SWebAcl) GetSourceIps() []string {
	return []string{}
}

func (self *SWebAcl) GetCcList() []string {
	return []string{}
}

func (self *SRegion) ListWebACLs(scope string) ([]SWebAcl, error) {
	if scope == SCOPE_CLOUDFRONT && self.RegionId != "us-east-1" {
		return []SWebAcl{}, nil
	}
	ret := []SWebAcl{}
	params := map[string]interface{}{"Scope": scope}
	for {
		resp := struct {
			WebACLs    []SWebAcl
			NextMarker string
		}{}
		err := self.wafRequest("ListWebACLs", params, &resp)
		if err != nil {
			return nil, errors.Wrapf(err, "ListWebACLs")
		}
		ret = append(ret, resp.WebACLs...)
		if len(resp.NextMarker) == 0 {
			break
		}
		params["NextMarker"] = resp.NextMarker
	}
	return ret, nil
}

func (self *SRegion) GetWebAcl(id, name, scope string) (*SWebAcl, error) {
	params := map[string]interface{}{
		"Id":    id,
		"Name":  name,
		"Scope": scope,
	}
	resp := struct {
		WebACL    *sWafWebACL
		LockToken string
	}{}
	err := self.wafRequest("GetWebACL", params, &resp)
	if err != nil {
		if strings.Contains(err.Error(), "WAFNonexistentItemException") {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", err.Error())
		}
		return nil, errors.Wrapf(err, "GetWebAcl")
	}
	ret := &SWebAcl{region: self, scope: scope, sWafWebACL: resp.WebACL, LockToken: resp.LockToken}
	return ret, nil
}

func (self *SRegion) DeleteWebAcl(id, name, scope, lockToken string) error {
	params := map[string]interface{}{
		"Id":        id,
		"Name":      name,
		"Scope":     scope,
		"LockToken": lockToken,
	}
	err := self.wafRequest("DeleteWebACL", params, nil)
	return errors.Wrapf(err, "DeleteWebACL")
}

func (self *SRegion) ListResourcesForWebACL(resType, arn string) ([]string, error) {
	params := map[string]interface{}{
		"ResourceType": resType,
		"WebACLArn":    arn,
	}
	resp := struct {
		ResourceArns []string
	}{}
	err := self.wafRequest("ListResourcesForWebACL", params, &resp)
	if err != nil {
		return nil, errors.Wrapf(err, "ListResourcesForWebACL")
	}
	return resp.ResourceArns, nil
}

func (self *SRegion) GetICloudWafInstanceById(id string) (cloudprovider.ICloudWafInstance, error) {
	idInfo := strings.Split(id, "/")
	if len(idInfo) != 4 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "invalid arn %s", id)
	}
	scope := SCOPE_CLOUDFRONT
	if strings.HasSuffix(idInfo[0], "regional") {
		scope = SCOPE_REGIONAL
	}
	ins, err := self.GetWebAcl(idInfo[3], idInfo[2], scope)
	if err != nil {
		return nil, errors.Wrapf(err, "GetWebAcl(%s, %s, %s)", idInfo[3], idInfo[2], scope)
	}
	return ins, nil
}

func (self *SRegion) GetICloudWafInstances() ([]cloudprovider.ICloudWafInstance, error) {
	ret := []cloudprovider.ICloudWafInstance{}
	for _, scope := range WAF_SCOPES {
		ins, err := self.ListWebACLs(scope)
		if err != nil {
			return nil, errors.Wrapf(err, "ListWebACLs")
		}
		for i := range ins {
			ins[i].region = self
			ins[i].scope = scope
			ret = append(ret, &ins[i])
		}
	}
	return ret, nil
}

func (self *SWebAcl) GetEnabled() bool {
	return true
}

func (self *SWebAcl) GetGlobalId() string {
	return *self.ARN
}

func (self *SWebAcl) GetName() string {
	return *self.Name
}

func (self *SWebAcl) GetId() string {
	return *self.ARN
}

func (self *SWebAcl) GetWafType() cloudprovider.TWafType {
	if self.scope == SCOPE_CLOUDFRONT {
		return cloudprovider.WafTypeCloudFront
	}
	return cloudprovider.WafTypeRegional
}

func (self *SWebAcl) GetStatus() string {
	return api.WAF_STATUS_AVAILABLE
}

func (self *SWebAcl) GetDefaultAction() *cloudprovider.DefaultAction {
	ret := &cloudprovider.DefaultAction{}
	if self.sWafWebACL.DefaultAction == nil {
		self.Refresh()
	}
	if self.sWafWebACL.DefaultAction != nil {
		action := self.sWafWebACL.DefaultAction
		if action.Allow != nil {
			ret.Action = cloudprovider.WafActionAllow
		} else if action.Block != nil {
			ret.Action = cloudprovider.WafActionBlock
		}
	}
	return ret
}

func (self *SWebAcl) Refresh() error {
	acl, err := self.region.GetWebAcl(*self.Id, *self.Name, self.scope)
	if err != nil {
		return errors.Wrapf(err, "GetWebAcl")
	}
	self.sWafWebACL = acl.sWafWebACL
	return jsonutils.Update(self, acl)
}

func (self *SWebAcl) Delete() error {
	return self.region.DeleteWebAcl(*self.Id, *self.Name, self.scope, self.LockToken)
}

func (self *SRegion) CreateICloudWafInstance(opts *cloudprovider.WafCreateOptions) (cloudprovider.ICloudWafInstance, error) {
	waf, err := self.CreateWebAcl(opts.Name, opts.Desc, opts.Type, opts.DefaultAction)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateWebAcl")
	}
	return waf, nil
}

func (self *SRegion) CreateWebAcl(name, desc string, wafType cloudprovider.TWafType, action *cloudprovider.DefaultAction) (*SWebAcl, error) {
	var scope string
	switch wafType {
	case cloudprovider.WafTypeRegional, cloudprovider.WafTypeCloudFront:
		scope = strings.ToUpper(string(wafType))
	default:
		return nil, errors.Errorf("invalid waf type %s", wafType)
	}
	params := map[string]interface{}{
		"Name":  name,
		"Scope": scope,
		"VisibilityConfig": &sWafVisibilityConfig{
			SampledRequestsEnabled:   true,
			CloudWatchMetricsEnabled: true,
			MetricName:               name,
		},
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	if action != nil {
		defaultAction := &sWafDefaultAction{}
		switch action.Action {
		case cloudprovider.WafActionAllow:
			defaultAction.Allow = &sWafAllowAction{}
		case cloudprovider.WafActionBlock:
			defaultAction.Block = &sWafBlockAction{}
		}
		params["DefaultAction"] = defaultAction
	}
	resp := struct {
		Summary struct {
			Id string
		}
	}{}
	err := self.wafRequest("CreateWebACL", params, &resp)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateWebAcl")
	}
	return self.GetWebAcl(resp.Summary.Id, name, scope)
}

func reverseConvertField(opts cloudprovider.SWafStatement) *sWafFieldToMatch {
	ret := &sWafFieldToMatch{}
	switch opts.MatchField {
	case cloudprovider.WafMatchFieldBody:
		ret.Body = &sWafBody{}
	case cloudprovider.WafMatchFieldJsonBody:
	case cloudprovider.WafMatchFieldMethod:
		ret.Method = &sWafMethod{}
	case cloudprovider.WafMatchFieldQuery:
		switch opts.MatchFieldKey {
		case "SingleArgument":
			ret.SingleQueryArgument = &sWafSingleQueryArgument{}
		case "AllArguments":
			ret.AllQueryArguments = &sWafAllQueryArguments{}
		default:
			ret.QueryString = &sWafQueryString{}
		}
	case cloudprovider.WafMatchFiledHeader:
		ret.SingleHeader = &sWafSingleHeader{Name: awsWafString(opts.MatchFieldKey)}
	case cloudprovider.WafMatchFiledUriPath:
		ret.UriPath = &sWafUriPath{}
	}
	return ret
}

func reverseConvertStatement(statement cloudprovider.SWafStatement) *sWafStatement {
	ret := &sWafStatement{}
	trans := []*sWafTextTransformation{}
	if statement.Transformations != nil {
		for i, tran := range *statement.Transformations {
			t := &sWafTextTransformation{Priority: awsWafInt64(int64(i))}
			switch tran {
			case cloudprovider.WafTextTransformationNone:
				t.Type = awsWafString(wafTextTransformationTypeNone)
			case cloudprovider.WafTextTransformationLowercase:
				t.Type = awsWafString(wafTextTransformationTypeLowercase)
			case cloudprovider.WafTextTransformationCmdLine:
				t.Type = awsWafString(wafTextTransformationTypeCmdLine)
			case cloudprovider.WafTextTransformationUrlDecode:
				t.Type = awsWafString(wafTextTransformationTypeUrlDecode)
			case cloudprovider.WafTextTransformationHtmlEntityDecode:
				t.Type = awsWafString(wafTextTransformationTypeHtmlEntityDecode)
			case cloudprovider.WafTextTransformationCompressWithSpace:
				t.Type = awsWafString(wafTextTransformationTypeCompressWhiteSpace)
			}
			trans = append(trans, t)
		}
	}
	rules := []*sWafExcludedRule{}
	if statement.ExcludeRules != nil {
		for _, r := range *statement.ExcludeRules {
			name := r.Name
			rules = append(rules, &sWafExcludedRule{Name: &name})
		}
	}
	field := reverseConvertField(statement)
	switch statement.Type {
	case cloudprovider.WafStatementTypeRate:
		rate := &sWafRateBasedStatement{AggregateKeyType: awsWafString("IP")}
		limit := int(0)
		if statement.MatchFieldValues != nil && len(*statement.MatchFieldValues) == 1 {
			limit, _ = strconv.Atoi((*statement.MatchFieldValues)[0])
		}
		rate.Limit = awsWafInt64(int64(limit))
		if len(statement.ForwardedIPHeader) > 0 {
			rate.ForwardedIPConfig = &sWafForwardedIPConfig{HeaderName: awsWafString(statement.ForwardedIPHeader)}
		}
		ret.RateBasedStatement = rate
	case cloudprovider.WafStatementTypeIPSet:
		ipset := &sWafIPSetReferenceStatement{ARN: awsWafString(statement.IPSetId)}
		if len(statement.ForwardedIPHeader) > 0 {
			ipset.IPSetForwardedIPConfig = &sWafIPSetForwardedIPConfig{HeaderName: awsWafString(statement.ForwardedIPHeader)}
		}
		ret.IPSetReferenceStatement = ipset
	case cloudprovider.WafStatementTypeXssMatch:
		xss := &sWafXssMatchStatement{FieldToMatch: field, TextTransformations: trans}
		ret.XssMatchStatement = xss
	case cloudprovider.WafStatementTypeSize:
		size := &sWafSizeConstraintStatement{FieldToMatch: field}
		value := int(0)
		if statement.MatchFieldValues != nil && len(*statement.MatchFieldValues) == 1 {
			value, _ = strconv.Atoi((*statement.MatchFieldValues)[0])
		}
		size.Size = awsWafInt64(int64(value))
		ret.SizeConstraintStatement = size
	case cloudprovider.WafStatementTypeGeoMatch:
		geo := &sWafGeoMatchStatement{}
		values := []*string{}
		if statement.MatchFieldValues != nil {
			for i := range *statement.MatchFieldValues {
				v := (*statement.MatchFieldValues)[i]
				values = append(values, &v)
			}
			geo.CountryCodes = values
		}
		if len(statement.ForwardedIPHeader) > 0 {
			geo.ForwardedIPConfig = &sWafForwardedIPConfig{HeaderName: awsWafString(statement.ForwardedIPHeader)}
		}
		ret.GeoMatchStatement = geo
	case cloudprovider.WafStatementTypeRegexSet:
		regex := &sWafRegexPatternSetReferenceStatement{
			ARN:                 awsWafString(statement.RegexSetId),
			FieldToMatch:        field,
			TextTransformations: trans,
		}
		ret.RegexPatternSetReferenceStatement = regex
	case cloudprovider.WafStatementTypeByteMatch:
		bm := &sWafByteMatchStatement{
			FieldToMatch:        field,
			SearchString:        []byte(statement.SearchString),
			TextTransformations: trans,
		}
		if len(statement.Operator) > 0 {
			bm.PositionalConstraint = awsWafString(string(statement.Operator))
		}
		ret.ByteMatchStatement = bm
	case cloudprovider.WafStatementTypeRuleGroup:
		rg := &sWafRuleGroupReferenceStatement{ARN: awsWafString(statement.RuleGroupId), ExcludedRules: rules}
		ret.RuleGroupReferenceStatement = rg
	case cloudprovider.WafStatementTypeSqliMatch:
		sqli := &sWafSqliMatchStatement{FieldToMatch: field, TextTransformations: trans}
		ret.SqliMatchStatement = sqli
	case cloudprovider.WafStatementTypeLabelMatch:
	case cloudprovider.WafStatementTypeManagedRuleGroup:
		rg := &sWafManagedRuleGroupStatement{
			Name:          awsWafString(statement.ManagedRuleGroupName),
			VendorName:    awsWafString("aws"),
			ExcludedRules: rules,
		}
		ret.ManagedRuleGroupStatement = rg
	}
	return ret
}

func (self *SWebAcl) AddRule(opts *cloudprovider.SWafRule) (cloudprovider.ICloudWafRule, error) {
	rules := self.Rules
	rule := &sWafRuleItem{
		Name:     awsWafString(opts.Name),
		Priority: awsWafInt64(int64(opts.Priority)),
	}
	if opts.Action != nil {
		action := &sWafRuleAction{}
		switch opts.Action.Action {
		case cloudprovider.WafActionAllow:
			action.Allow = &sWafAllowAction{}
		case cloudprovider.WafActionBlock:
			action.Block = &sWafBlockAction{}
		case cloudprovider.WafActionCount:
			action.Count = &sWafCountAction{}
		}
		rule.Action = action
	}
	rule.VisibilityConfig = &sWafVisibilityConfig{
		SampledRequestsEnabled:   false,
		CloudWatchMetricsEnabled: true,
		MetricName:               opts.Name,
	}
	statement := &sWafStatement{}
	switch opts.StatementCondition {
	case cloudprovider.WafStatementConditionOr:
		ss := &sWafOrStatement{}
		for _, s := range opts.Statements {
			ss.Statements = append(ss.Statements, reverseConvertStatement(s))
		}
		statement.OrStatement = ss
	case cloudprovider.WafStatementConditionAnd:
		ss := &sWafAndStatement{}
		for _, s := range opts.Statements {
			ss.Statements = append(ss.Statements, reverseConvertStatement(s))
		}
		statement.AndStatement = ss
	case cloudprovider.WafStatementConditionNot:
		ss := &sWafNotStatement{}
		for _, s := range opts.Statements {
			ss.Statement = reverseConvertStatement(s)
			break
		}
		statement.NotStatement = ss
	case cloudprovider.WafStatementConditionNone:
		for _, s := range opts.Statements {
			statement = reverseConvertStatement(s)
			break
		}
	}
	rule.Statement = statement
	rules = append(rules, rule)
	params := map[string]interface{}{
		"LockToken":        self.LockToken,
		"Id":               *self.Id,
		"Name":             *self.Name,
		"Scope":            self.scope,
		"Description":      *self.Description,
		"DefaultAction":    self.DefaultAction,
		"VisibilityConfig": self.VisibilityConfig,
		"Rules":            rules,
	}
	err := self.region.wafRequest("UpdateWebACL", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "UpdateWebACL")
	}
	ret := &sWafRule{waf: self, sWafRuleItem: rule}
	return ret, nil
}

func (self *SWebAcl) GetCloudResources() ([]cloudprovider.SCloudResource, error) {
	ret := []cloudprovider.SCloudResource{}
	if self.scope != SCOPE_REGIONAL {
		return ret, nil
	}
	for _, resType := range []string{"APPLICATION_LOAD_BALANCER", "API_GATEWAY", "APPSYNC"} {
		resIds, err := self.region.ListResourcesForWebACL(resType, *self.ARN)
		if err != nil {
			return nil, errors.Wrapf(err, "ListResourcesForWebACL(%s, %s)", resType, *self.ARN)
		}
		for _, resId := range resIds {
			ret = append(ret, cloudprovider.SCloudResource{
				Id:   resId,
				Name: resId,
				Type: resType,
			})
		}
	}
	return ret, nil
}

func (self *SWebAcl) GetDescription() string {
	return self.AwsTags.GetDescription()
}
