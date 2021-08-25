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

package policy

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	common_policy "yunion.io/x/onecloud/pkg/cloudcommon/policy"
)

var (
	computeSystemResources = []string{
		"zones",
		"cloudregions",
		"serverskus",
		"cachedimages",
		"dynamicschedtags",
		"baremetalagents",
		"schedpolicies",
		"dnsrecords",
		"metadatas",
		"loadbalancerclusters",
		"loadbalanceragents",
		// "reservedips",
		"policy_definitions",
		"schedtags",
	}
	computeDomainResources = []string{
		"cloudaccounts",
		"cloudproviders",
		"recyclebins",
		// migrate system resources to domain resources
		"hosts",
		"baremetalnetworks",
		"hoststorages",
		"hostwires",
		"isolated-devices",
		"vpcs",
		"storages",
		"wires",
		"globalvpcs",
		"route_tables",
		"networkinterfaces",
		"natgateways",
		"natsentries",
		"natdentries",
		"policy_assignments",
		"proxysettings",
		"project_mappings",
		"waf_instances",
		"waf_rules",
		"waf_rule_groups",
		"waf_ipsets",
		"waf_regexsets",
		"cdn_domains",
	}
	computeUserResources = []string{
		"keypairs",
	}
)

func init() {
	common_policy.RegisterSystemResources(api.SERVICE_TYPE, computeSystemResources)
	common_policy.RegisterDomainResources(api.SERVICE_TYPE, computeDomainResources)
	common_policy.RegisterUserResources(api.SERVICE_TYPE, computeUserResources)
}
