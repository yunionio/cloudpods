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

	"github.com/aws/aws-sdk-go/service/wafv2"

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
	*wafv2.WebACL

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
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	ret := []SWebAcl{}
	input := wafv2.ListWebACLsInput{}
	input.SetScope(scope)
	for {
		resp, err := client.ListWebACLs(&input)
		if err != nil {
			return nil, errors.Wrapf(err, "ListWebACLs")
		}
		part := []SWebAcl{}
		jsonutils.Update(&part, resp.WebACLs)
		ret = append(ret, part...)
		if resp.NextMarker == nil || len(*resp.NextMarker) == 0 {
			break
		}
		input.SetNextMarker(*resp.NextMarker)
	}
	return ret, nil
}

func (self *SRegion) GetWebAcl(id, name, scope string) (*SWebAcl, error) {
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	input := wafv2.GetWebACLInput{}
	input.SetId(id)
	input.SetName(name)
	input.SetScope(scope)
	resp, err := client.GetWebACL(&input)
	if err != nil {
		if _, ok := err.(*wafv2.WAFNonexistentItemException); ok {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
		}
		return nil, errors.Wrapf(err, "GetWebAcl")
	}
	ret := &SWebAcl{region: self, scope: scope, WebACL: resp.WebACL, LockToken: *resp.LockToken}
	return ret, nil
}

func (self *SRegion) DeleteWebAcl(id, name, scope, lockToken string) error {
	client, err := self.getWafClient()
	if err != nil {
		return errors.Wrapf(err, "getWafClient")
	}
	input := wafv2.DeleteWebACLInput{}
	input.SetId(id)
	input.SetName(name)
	input.SetScope(scope)
	input.SetLockToken(lockToken)
	_, err = client.DeleteWebACL(&input)
	return errors.Wrapf(err, "DeleteWebACL")
}

func (self *SRegion) ListResourcesForWebACL(resType, arn string) ([]string, error) {
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	input := wafv2.ListResourcesForWebACLInput{}
	input.SetResourceType(resType)
	input.SetWebACLArn(arn)
	resp, err := client.ListResourcesForWebACL(&input)
	if err != nil {
		return nil, errors.Wrapf(err, "ListResourcesForWebACL")
	}
	ret := []string{}
	for _, id := range resp.ResourceArns {
		ret = append(ret, *id)
	}
	return ret, nil
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
	if self.WebACL.DefaultAction == nil {
		self.Refresh()
	}
	if self.WebACL.DefaultAction != nil {
		action := self.WebACL.DefaultAction
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
	self.WebACL = acl.WebACL
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
	input := wafv2.CreateWebACLInput{}
	input.SetName(name)
	if len(desc) > 0 {
		input.SetDescription(desc)
	}
	switch wafType {
	case cloudprovider.WafTypeRegional, cloudprovider.WafTypeCloudFront:
		input.SetScope(strings.ToUpper(string(wafType)))
	default:
		return nil, errors.Errorf("invalid waf type %s", wafType)
	}
	if action != nil {
		defaultAction := wafv2.DefaultAction{}
		switch action.Action {
		case cloudprovider.WafActionAllow:
			defaultAction.Allow = &wafv2.AllowAction{}
		case cloudprovider.WafActionBlock:
			defaultAction.Block = &wafv2.BlockAction{}
		}
		input.SetDefaultAction(&defaultAction)
	}
	visib := &wafv2.VisibilityConfig{}
	visib.SetSampledRequestsEnabled(true)
	visib.SetCloudWatchMetricsEnabled(true)
	visib.SetMetricName(name)
	input.SetVisibilityConfig(visib)
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	output, err := client.CreateWebACL(&input)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateWebAcl")
	}
	return self.GetWebAcl(*output.Summary.Id, name, *input.Scope)
}

func reverseConvertField(opts cloudprovider.SWafStatement) *wafv2.FieldToMatch {
	ret := &wafv2.FieldToMatch{}
	switch opts.MatchField {
	case cloudprovider.WafMatchFieldBody:
		body := &wafv2.Body{}
		ret.SetBody(body)
	case cloudprovider.WafMatchFieldJsonBody:
	case cloudprovider.WafMatchFieldMethod:
		method := &wafv2.Method{}
		ret.SetMethod(method)
	case cloudprovider.WafMatchFieldQuery:
		switch opts.MatchFieldKey {
		case "SingleArgument":
			query := &wafv2.SingleQueryArgument{}
			ret.SetSingleQueryArgument(query)
		case "AllArguments":
			query := &wafv2.AllQueryArguments{}
			ret.SetAllQueryArguments(query)
		default:
			query := &wafv2.QueryString{}
			ret.SetQueryString(query)
		}
	case cloudprovider.WafMatchFiledHeader:
		head := &wafv2.SingleHeader{}
		head.SetName(opts.MatchFieldKey)
		ret.SetSingleHeader(head)
	case cloudprovider.WafMatchFiledUriPath:
		uri := &wafv2.UriPath{}
		ret.SetUriPath(uri)
	}
	return ret
}

