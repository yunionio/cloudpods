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

import "net/url"

type SFrontDoorProperties struct {
	ResourceState     string
	ProvisioningState string
	PolicySettings    struct {
		EnabledState                  string
		Mode                          string
		RedirectUrl                   string
		CustomBlockResponseStatusCode int
		CustomBlockResponseBody       string
		RequestBodyCheck              string
	}
	CustomRules struct {
		Rules []struct{}
	}
	ManagedRules struct {
		ManagedRuleSets []struct {
			RuleSetType        string
			RuleSetVersion     string
			RuleSetAction      string
			RuleGroupOverrides []struct {
			}
			Exclusions []struct{}
		}
	}
	FrontendEndpointLinks []struct{}
	RoutingRuleLinks      []struct{}
	SecurityPolicyLinks   []struct{}
}

type SFrontDoorWaf struct {
	Id       string
	Name     string
	Type     string
	Tags     map[string]string
	Location string
	Sku      struct {
		Name string
	}
	Properties SFrontDoorProperties
}

func (self *SRegion) ListFrontDoorWafs(resGroup string) ([]SFrontDoorWaf, error) {
	params := url.Values{}
	params.Set("resourceGroups", resGroup)
	ret := []SFrontDoorWaf{}
	err := self.list("Microsoft.Network/frontdoorWebApplicationFirewallPolicies", params, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
