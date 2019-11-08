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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/elbv2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SElbListenerRule struct {
	listener *SElbListener
	region   *SRegion

	Priority      string      `json:"Priority"`
	IsDefaultRule bool        `json:"IsDefault"`
	Actions       []Action    `json:"Actions"`
	RuleArn       string      `json:"RuleArn"`
	Conditions    []Condition `json:"Conditions"`
}

type Action struct {
	TargetGroupArn string `json:"TargetGroupArn"`
	Type           string `json:"Type"`
}

type Condition struct {
	Field                   string             `json:"field"`
	HTTPRequestMethodConfig *Config            `json:"httpRequestMethodConfig,omitempty"`
	Values                  []string           `json:"values"`
	SourceIPConfig          *Config            `json:"sourceIpConfig,omitempty"`
	QueryStringConfig       *QueryStringConfig `json:"queryStringConfig,omitempty"`
	HTTPHeaderConfig        *HTTPHeaderConfig  `json:"httpHeaderConfig,omitempty"`
	PathPatternConfig       *Config            `json:"pathPatternConfig,omitempty"`
	HostHeaderConfig        *Config            `json:"hostHeaderConfig,omitempty"`
}

type HTTPHeaderConfig struct {
	HTTPHeaderName string   `json:"HttpHeaderName"`
	Values         []string `json:"values"`
}

type Config struct {
	Values []string `json:"values"`
}

type QueryStringConfig struct {
	Values []Query `json:"values"`
}

type Query struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *SElbListenerRule) GetId() string {
	return self.RuleArn
}

func (self *SElbListenerRule) GetName() string {
	return self.RuleArn
}

func (self *SElbListenerRule) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbListenerRule) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbListenerRule) Refresh() error {
	rule, err := self.region.GetElbListenerRuleById(self.GetId())
	if err != nil {
		return err
	}

	err = jsonutils.Update(self, rule)
	if err != nil {
		return err
	}

	return nil
}

func (self *SElbListenerRule) IsDefault() bool {
	return self.IsDefaultRule
}

func (self *SElbListenerRule) IsEmulated() bool {
	return false
}

func (self *SElbListenerRule) GetMetadata() *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

func (self *SElbListenerRule) GetProjectId() string {
	return ""
}

func (self *SElbListenerRule) GetDomain() string {
	for _, condition := range self.Conditions {
		if condition.Field == "host-header" {
			return strings.Join(condition.Values, ",")
		}
	}

	return ""
}

func (self *SElbListenerRule) GetCondition() string {
	conditon, err := json.Marshal(self.Conditions)
	if err != nil {
		log.Errorf("GetCondition %s", err)
		return ""
	}

	return string(conditon)
}

func (self *SElbListenerRule) GetPath() string {
	for _, condition := range self.Conditions {
		if condition.Field == "path-pattern" {
			return strings.Join(condition.Values, ",")
		}
	}

	return ""
}

func (self *SElbListenerRule) GetBackendGroupId() string {
	for _, action := range self.Actions {
		if action.Type == "forward" {
			return action.TargetGroupArn
		}
	}

	return ""
}

func (self *SElbListenerRule) Delete() error {
	return self.region.DeleteElbListenerRule(self.GetId())
}

func (self *SRegion) DeleteElbListenerRule(ruleId string) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	params := &elbv2.DeleteRuleInput{}
	params.SetRuleArn(ruleId)
	_, err = client.DeleteRule(params)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) CreateElbListenerRule(listenerId string, config *cloudprovider.SLoadbalancerListenerRule) (*SElbListenerRule, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	forward := "forward"
	action := &elbv2.Action{
		TargetGroupArn: &config.BackendGroupID,
		Type:           &forward,
	}

	condtions, err := parseConditions(config.Condition)
	if err != nil {
		return nil, err
	}

	params := &elbv2.CreateRuleInput{}
	params.SetListenerArn(listenerId)
	params.SetActions([]*elbv2.Action{action})
	params.SetConditions(condtions)
	params.SetPriority(int64(1))
	ret, err := client.CreateRule(params)
	if err != nil {
		return nil, err
	}

	if len(ret.Rules) == 0 {
		return nil, errors.Wrap(fmt.Errorf("empty rules"), "Region.CreateElbListenerRule.len")
	}

	rule := SElbListenerRule{}
	err = unmarshalAwsOutput(ret.Rules[0], "", &rule)
	if err != nil {
		return nil, err
	}

	rule.region = self
	return &rule, nil
}