func reverseConvertStatement(statement cloudprovider.SWafStatement) *wafv2.Statement {
	ret := &wafv2.Statement{}
	trans := []*wafv2.TextTransformation{}
	if statement.Transformations != nil {
		for i, tran := range *statement.Transformations {
			t := &wafv2.TextTransformation{}
			switch tran {
			case cloudprovider.WafTextTransformationNone:
				t.SetType(wafv2.TextTransformationTypeNone)
			case cloudprovider.WafTextTransformationLowercase:
				t.SetType(wafv2.TextTransformationTypeLowercase)
			case cloudprovider.WafTextTransformationCmdLine:
				t.SetType(wafv2.TextTransformationTypeCmdLine)
			case cloudprovider.WafTextTransformationUrlDecode:
				t.SetType(wafv2.TextTransformationTypeUrlDecode)
			case cloudprovider.WafTextTransformationHtmlEntityDecode:
				t.SetType(wafv2.TextTransformationTypeHtmlEntityDecode)
			case cloudprovider.WafTextTransformationCompressWithSpace:
				t.SetType(wafv2.TextTransformationTypeCompressWhiteSpace)
			}
			t.SetPriority(int64(i))
			trans = append(trans, t)
		}
	}
	rules := []*wafv2.ExcludedRule{}
	if statement.ExcludeRules != nil {
		for _, r := range *statement.ExcludeRules {
			name := r.Name
			rules = append(rules, &wafv2.ExcludedRule{
				Name: &name,
			})
		}
	}
	field := reverseConvertField(statement)
	switch statement.Type {
	case cloudprovider.WafStatementTypeRate:
		rate := &wafv2.RateBasedStatement{}
		limit := int(0)
		if statement.MatchFieldValues != nil && len(*statement.MatchFieldValues) == 1 {
			limit, _ = strconv.Atoi((*statement.MatchFieldValues)[0])
		}
		rate.SetLimit(int64(limit))
		fd := &wafv2.ForwardedIPConfig{}
		if len(statement.ForwardedIPHeader) > 0 {
			fd.SetHeaderName(statement.ForwardedIPHeader)
			rate.SetForwardedIPConfig(fd)
		}
		ret.SetRateBasedStatement(rate)
	case cloudprovider.WafStatementTypeIPSet:
		ipset := &wafv2.IPSetReferenceStatement{}
		ipset.SetARN(statement.IPSetId)
		fd := &wafv2.IPSetForwardedIPConfig{}
		if len(statement.ForwardedIPHeader) > 0 {
			fd.SetHeaderName(statement.ForwardedIPHeader)
			ipset.SetIPSetForwardedIPConfig(fd)
		}
		ret.SetIPSetReferenceStatement(ipset)
	case cloudprovider.WafStatementTypeXssMatch:
		xss := &wafv2.XssMatchStatement{}
		if len(trans) > 0 {
			xss.SetTextTransformations(trans)
		}
		field := &wafv2.FieldToMatch{}
		xss.SetFieldToMatch(field)
		xss.SetTextTransformations(trans)
		ret.SetXssMatchStatement(xss)
	case cloudprovider.WafStatementTypeSize:
		size := &wafv2.SizeConstraintStatement{}
		size.SetFieldToMatch(field)
		value := int(0)
		if statement.MatchFieldValues != nil && len(*statement.MatchFieldValues) == 1 {
			value, _ = strconv.Atoi((*statement.MatchFieldValues)[0])
		}
		size.SetSize(int64(value))
		ret.SetSizeConstraintStatement(size)
	case cloudprovider.WafStatementTypeGeoMatch:
		geo := &wafv2.GeoMatchStatement{}
		values := []*string{}
		if statement.MatchFieldValues != nil {
			for i := range *statement.MatchFieldValues {
				v := (*statement.MatchFieldValues)[i]
				values = append(values, &v)
			}
			geo.SetCountryCodes(values)
		}
		fd := &wafv2.ForwardedIPConfig{}
		if len(statement.ForwardedIPHeader) > 0 {
			fd.SetHeaderName(statement.ForwardedIPHeader)
			geo.SetForwardedIPConfig(fd)
		}
		ret.SetGeoMatchStatement(geo)
	case cloudprovider.WafStatementTypeRegexSet:
		regex := &wafv2.RegexPatternSetReferenceStatement{}
		regex.SetARN(statement.RegexSetId)
		if len(trans) > 0 {
			regex.SetTextTransformations(trans)
		}
		regex.SetFieldToMatch(field)
		ret.SetRegexPatternSetReferenceStatement(regex)
	case cloudprovider.WafStatementTypeByteMatch:
		bm := &wafv2.ByteMatchStatement{}
		if len(trans) > 0 {
			bm.SetTextTransformations(trans)
		}
		bm.SetSearchString([]byte(statement.SearchString))
		if len(statement.Operator) > 0 {
			bm.SetPositionalConstraint(string(statement.Operator))
		}
		bm.SetFieldToMatch(field)
		ret.SetByteMatchStatement(bm)
	case cloudprovider.WafStatementTypeRuleGroup:
		rg := &wafv2.RuleGroupReferenceStatement{}
		rg.SetARN(statement.RuleGroupId)
		if len(rules) > 0 {
			rg.SetExcludedRules(rules)
		}
		ret.SetRuleGroupReferenceStatement(rg)
	case cloudprovider.WafStatementTypeSqliMatch:
		sqli := &wafv2.SqliMatchStatement{}
		if len(trans) > 0 {
			sqli.SetTextTransformations(trans)
		}
		sqli.SetFieldToMatch(field)
		ret.SetSqliMatchStatement(sqli)
	case cloudprovider.WafStatementTypeLabelMatch:
	case cloudprovider.WafStatementTypeManagedRuleGroup:
		rg := &wafv2.ManagedRuleGroupStatement{}
		rg.SetName(statement.ManagedRuleGroupName)
		rg.SetVendorName("aws")
		if len(rules) > 0 {
			rg.SetExcludedRules(rules)
		}
		ret.SetManagedRuleGroupStatement(rg)
	}
	return ret
}

