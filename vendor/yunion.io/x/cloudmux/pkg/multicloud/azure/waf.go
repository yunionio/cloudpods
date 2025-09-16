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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SMatchvariable struct {
	Variablename string `json:"variableName"`
	Selector     string `json:"selector"`
}

type SMatchcondition struct {
	Matchvariables   []SMatchvariable `json:"matchVariables"`
	Operator         string           `json:"operator"`
	Negationconditon bool             `json:"negationConditon"`
	Matchvalues      []string         `json:"matchValues"`
	Transforms       []string         `json:"transforms"`
}

type CustomRule struct {
	waf *SAppGatewayWaf

	Name     string `json:"name"`
	Priority int    `json:"priority"`
	Ruletype string `json:"ruleType"`
	//RateLimitThreshold *int              `json:"rateLimitThreshold"`
	Matchconditions []SMatchcondition `json:"matchConditions"`
	Action          string            `json:"action"`
}

func (self *CustomRule) GetName() string {
	return self.Name
}

func (self *CustomRule) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.waf.GetGlobalId(), self.GetName())
}

func (self *CustomRule) GetDesc() string {
	return ""
}

func (self *CustomRule) GetPriority() int {
	return self.Priority
}

func (self *CustomRule) Delete() error {
	rules := []CustomRule{}
	for _, rule := range self.waf.Properties.Customrules {
		if rule.Name != self.Name {
			rules = append(rules, rule)
		}
	}
	self.waf.Properties.Customrules = rules
	return self.waf.region.update(jsonutils.Marshal(self.waf), nil)
}

func wafMatchFieldAndKeyLocal2Cloud(opts cloudprovider.SWafStatement) ([]SMatchvariable, error) {
	ret := []SMatchvariable{}
	switch opts.MatchField {
	case cloudprovider.WafMatchFieldQuery:
		ret = append(ret, SMatchvariable{
			Variablename: "QueryString",
		})
	case cloudprovider.WafMatchFieldMethod:
		ret = append(ret, SMatchvariable{
			Variablename: "RequestMethod",
		})
	case cloudprovider.WafMatchFiledUriPath:
		ret = append(ret, SMatchvariable{
			Variablename: "RequestUri",
		})
	case cloudprovider.WafMatchFiledHeader:
		ret = append(ret, SMatchvariable{
			Variablename: "RequestHeaders",
			Selector:     opts.MatchFieldKey,
		})
	case cloudprovider.WafMatchFiledPostArgs:
		ret = append(ret, SMatchvariable{
			Variablename: "PostArgs",
			Selector:     opts.MatchFieldKey,
		})
	case cloudprovider.WafMatchFieldBody:
		ret = append(ret, SMatchvariable{
			Variablename: "RequestBody",
		})
	case cloudprovider.WafMatchFiledCookie:
		ret = append(ret, SMatchvariable{
			Variablename: "RequestCookies",
			Selector:     opts.MatchFieldKey,
		})
	default:
		return ret, fmt.Errorf("unsupported match filed %s", opts.MatchField)
	}
	return ret, nil
}

func wafMatchFieldAndKeyCloud2Local(v SMatchvariable) (cloudprovider.TWafMatchField, string, error) {
	switch v.Variablename {
	case "QueryString":
		return cloudprovider.WafMatchFieldQuery, v.Selector, nil
	case "RequestMethod":
		return cloudprovider.WafMatchFieldMethod, "", nil
	case "RequestUri":
		return cloudprovider.WafMatchFiledUriPath, "", nil
	case "RequestHeaders":
		return cloudprovider.WafMatchFiledHeader, v.Selector, nil
	case "PostArgs":
		return cloudprovider.WafMatchFiledPostArgs, v.Selector, nil
	case "RequestBody":
		return cloudprovider.WafMatchFieldBody, "", nil
	case "RequestCookies":
		return cloudprovider.WafMatchFiledCookie, v.Selector, nil
	case "RemoteAddr":
		return cloudprovider.WafMatchFiledHeader, v.Selector, nil
	default:
		return "", "", fmt.Errorf("invalid variablename %s", v.Variablename)
	}
}