func parseConditions(conditions string) ([]*elbv2.RuleCondition, error) {
	obj, err := jsonutils.ParseString(conditions)
	if err != nil {
		return nil, err
	}

	conditionArray, ok := obj.(*jsonutils.JSONArray)
	if !ok {
		return nil, fmt.Errorf("parseConditions invalid condition fromat.")
	}

	ret := []*elbv2.RuleCondition{}
	cs := conditionArray.Value()
	for i := range cs {
		c, err := parseCondition(cs[i])
		if err != nil {
			return nil, err
		}

		ret = append(ret, c)
	}

	return ret, nil
}

func parseCondition(condition jsonutils.JSONObject) (*elbv2.RuleCondition, error) {
	conditionDict, ok := condition.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("parseCondition invalid condition fromat.")
	}

	dict := conditionDict.Value()
	field, ok := dict["field"]
	if !ok {
		return nil, fmt.Errorf("parseCondition invalid condition, missing field: %#v", condition)
	}

	f, _ := field.GetString()
	switch f {
	case "http-header":
		return parseHttpHeaderCondition(conditionDict)
	case "path-pattern":
		return parsePathPatternCondition(conditionDict)
	case "http-request-method":
		return parseRequestModthdCondition(conditionDict)
	case "host-header":
		return parseHostHeaderCondition(conditionDict)
	case "query-string":
		return parseQueryStringCondition(conditionDict)
	case "source-ip":
		return parseSourceIpCondition(conditionDict)
	default:
		return nil, fmt.Errorf("parseCondition invalid condition key %#v", field)
	}
}

func parseHttpHeaderCondition(conditon *jsonutils.JSONDict) (*elbv2.RuleCondition, error) {
	ret := &elbv2.RuleCondition{}
	ret.SetField("http-header")

	values, err := conditon.GetMap("httpHeaderConfig")
	if err != nil {
		return nil, err
	}

	name, ok := values["HttpHeaderName"]
	if !ok {
		return nil, fmt.Errorf("parseHttpHeaderCondition missing filed HttpHeaderName")
	}

	nameObj, ok := name.(*jsonutils.JSONString)
	if !ok {
		return nil, fmt.Errorf("parseHttpHeaderCondition missing invalid data %#v", name)
	}

	config := &elbv2.HttpHeaderConditionConfig{}
	config.SetHttpHeaderName(nameObj.Value())

	vs, ok := values["values"]
	if !ok {
		return nil, fmt.Errorf("parseHttpHeaderCondition missing filed values")
	}

	_vs, err := parseConditionStringArrayValues(vs)
	if err != nil {
		return nil, err
	}
	config.SetValues(_vs)
	ret.SetHttpHeaderConfig(config)
	return ret, nil
}

func parsePathPatternCondition(condition *jsonutils.JSONDict) (*elbv2.RuleCondition, error) {
	ret := &elbv2.RuleCondition{}
	ret.SetField("path-pattern")

	values, err := condition.GetMap("pathPatternConfig")
	if err != nil {
		return nil, err
	}

	config := &elbv2.PathPatternConditionConfig{}
	vs, ok := values["values"]
	if !ok {
		return nil, fmt.Errorf("parsePathPatternCondition missing filed values")
	}

	_vs, err := parseConditionStringArrayValues(vs)
	if err != nil {
		return nil, err
	}
	config.SetValues(_vs)
	ret.SetPathPatternConfig(config)
	return ret, nil

}

