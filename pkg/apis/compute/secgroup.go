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

package compute

import (
	"fmt"

	"github.com/pkg/errors"

	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/apis"
)

type SSecgroupRuleCreateInput struct {
	apis.Meta

	Priority    int
	Protocol    string
	Ports       string
	PortStart   int
	PortEnd     int
	Direction   string
	CIDR        string
	Action      string
	Description string
	SecgroupId  string
}

func (input *SSecgroupRuleCreateInput) Check() error {
	rule := secrules.SecurityRule{
		Priority:  input.Priority,
		Direction: secrules.TSecurityRuleDirection(input.Direction),
		Action:    secrules.TSecurityRuleAction(input.Action),
		Protocol:  input.Protocol,
		PortStart: input.PortStart,
		PortEnd:   input.PortEnd,
		Ports:     []int{},
	}

	if len(input.Ports) > 0 {
		err := rule.ParsePorts(input.Ports)
		if err != nil {
			return errors.Wrapf(err, "ParsePorts(%s)", input.Ports)
		}
	}

	if len(input.CIDR) > 0 {
		if !regutils.MatchCIDR(input.CIDR) && !regutils.MatchIPAddr(input.CIDR) {
			return fmt.Errorf("invalid ip address: %s", input.CIDR)
		}
	} else {
		input.CIDR = "0.0.0.0/0"
	}

	return rule.ValidateRule()
}

type SSecgroupCreateInput struct {
	apis.Meta

	Name        string
	Status      string
	Description string
	Rules       []SSecgroupRuleCreateInput
}