func wafStatementLocal2Cloud(opts cloudprovider.SWafStatement) (SMatchcondition, error) {
	ret := SMatchcondition{}
	if opts.Transformations != nil {
		for _, tran := range *opts.Transformations {
			ret.Transforms = append(ret.Transforms, string(tran))
		}
	}
	if opts.MatchFieldValues != nil {
		ret.Matchvalues = *opts.MatchFieldValues
	}
	ret.Negationconditon = opts.Negation
	ret.Operator = string(opts.Operator)
	var err error
	switch opts.Type {
	case cloudprovider.WafStatementTypeIPSet:
		ret.Operator = "IPMatch"
		ret.Matchvariables = []SMatchvariable{
			SMatchvariable{
				Variablename: "RemoteAddr",
			},
		}
	case cloudprovider.WafStatementTypeGeoMatch:
		ret.Operator = "GeoMatch"
		if len(opts.ForwardedIPHeader) == 0 {
			ret.Matchvariables = []SMatchvariable{
				SMatchvariable{
					Variablename: "RemoteAddr",
				},
			}
		} else {
			ret.Matchvariables = []SMatchvariable{
				SMatchvariable{
					Variablename: "RequestHeaders",
					Selector:     opts.ForwardedIPHeader,
				},
			}
		}
	case cloudprovider.WafStatementTypeSize:
		switch opts.Operator {
		case "LT":
			ret.Operator = "LessThan"
		case "LE":
			ret.Operator = "LessThanOrEqual"
		case "GT":
			ret.Operator = "GreaterThan"
		default:
			return ret, fmt.Errorf("invalid operator %s for %s", opts.Operator, opts.Type)
		}
		ret.Matchvariables, err = wafMatchFieldAndKeyLocal2Cloud(opts)
		if err != nil {
			return ret, errors.Wrapf(err, "wafMatchFieldAndKeyLocal2Cloud")
		}
	case cloudprovider.WafStatementTypeByteMatch:
		switch opts.Operator {
		case "Contains", "EndsWith", "Regex":
		case "StartsWith":
			ret.Operator = "BeginsWith"
		case "Exactly":
			ret.Operator = "Equal"
		default:
			return ret, fmt.Errorf("invalid operator %s for %s", opts.Operator, opts.Type)
		}
		ret.Matchvariables, err = wafMatchFieldAndKeyLocal2Cloud(opts)
		if err != nil {
			return ret, errors.Wrapf(err, "wafMatchFieldAndKeyLocal2Cloud")
		}
	}
	return ret, nil
}

func wafRuleLocal2Cloud(opts *cloudprovider.SWafRule) (*CustomRule, error) {
	ret := &CustomRule{}
	ret.Name = opts.Name
	ret.Priority = opts.Priority
	ret.Ruletype = "MatchRule"
	ret.Matchconditions = []SMatchcondition{}
	for _, s := range opts.Statements {
		cds, err := wafStatementLocal2Cloud(s)
		if err != nil {
			return nil, errors.Wrapf(err, "wafStatementLocal2Cloud")
		}
		ret.Matchconditions = append(ret.Matchconditions, cds)
	}
	ret.Action = "Block"
	if opts.Action != nil {
		ret.Action = string(opts.Action.Action)
	}
	return ret, nil
}

func (self *CustomRule) Update(opts *cloudprovider.SWafRule) error {
	rules := []CustomRule{}
	for _, rule := range self.waf.Properties.Customrules {
		if rule.Name != self.Name {
			rules = append(rules, rule)
		} else {
			rule, err := wafRuleLocal2Cloud(opts)
			if err != nil {
				return errors.Wrapf(err, "wafRuleLocal2Cloud")
			}
			rules = append(rules, *rule)
		}
	}
	self.waf.Properties.Customrules = rules
	return self.waf.region.update(jsonutils.Marshal(self.waf), nil)
}

func (self *CustomRule) GetAction() *cloudprovider.DefaultAction {
	return &cloudprovider.DefaultAction{
		Action: cloudprovider.TWafAction(self.Action),
	}
}

func (self *CustomRule) GetStatementCondition() cloudprovider.TWafStatementCondition {
	return cloudprovider.WafStatementConditionAnd
}

