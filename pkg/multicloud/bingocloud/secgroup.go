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

package bingocloud

import (
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SResourceBase
	multicloud.BingoTags
	region *SRegion

	ComplexIPPermissions       string          `json:"complexIpPermissions"`
	ComplexIPPermissionsEgress string          `json:"complexIpPermissionsEgress"`
	DisplayName                string          `json:"displayName"`
	GroupDescription           string          `json:"groupDescription"`
	GroupId                    string          `json:"groupId"`
	GroupName                  string          `json:"groupName"`
	IPPermissionType           string          `json:"ipPermissionType"`
	IPPermissions              []IPPermissions `json:"ipPermissions"`
	IPPermissionsEgress        []IPPermissions `json:"ipPermissionsEgress"`
	OwnerId                    string          `json:"ownerId"`
}

type IPPermissions struct {
	BoundType   string `json:"boundType"`
	Description string `json:"description"`
	FromPort    int    `json:"fromPort"`
	IPProtocol  string `json:"ipProtocol"`
	Groups      []struct {
		GroupId   string
		GroupName string
	} `json:"groups"`
	IPRanges []struct {
		CIdRIP string `json:"cidrIp"`
	} `json:"ipRanges"`
	L2Accept     string `json:"l2Accept"`
	PermissionId string `json:"permissionId"`
	Policy       string `json:"policy"`
	ToPort       int    `json:"toPort"`
}

func (self *SSecurityGroup) GetId() string {
	return self.GroupId
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.GetId()
}

func (self *SSecurityGroup) GetName() string {
	return self.GroupName
}

func (self *SSecurityGroup) GetDescription() string {
	return self.GroupDescription
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	return []cloudprovider.SecurityGroupReference{}, nil
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	var ret []cloudprovider.SecurityRule
	for _, _rule := range append(self.IPPermissionsEgress, self.IPPermissions...) {
		if len(_rule.Groups) > 0 {
			continue
		}
		rule := cloudprovider.SecurityRule{}
		rule.Direction = secrules.DIR_IN
		rule.Priority = 1
		rule.Action = secrules.SecurityRuleAllow
		rule.Protocol = secrules.PROTO_ANY
		rule.Description = _rule.Description
		if _rule.BoundType == "Out" {
			rule.Direction = secrules.DIR_OUT
		}
		if _rule.Policy == "DROP" {
			rule.Action = secrules.SecurityRuleDeny
		}
		if _rule.IPProtocol != "all" {
			rule.Protocol = _rule.IPProtocol
		}
		if rule.Protocol == secrules.PROTO_TCP || rule.Protocol == secrules.PROTO_UDP {
			rule.PortStart, rule.PortEnd = _rule.FromPort, _rule.ToPort
		}

		for _, ip := range _rule.IPRanges {
			if ip.CIdRIP == "::/0" {
				ip.CIdRIP = "0.0.0.0/0"
			}
			rule.ParseCIDR(ip.CIdRIP)
			err := rule.ValidateRule()
			if err != nil {
				return nil, err
			}
			ret = append(ret, rule)
		}
	}
	return ret, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return api.NORMAL_VPC_ID
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	return nil
}

func (self *SSecurityGroup) Delete() error {
	return self.region.deleteSecurityGroup(self.GroupId)
}

func (self *SRegion) GetSecurityGroups(id, name, nextToken string) ([]SSecurityGroup, string, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["GroupId.1"] = id
	}
	if len(name) > 0 {
		params["Filter.1.Name"] = "group-name"
		params["Filter.1.Value.1"] = name
	}
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
	}

	resp, err := self.invoke("DescribeSecurityGroups", params)
	if err != nil {
		return nil, "", err
	}
	ret := struct {
		SecurityGroupInfo []SSecurityGroup
		NextToken         string
	}{}
	_ = resp.Unmarshal(&ret)
	return ret.SecurityGroupInfo, ret.NextToken, nil
}

func (self *SRegion) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	groups, _, err := self.GetSecurityGroups(id, "", "")
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].GetGlobalId() == id {
			groups[i].region = self
			return &groups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	groups, _, err := self.GetSecurityGroups("", opts.Name, "")
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].GetName() == opts.Name {
			groups[i].region = self
			return &groups[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) deleteSecurityGroup(id string) error {
	params := map[string]string{}
	params["GroupId"] = id

	_, err := self.invoke("DeleteSecurityGroup", params)
	return err
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}
