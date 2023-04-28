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

package jdcloud

import (
	"fmt"
	"net"

	commodels "github.com/jdcloud-api/jdcloud-sdk-go/services/common/models"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/models"

	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	JdcloudTags

	vpc *SVpc
	models.NetworkSecurityGroup
}

func (sg *SSecurityGroup) GetVpcId() string {
	return sg.VpcId
}

func (sg *SSecurityGroup) GetId() string {
	return sg.NetworkSecurityGroupId
}

func (sg *SSecurityGroup) GetGlobalId() string {
	return sg.GetId()
}

func (sg *SSecurityGroup) GetName() string {
	return sg.NetworkSecurityGroupName
}

func (sg *SSecurityGroup) GetDescription() string {
	return sg.Description
}

func (sg *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	rules := sg.SecurityGroupRules
	srs := make([]cloudprovider.SecurityRule, 0, len(rules))
	for i := range rules {
		rule := secrules.SecurityRule{
			Priority: 1,
		}

		if rules[i].Direction == 0 {
			rule.Direction = secrules.SecurityRuleIngress
		} else {
			rule.Direction = secrules.SecurityRuleEgress
		}

		switch rules[i].Protocol {
		case 6:
			rule.Protocol = secrules.PROTO_TCP
		case 17:
			rule.Protocol = secrules.PROTO_UDP
		case 1:
			rule.Protocol = secrules.PROTO_ICMP
		case 300:
			rule.Protocol = secrules.PROTO_ANY
		}

		_, rule.IPNet, _ = net.ParseCIDR(rules[i].AddressPrefix)
		rule.Description = rules[i].Description
		rule.PortStart = rules[i].FromPort
		rule.PortEnd = rules[i].ToPort
		rule.Action = secrules.SecurityRuleAllow

		if rules[i].RuleType == "default" && rules[i].FromPort == 0 && rules[i].ToPort == 0 {
			rule.Action = secrules.SecurityRuleDeny
		}

		sr := cloudprovider.SecurityRule{
			Id:           rules[i].RuleId,
			SecurityRule: rule,
		}
		srs = append(srs, sr)
	}
	return srs, nil
}

func (sg *SSecurityGroup) GetStatus() string {
	return ""
}

func (sg *SSecurityGroup) IsEmulated() bool {
	return false
}

func (sg *SSecurityGroup) Refresh() error {
	return nil
}

func (sg *SSecurityGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (sg *SSecurityGroup) GetProjectId() string {
	return ""
}

func (sg *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	return nil
}

func (r *SRegion) GetSecurityGroups(vpcId string, securityGroupIds []string, pageNumber int, pageSize int) ([]SSecurityGroup, int, error) {
	filters := []commodels.Filter{}
	if vpcId != "" {
		filters = append(filters, commodels.Filter{
			Name:   "vpcId",
			Values: []string{vpcId},
		})
	}
	if len(securityGroupIds) > 0 {
		filters = append(filters, commodels.Filter{
			Name:   "networkSecurityGroupIds",
			Values: securityGroupIds,
		})
	}
	req := apis.NewDescribeNetworkSecurityGroupsRequestWithAllParams(r.ID, &pageNumber, &pageSize, filters)
	client := client.NewVpcClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeNetworkSecurityGroups(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error.Code >= 400 {
		err = fmt.Errorf(resp.Error.Message)
		return nil, 0, err
	}
	total := resp.Result.TotalCount
	sgs := make([]SSecurityGroup, 0, len(resp.Result.NetworkSecurityGroups))
	for i := range resp.Result.NetworkSecurityGroups {
		sgs = append(sgs, SSecurityGroup{
			NetworkSecurityGroup: resp.Result.NetworkSecurityGroups[i],
		})
	}
	return sgs, total, nil
}