func (self *CustomRule) GetStatements() ([]cloudprovider.SWafStatement, error) {
	ret := []cloudprovider.SWafStatement{}
	for _, condition := range self.Matchconditions {
		trans := cloudprovider.TextTransformations{}
		for _, tran := range condition.Transforms {
			trans = append(trans, cloudprovider.TWafTextTransformation(tran))
		}
		values := cloudprovider.TWafMatchFieldValues(condition.Matchvalues)
		statement := cloudprovider.SWafStatement{
			Negation:         condition.Negationconditon,
			Transformations:  &trans,
			MatchFieldValues: &values,
		}
		switch condition.Operator {
		case "IPMatch":
			statement.Type = cloudprovider.WafStatementTypeIPSet
		case "GeoMatch":
			statement.Type = cloudprovider.WafStatementTypeGeoMatch
		case "LessThan":
			statement.Type = cloudprovider.WafStatementTypeSize
			statement.Operator = cloudprovider.WafOperatorLT
		case "LessThanOrEqual":
			statement.Type = cloudprovider.WafStatementTypeSize
			statement.Operator = cloudprovider.WafOperatorLE
		case "GreaterThan":
			statement.Type = cloudprovider.WafStatementTypeSize
			statement.Operator = cloudprovider.WafOperatorGT
		case "BeginsWith":
			statement.Type = cloudprovider.WafStatementTypeByteMatch
			statement.Operator = cloudprovider.WafOperatorStartsWith
		case "Contains", "EndsWith", "Regex":
			statement.Type = cloudprovider.WafStatementTypeByteMatch
			statement.Operator = cloudprovider.TWafOperator(condition.Operator)
		case "Equal":
			statement.Type = cloudprovider.WafStatementTypeByteMatch
			statement.Operator = cloudprovider.WafOperatorExactly
		default:
			statement.Type = cloudprovider.WafStatementTypeByteMatch
		}

		var err error
		for _, v := range condition.Matchvariables {
			statement.MatchField, statement.MatchFieldKey, err = wafMatchFieldAndKeyCloud2Local(v)
			if err != nil {
				log.Errorf("wafMatchFieldAndKeyCloud2Local %s error: %v", v, err)
				continue
			}
			ret = append(ret, statement)
		}
	}
	return ret, nil
}

type ManagedRule struct {
	Rulesettype    string `json:"ruleSetType"`
	Rulesetversion string `json:"ruleSetVersion"`
}

type ManagedRules struct {
	waf             *SAppGatewayWaf
	Managedrulesets []ManagedRule `json:"managedRuleSets"`
}

func (self *ManagedRules) GetName() string {
	return fmt.Sprintf("%s Managed rules", self.waf.GetName())
}

func (self *ManagedRules) GetGlobalId() string {
	return self.waf.GetGlobalId()
}

func (self *ManagedRules) GetDesc() string {
	return ""
}

func (self *ManagedRules) GetPriority() int {
	return 0
}

func (self *ManagedRules) GetAction() *cloudprovider.DefaultAction {
	return nil
}

func (self *ManagedRules) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *ManagedRules) Update(opts *cloudprovider.SWafRule) error {
	rules := []ManagedRule{}
	for _, s := range opts.Statements {
		if len(s.ManagedRuleGroupName) == 0 {
			return fmt.Errorf("missing managed rule group name")
		}
		names := strings.Split(s.ManagedRuleGroupName, "_")
		if len(names) != 2 {
			return fmt.Errorf("invalid managed rule group name %s", s.ManagedRuleGroupName)
		}
		rules = append(rules, ManagedRule{
			Rulesettype:    names[0],
			Rulesetversion: names[1],
		})
	}
	if len(rules) == 0 {
		return fmt.Errorf("missing statements")
	}
	self.waf.Properties.Managedrules = ManagedRules{
		Managedrulesets: rules,
	}
	return self.waf.region.update(jsonutils.Marshal(self.waf), nil)
}

func (self *ManagedRules) GetStatementCondition() cloudprovider.TWafStatementCondition {
	return cloudprovider.WafStatementConditionAnd
}

func (self *ManagedRules) GetStatements() ([]cloudprovider.SWafStatement, error) {
	ret := []cloudprovider.SWafStatement{}
	for i := range self.Managedrulesets {
		ruleGroupName := fmt.Sprintf("%s_%s", self.Managedrulesets[i].Rulesettype, self.Managedrulesets[i].Rulesetversion)
		ret = append(ret, cloudprovider.SWafStatement{
			ManagedRuleGroupName: ruleGroupName,
			Type:                 cloudprovider.WafStatementTypeManagedRuleGroup,
			RuleGroupId:          ruleGroupName,
		})
	}
	return ret, nil
}

type SAppGatewayWaf struct {
	multicloud.SResourceBase
	AzureTags
	region *SRegion

	Name       string `json:"name"`
	ID         string `json:"id"`
	Type       string `json:"type"`
	Location   string `json:"location"`
	Properties struct {
		ApplicationGateways []SApplicationGateway
		HttpListeners       []struct {
			Id string
		}
		PathBasedRules []struct {
			Id string
		}
		Resourcestate     string `json:"resourceState"`
		Provisioningstate string `json:"provisioningState"`
		Policysettings    struct {
			State                  string `json:"state"`
			Mode                   string `json:"mode"`
			Maxrequestbodysizeinkb int    `json:"maxRequestBodySizeInKb"`
			Fileuploadlimitinmb    int    `json:"fileUploadLimitInMb"`
			Requestbodycheck       bool   `json:"requestBodyCheck"`
		} `json:"policySettings"`
		Customrules  []CustomRule `json:"customRules"`
		Managedrules ManagedRules `json:"managedRules"`
	} `json:"properties"`
}

