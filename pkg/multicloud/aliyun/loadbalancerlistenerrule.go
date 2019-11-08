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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerListenerRule struct {
	httpListener  *SLoadbalancerHTTPListener
	httpsListener *SLoadbalancerHTTPSListener

	Domain         string `json:"Domain"`
	ListenerSync   string
	RuleId         string
	RuleName       string `json:"RuleName"`
	Url            string `json:"Url"`
	VServerGroupId string `json:"VServerGroupId"`
}

func (lbr *SLoadbalancerListenerRule) GetName() string {
	return lbr.RuleName
}

func (lbr *SLoadbalancerListenerRule) GetId() string {
	return lbr.RuleId
}

func (lbr *SLoadbalancerListenerRule) GetGlobalId() string {
	return lbr.RuleId
}

func (lbr *SLoadbalancerListenerRule) GetStatus() string {
	return ""
}

func (lbr *SLoadbalancerListenerRule) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLoadbalancerListenerRule) IsDefault() bool {
	return false
}

func (lbr *SLoadbalancerListenerRule) IsEmulated() bool {
	return false
}

func (lbr *SLoadbalancerListenerRule) getRegion() *SRegion {
	if lbr.httpListener != nil {
		return lbr.httpListener.lb.region
	} else if lbr.httpsListener != nil {
		return lbr.httpsListener.lb.region
	}
	return nil
}

func (lbr *SLoadbalancerListenerRule) Refresh() error {
	region := lbr.getRegion()
	if region == nil {
		return fmt.Errorf("failed to find listener for rule %s", lbr.RuleName)
	}
	rule, err := region.GetLoadbalancerListenerRule(lbr.RuleId)
	if err != nil {
		return err
	}
	return jsonutils.Update(lbr, rule)
}

func (lbr *SLoadbalancerListenerRule) GetCondition() string {
	return ""
}

func (lbr *SLoadbalancerListenerRule) GetDomain() string {
	return lbr.Domain
}

func (lbr *SLoadbalancerListenerRule) GetPath() string {
	return lbr.Url
}

func (lbr *SLoadbalancerListenerRule) GetProjectId() string {
	return ""
}

func (lbr *SLoadbalancerListenerRule) GetBackendGroupId() string {
	return lbr.VServerGroupId
}

func (region *SRegion) GetLoadbalancerListenerRules(loadbalancerId string, listenerPort int) ([]SLoadbalancerListenerRule, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	body, err := region.lbRequest("DescribeRules", params)
	if err != nil {
		return nil, err
	}
	rules := []SLoadbalancerListenerRule{}
	return rules, body.Unmarshal(&rules, "Rules", "Rule")
}

func (lbr *SLoadbalancerListenerRule) Delete() error {
	if lbr.httpListener != nil {
		return lbr.httpListener.lb.region.DeleteLoadbalancerListenerRule(lbr.RuleId)
	}
	if lbr.httpsListener != nil {
		return lbr.httpsListener.lb.region.DeleteLoadbalancerListenerRule(lbr.RuleId)
	}
	return fmt.Errorf("failed to find listener for listener rule %s", lbr.RuleName)
}

func (region *SRegion) DeleteLoadbalancerListenerRule(ruleId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["RuleIds"] = fmt.Sprintf(`["%s"]`, ruleId)
	_, err := region.lbRequest("DeleteRules", params)
	return err
}

func (region *SRegion) CreateLoadbalancerListenerRule(listenerPort int, loadbalancerId string, _rule *SLoadbalancerListenerRule) (*SLoadbalancerListenerRule, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	params["LoadBalancerId"] = loadbalancerId
	_rules := jsonutils.NewArray()
	_rules.Add(jsonutils.Marshal(_rule))
	params["RuleList"] = _rules.String()
	body, err := region.lbRequest("CreateRules", params)
	if err != nil {
		return nil, err
	}
	rules := []SLoadbalancerListenerRule{}
	if err := body.Unmarshal(&rules, "Rules", "Rule"); err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if rule.RuleName == _rule.RuleName {
			return region.GetLoadbalancerListenerRule(rule.RuleId)
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetLoadbalancerListenerRule(ruleId string) (*SLoadbalancerListenerRule, error) {
	if len(ruleId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["RuleId"] = ruleId
	body, err := region.lbRequest("DescribeRuleAttribute", params)
	if err != nil {
		return nil, err
	}
	rule := &SLoadbalancerListenerRule{RuleId: ruleId}
	return rule, body.Unmarshal(rule)
}