func (self *SWebAcl) AddRule(opts *cloudprovider.SWafRule) (cloudprovider.ICloudWafRule, error) {
	input := &wafv2.UpdateWebACLInput{}
	input.SetLockToken(self.LockToken)
	input.SetId(*self.Id)
	input.SetName(*self.Name)
	input.SetScope(self.scope)
	input.SetDescription(*self.Description)
	input.SetDefaultAction(self.DefaultAction)
	input.SetVisibilityConfig(self.WebACL.VisibilityConfig)
	rules := self.Rules
	rule := &wafv2.Rule{}
	rule.SetName(opts.Name)
	rule.SetPriority(int64(opts.Priority))
	action := &wafv2.RuleAction{}
	if opts.Action != nil {
		switch opts.Action.Action {
		case cloudprovider.WafActionAllow:
			allow := &wafv2.AllowAction{}
			action.SetAllow(allow)
		case cloudprovider.WafActionBlock:
			block := &wafv2.BlockAction{}
			action.SetBlock(block)
		case cloudprovider.WafActionCount:
			count := &wafv2.CountAction{}
			action.SetCount(count)
		}
	}
	rule.SetAction(action)
	visib := &wafv2.VisibilityConfig{}
	visib.SetSampledRequestsEnabled(false)
	visib.SetCloudWatchMetricsEnabled(true)
	visib.SetMetricName(opts.Name)
	rule.SetVisibilityConfig(visib)
	statement := &wafv2.Statement{}
	switch opts.StatementCondition {
	case cloudprovider.WafStatementConditionOr:
		ss := &wafv2.OrStatement{}
		for _, s := range opts.Statements {
			ss.Statements = append(ss.Statements, reverseConvertStatement(s))
		}
		statement.SetOrStatement(ss)
	case cloudprovider.WafStatementConditionAnd:
		ss := &wafv2.AndStatement{}
		for _, s := range opts.Statements {
			ss.Statements = append(ss.Statements, reverseConvertStatement(s))
		}
		statement.SetAndStatement(ss)
	case cloudprovider.WafStatementConditionNot:
		ss := &wafv2.NotStatement{}
		for _, s := range opts.Statements {
			ss.SetStatement(reverseConvertStatement(s))
			break
		}
		statement.SetNotStatement(ss)
	case cloudprovider.WafStatementConditionNone:
		for _, s := range opts.Statements {
			statement = reverseConvertStatement(s)
			break
		}
	}
	rule.SetStatement(statement)
	rules = append(rules, rule)
	input.SetRules(rules)
	client, err := self.region.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	_, err = client.UpdateWebACL(input)
	if err != nil {
		return nil, errors.Wrapf(err, "UpdateWebACL")
	}
	ret := &sWafRule{waf: self, Rule: rule}
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