func (self *SAppGatewayWaf) GetEnabled() bool {
	return self.Properties.Policysettings.State == "Enabled"
}

func (self *SAppGatewayWaf) GetName() string {
	return self.Name
}

func (self *SAppGatewayWaf) GetId() string {
	return self.ID
}

func (self *SAppGatewayWaf) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SAppGatewayWaf) Delete() error {
	return self.region.del(self.ID)
}

func (self *SAppGatewayWaf) GetWafType() cloudprovider.TWafType {
	return cloudprovider.WafTypeAppGateway
}

func (self *SAppGatewayWaf) GetIsAccessProduct() bool {
	return true
}

func (self *SAppGatewayWaf) GetAccessHeaders() []string {
	return []string{}
}

func (self *SAppGatewayWaf) GetHttpPorts() []int {
	return []int{}
}

func (self *SAppGatewayWaf) GetHttpsPorts() []int {
	return []int{}
}

func (self *SAppGatewayWaf) GetCname() string {
	return ""
}

func (self *SAppGatewayWaf) GetUpstreamScheme() string {
	return ""
}

func (self *SAppGatewayWaf) GetCertId() string {
	return ""
}

func (self *SAppGatewayWaf) GetCertName() string {
	return ""
}

func (self *SAppGatewayWaf) GetUpstreamPort() int {
	return 0
}

func (self *SAppGatewayWaf) GetSourceIps() []string {
	return []string{}
}

func (self *SAppGatewayWaf) GetCcList() []string {
	return []string{}
}

func (self *SAppGatewayWaf) AddRule(opts *cloudprovider.SWafRule) (cloudprovider.ICloudWafRule, error) {
	rule, err := wafRuleLocal2Cloud(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "wafRuleLocal2Cloud")
	}
	rule.waf = self
	self.Properties.Customrules = append(self.Properties.Customrules, *rule)
	err = self.region.update(jsonutils.Marshal(self), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "update")
	}
	return rule, nil
}

func (self *SAppGatewayWaf) GetStatus() string {
	switch self.Properties.Provisioningstate {
	case "Deleting":
		return api.WAF_STATUS_DELETING
	case "Failed":
		return api.WAF_STATUS_CREATE_FAILED
	case "Succeeded":
		return api.WAF_STATUS_AVAILABLE
	case "Updating":
		return api.WAF_STATUS_UPDATING
	default:
		return self.Properties.Provisioningstate
	}
}

func (self *SAppGatewayWaf) GetRules() ([]cloudprovider.ICloudWafRule, error) {
	ret := []cloudprovider.ICloudWafRule{}
	for i := range self.Properties.Customrules {
		self.Properties.Customrules[i].waf = self
		ret = append(ret, &self.Properties.Customrules[i])
	}
	self.Properties.Managedrules.waf = self
	ret = append(ret, &self.Properties.Managedrules)
	return ret, nil
}

func (self *SAppGatewayWaf) Refresh() error {
	waf, err := self.region.GetAppGatewayWaf(self.ID)
	if err != nil {
		return errors.Wrapf(err, "GetAppGatewayWa")
	}
	return jsonutils.Update(self, waf)
}

func (self *SAppGatewayWaf) GetDefaultAction() *cloudprovider.DefaultAction {
	return &cloudprovider.DefaultAction{
		Action: cloudprovider.TWafAction(self.Properties.Policysettings.Mode),
	}
}

func (self *SRegion) ListAppWafs() ([]SAppGatewayWaf, error) {
	ret := []SAppGatewayWaf{}
	err := self.list("Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies", url.Values{}, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return ret, nil
}

type SAppWafRuleGroup struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Type       string `json:"type"`
	Properties struct {
		Provisioningstate string `json:"provisioningState"`
		Rulesettype       string `json:"ruleSetType"`
		Rulesetversion    string `json:"ruleSetVersion"`
		Rulegroups        []struct {
			Rulegroupname string `json:"ruleGroupName"`
			Description   string `json:"description"`
			Rules         []struct {
				Ruleid      int    `json:"ruleId"`
				Description string `json:"description"`
			} `json:"rules"`
		} `json:"ruleGroups"`
	} `json:"properties"`
}