func parseRequestModthdCondition(condition *jsonutils.JSONDict) (*elbv2.RuleCondition, error) {
	ret := &elbv2.RuleCondition{}
	ret.SetField("http-request-method")

	values, err := condition.GetMap("httpRequestMethodConfig")
	if err != nil {
		return nil, err
	}

	config := &elbv2.HttpRequestMethodConditionConfig{}
	vs, ok := values["values"]
	if !ok {
		return nil, fmt.Errorf("parseRequestModthdCondition missing filed values")
	}

	_vs, err := parseConditionStringArrayValues(vs)
	if err != nil {
		return nil, err
	}
	config.SetValues(_vs)
	return ret, nil
}

func parseHostHeaderCondition(condition *jsonutils.JSONDict) (*elbv2.RuleCondition, error) {
	ret := &elbv2.RuleCondition{}
	ret.SetField("host-header")

	values, err := condition.GetMap("hostHeaderConfig")
	if err != nil {
		return nil, err
	}

	config := &elbv2.HostHeaderConditionConfig{}
	vs, ok := values["values"]
	if !ok {
		return nil, fmt.Errorf("parseHostHeaderCondition missing filed values")
	}

	_vs, err := parseConditionStringArrayValues(vs)
	if err != nil {
		return nil, err
	}
	config.SetValues(_vs)
	ret.SetHostHeaderConfig(config)
	return ret, nil
}

func parseQueryStringCondition(condition *jsonutils.JSONDict) (*elbv2.RuleCondition, error) {
	ret := &elbv2.RuleCondition{}
	ret.SetField("query-string")

	values, err := condition.GetMap("queryStringConfig")
	if err != nil {
		return nil, err
	}

	config := &elbv2.QueryStringConditionConfig{}
	vs, ok := values["values"]
	if !ok {
		return nil, fmt.Errorf("parseQueryStringCondition missing filed values")
	}

	_vs, err := parseConditionDictArrayValues(vs)
	if err != nil {
		return nil, err
	}
	config.SetValues(_vs)
	ret.SetQueryStringConfig(config)
	return ret, nil
}

func parseSourceIpCondition(condition *jsonutils.JSONDict) (*elbv2.RuleCondition, error) {
	ret := &elbv2.RuleCondition{}
	ret.SetField("source-ip")

	values, err := condition.GetMap("sourceIpConfig")
	if err != nil {
		return nil, err
	}

	config := &elbv2.SourceIpConditionConfig{}
	vs, ok := values["values"]
	if !ok {
		return nil, fmt.Errorf("parseSourceIpCondition missing filed values")
	}

	_vs, err := parseConditionStringArrayValues(vs)
	if err != nil {
		return nil, err
	}
	config.SetValues(_vs)
	return ret, nil
}

func parseConditionStringArrayValues(values jsonutils.JSONObject) ([]*string, error) {
	objs, ok := values.(*jsonutils.JSONArray)
	if !ok {
		return nil, fmt.Errorf("parseConditionStringArrayValues invalid values format, required array: %#v", values)
	}

	ret := []*string{}
	vs := objs.Value()
	for i := range vs {
		v, ok := vs[i].(*jsonutils.JSONString)
		if !ok {
			return nil, fmt.Errorf("parseConditionStringArrayValues invalid value, required string: %#v", v)
		}

		_v := v.Value()
		ret = append(ret, &_v)
	}

	return ret, nil
}

func parseConditionDictArrayValues(values jsonutils.JSONObject) ([]*elbv2.QueryStringKeyValuePair, error) {
	objs, ok := values.(*jsonutils.JSONArray)
	if !ok {
		return nil, fmt.Errorf("parseConditionDictArrayValues invalid values format, required array: %#v", values)
	}

	ret := []*elbv2.QueryStringKeyValuePair{}
	vs := objs.Value()
	for i := range vs {
		v, ok := vs[i].(*jsonutils.JSONDict)
		if !ok {
			return nil, fmt.Errorf("parseConditionDictArrayValues invalid value, required dict: %#v", v)
		}

		key, err := v.GetString("key")
		if err != nil {
			return nil, err
		}

		value, err := v.GetString("value")
		if err != nil {
			return nil, err
		}

		pair := &elbv2.QueryStringKeyValuePair{}
		pair.SetKey(key)
		pair.SetValue(value)
		ret = append(ret, pair)
	}

	return ret, nil
}
