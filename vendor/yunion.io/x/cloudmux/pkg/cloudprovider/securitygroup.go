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

package cloudprovider

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/util/secrules"
)

type SecurityGroupReference struct {
	Id   string
	Name string
}

type SecurityGroupCreateInput struct {
	Name      string
	Desc      string
	VpcId     string
	ProjectId string

	Tags map[string]string
}

type SecurityGroupRuleCreateOptions struct {
	Desc      string
	Priority  int
	Protocol  string
	Ports     string
	Direction secrules.TSecurityRuleDirection
	CIDR      string
	Action    secrules.TSecurityRuleAction
}

type SecurityGroupRuleUpdateOptions struct {
	CIDR     string
	Action   secrules.TSecurityRuleAction
	Desc     string
	Ports    string
	Protocol string
	Priority int
}

func (rule *SecurityGroupRuleCreateOptions) String() string {
	ret := fmt.Sprintf("%s_%s_%s", rule.Direction, rule.Action, rule.Protocol)
	if len(rule.CIDR) > 0 {
		ret += "_" + rule.CIDR
	}
	if len(rule.Ports) > 0 {
		ret += "_" + rule.Ports
	}
	ret = strings.ReplaceAll(ret, ".", "_")
	ret = strings.ReplaceAll(ret, ",", "_")
	return ret
}