func (self *SRegion) CreateICloudWafInstance(opts *cloudprovider.WafCreateOptions) (cloudprovider.ICloudWafInstance, error) {
	switch opts.Type {
	case cloudprovider.WafTypeAppGateway:
		return self.CreateAppWafInstance(opts.Name, opts.DefaultAction)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNoSuchProvder, "invalid waf type %s", opts.Type)
	}
}

func (self *SRegion) GetICloudWafInstanceById(id string) (cloudprovider.ICloudWafInstance, error) {
	if strings.Contains(id, "microsoft.network/applicationgatewaywebapplicationfirewallpolicies") {
		return self.GetAppGatewayWaf(id)
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, id)
}

func (self *SRegion) CreateAppWafInstance(name string, action *cloudprovider.DefaultAction) (*SAppGatewayWaf, error) {
	mode := cloudprovider.WafActionDetection
	if action != nil {
		switch action.Action {
		case cloudprovider.WafActionDetection, cloudprovider.WafActionPrevention:
			mode = action.Action
		default:
			return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "invalid action %s", action.Action)
		}
	}
	params := map[string]interface{}{
		"Type":     "Microsoft.Network/applicationGatewayWebApplicationFirewallPolicies",
		"Name":     name,
		"Location": self.Name,
		"properties": map[string]interface{}{
			"customRules": []string{},
			"policySettings": map[string]interface{}{
				"fileUploadLimitInMb":    100,
				"maxRequestBodySizeInKb": 128,
				"mode":                   mode,
				"requestBodyCheck":       true,
				"state":                  "Enabled",
			},
			"managedRules": map[string]interface{}{
				"exclusions": []string{},
				"managedRuleSets": []map[string]interface{}{
					map[string]interface{}{
						"ruleSetType":        "OWASP",
						"ruleSetVersion":     "3.1",
						"ruleGroupOverrides": []string{},
					},
				},
			},
		},
	}
	ret := &SAppGatewayWaf{region: self}
	err := self.create("", jsonutils.Marshal(params), ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetAppGatewayWaf(id string) (*SAppGatewayWaf, error) {
	res := &SAppGatewayWaf{region: self}
	return res, self.get(id, nil, &res)
}

func (self *SRegion) ListAppWafManagedRuleGroup() ([]SAppWafRuleGroup, error) {
	ret := []SAppWafRuleGroup{}
	err := self.list("Microsoft.Network/applicationGatewayAvailableWafRuleSets", url.Values{}, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return ret, nil
}

func (self *SRegion) GetICloudWafInstances() ([]cloudprovider.ICloudWafInstance, error) {
	wafs, err := self.ListAppWafs()
	if err != nil {
		return nil, errors.Wrapf(err, "ListAppWafs")
	}
	ret := []cloudprovider.ICloudWafInstance{}
	for i := range wafs {
		wafs[i].region = self
		ret = append(ret, &wafs[i])
	}
	return ret, nil
}

func (self *SAppGatewayWaf) GetCloudResources() ([]cloudprovider.SCloudResource, error) {
	ret := []cloudprovider.SCloudResource{}
	for _, ag := range self.Properties.ApplicationGateways {
		ret = append(ret, cloudprovider.SCloudResource{
			Id:            ag.Id,
			Name:          ag.Id[strings.LastIndex(ag.Id, "/")+1:],
			Type:          "Application Gateway",
			CanDissociate: true,
		})
	}
	for _, lis := range self.Properties.HttpListeners {
		ret = append(ret, cloudprovider.SCloudResource{
			Id:            lis.Id,
			Name:          lis.Id[strings.LastIndex(lis.Id, "/")+1:],
			Type:          "HTTP Listener",
			CanDissociate: true,
		})
	}
	for _, route := range self.Properties.PathBasedRules {
		ret = append(ret, cloudprovider.SCloudResource{
			Id:            route.Id,
			Name:          route.Id[strings.LastIndex(route.Id, "/")+1:],
			Type:          "Route Path",
			CanDissociate: true,
		})

	}
	return ret, nil
}

func (self *SAppGatewayWaf) SetTags(tags map[string]string, replace bool) error {
	if !replace {
		for k, v := range self.Tags {
			if _, ok := tags[k]; !ok {
				tags[k] = v
			}
		}
	}
	_, err := self.region.client.SetTags(self.ID, tags)
	if err != nil {
		return errors.Wrapf(err, "SetTags")
	}
	return nil
}
