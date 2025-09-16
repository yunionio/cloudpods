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

package aliyun

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAlbRule struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	AliyunTags
	albListener *SAlbListener

	RuleId         string          `json:"RuleId"`
	RuleName       string          `json:"RuleName"`
	RuleStatus     string          `json:"RuleStatus"`
	Priority       int             `json:"Priority"`
	ListenerId     string          `json:"ListenerId"`
	LoadBalancerId string          `json:"LoadBalancerId"`
	RuleConditions []RuleCondition `json:"RuleConditions"`
	RuleActions    []AlbRuleAction `json:"RuleActions"`
	Direction      string          `json:"Direction"`
	CreateTime     string          `json:"CreateTime"`
	RegionId       string          `json:"RegionId"`
}

type RuleCondition struct {
	Type                     string                   `json:"Type"`
	HostConfig               HostConfig               `json:"HostConfig"`
	PathConfig               PathConfig               `json:"PathConfig"`
	MethodConfig             MethodConfig             `json:"MethodConfig"`
	QueryStringConfig        QueryStringConfig        `json:"QueryStringConfig"`
	HeaderConfig             HeaderConfig             `json:"HeaderConfig"`
	CookieConfig             CookieConfig             `json:"CookieConfig"`
	SourceIpConfig           SourceIpConfig           `json:"SourceIpConfig"`
	ResponseHeaderConfig     ResponseHeaderConfig     `json:"ResponseHeaderConfig"`
	ResponseStatusCodeConfig ResponseStatusCodeConfig `json:"ResponseStatusCodeConfig"`
}

type AlbRuleAction struct {
	Type               string             `json:"Type"`
	Order              int                `json:"Order"`
	ForwardGroupConfig ForwardGroupConfig `json:"ForwardGroupConfig"`
}

type HostConfig struct {
	Values []string `json:"Values"`
}

type PathConfig struct {
	Values []string `json:"Values"`
}

type MethodConfig struct {
	Values []string `json:"Values"`
}

type QueryStringConfig struct {
	Values []QueryStringValue `json:"Values"`
}

type QueryStringValue struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type HeaderConfig struct {
	Key    string   `json:"Key"`
	Values []string `json:"Values"`
}

type CookieConfig struct {
	Values []CookieValue `json:"Values"`
}

type CookieValue struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type SourceIpConfig struct {
	Values []string `json:"Values"`
}

type ResponseHeaderConfig struct {
	Key    string   `json:"Key"`
	Values []string `json:"Values"`
}

type ResponseStatusCodeConfig struct {
	Values []string `json:"Values"`
}

func (rule *SAlbRule) GetName() string {
	return rule.RuleName
}

func (rule *SAlbRule) GetId() string {
	return rule.RuleId
}

func (rule *SAlbRule) GetGlobalId() string {
	return rule.RuleId
}

func (rule *SAlbRule) GetStatus() string {
	switch rule.RuleStatus {
	case "Available":
		return api.LB_STATUS_ENABLED
	case "Configuring":
		return api.LB_STATUS_UNKNOWN
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (rule *SAlbRule) IsDefault() bool {
	return false
}

func (rule *SAlbRule) IsEmulated() bool {
	return false
}

func (rule *SAlbRule) Refresh() error {
	r, err := rule.albListener.alb.region.GetAlbRule(rule.RuleId)
	if err != nil {
		return err
	}
	return jsonutils.Update(rule, r)
}

func (rule *SAlbRule) GetCondition() string {
	return ""
}

func (rule *SAlbRule) GetDomain() string {
	for _, condition := range rule.RuleConditions {
		if condition.Type == "Host" && len(condition.HostConfig.Values) > 0 {
			return condition.HostConfig.Values[0]
		}
	}
	return ""
}

func (rule *SAlbRule) GetPath() string {
	for _, condition := range rule.RuleConditions {
		if condition.Type == "Path" && len(condition.PathConfig.Values) > 0 {
			return condition.PathConfig.Values[0]
		}
	}
	return ""
}

func (rule *SAlbRule) GetProjectId() string {
	return rule.albListener.GetProjectId()
}

func (rule *SAlbRule) GetBackendGroupId() string {
	for _, action := range rule.RuleActions {
		if action.Type == "ForwardGroup" && len(action.ForwardGroupConfig.ServerGroupTuples) > 0 {
			return action.ForwardGroupConfig.ServerGroupTuples[0].ServerGroupId
		}
	}
	return ""
}

func (rule *SAlbRule) GetBackendGroups() ([]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (rule *SAlbRule) GetRedirectPool() (cloudprovider.SRedirectPool, error) {
	return cloudprovider.SRedirectPool{}, cloudprovider.ErrNotImplemented
}

func (rule *SAlbRule) Delete(ctx context.Context) error {
	return rule.albListener.alb.region.DeleteAlbRule(rule.RuleId)
}

func (rule *SAlbRule) Update(ctx context.Context, opts *cloudprovider.SLoadbalancerListenerRule) error {
	return cloudprovider.ErrNotImplemented
}

// region methods
func (region *SRegion) GetAlbRules(listenerId string) ([]SAlbRule, error) {
	params := map[string]string{
		"RegionId":      region.RegionId,
		"ListenerIds.1": listenerId,
	}

	body, err := region.albRequest("ListRules", params)
	if err != nil {
		return nil, err
	}

	rules := []SAlbRule{}
	err = body.Unmarshal(&rules, "Rules")
	if err != nil {
		return nil, err
	}

	return rules, nil
}

func (region *SRegion) GetAlbRule(ruleId string) (*SAlbRule, error) {
	rules, err := region.GetAlbRules("")
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if rule.RuleId == ruleId {
			return &rule, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) CreateAlbRule(listenerId string, rule *cloudprovider.SLoadbalancerListenerRule) (*SAlbRule, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
		"RuleName":   rule.Name,
		"Priority":   "100",
	}

	// 构建规则条件
	conditions := jsonutils.NewArray()
	if len(rule.Domain) > 0 {
		condition := jsonutils.Marshal(map[string]interface{}{
			"Type": "Host",
			"HostConfig": map[string]interface{}{
				"Values": []string{rule.Domain},
			},
		})
		conditions.Add(condition)
	}

	if len(rule.Path) > 0 {
		condition := jsonutils.Marshal(map[string]interface{}{
			"Type": "Path",
			"PathConfig": map[string]interface{}{
				"Values": []string{rule.Path},
			},
		})
		conditions.Add(condition)
	}

	params["RuleConditions"] = conditions.String()

	// 构建规则动作
	actions := jsonutils.NewArray()
	if len(rule.BackendGroupId) > 0 {
		action := jsonutils.Marshal(map[string]interface{}{
			"Type":  "ForwardGroup",
			"Order": 1,
			"ForwardGroupConfig": map[string]interface{}{
				"ServerGroupTuples": []map[string]interface{}{
					{
						"ServerGroupId": rule.BackendGroupId,
						"Weight":        100,
					},
				},
			},
		})
		actions.Add(action)
	}

	params["RuleActions"] = actions.String()

	body, err := region.albRequest("CreateRule", params)
	if err != nil {
		return nil, err
	}

	ruleId, err := body.GetString("RuleId")
	if err != nil {
		return nil, err
	}

	return region.GetAlbRule(ruleId)
}

func (region *SRegion) DeleteAlbRule(ruleId string) error {
	params := map[string]string{
		"RegionId": region.RegionId,
		"RuleId":   ruleId,
	}

	_, err := region.albRequest("DeleteRule", params)
	return err
}
