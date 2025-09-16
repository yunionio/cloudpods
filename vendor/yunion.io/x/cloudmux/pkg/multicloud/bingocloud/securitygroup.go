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
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	BingoTags
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

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := []cloudprovider.ISecurityGroupRule{}
	for i := range self.IPPermissionsEgress {
		self.IPPermissionsEgress[i].direction = secrules.DIR_OUT
		ret = append(ret, &self.IPPermissionsEgress[i])
	}
	for i := range self.IPPermissions {
		self.IPPermissions[i].direction = secrules.DIR_IN
		ret = append(ret, &self.IPPermissions[i])
	}
	return ret, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return ""
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Delete() error {
	return self.region.deleteSecurityGroup(self.GroupId)
}

func (self *SRegion) CreateSecurityGroupRules(secGrpId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) error {
	params := map[string]string{
		"GroupId":    secGrpId,
		"IpProtocol": "all",
		"BoundType":  "In",
		"Policy":     "DROP",
		"FromPort":   "0",
		"ToPort":     "65535",
	}
	if opts.Protocol != secrules.PROTO_ANY {
		params["IpProtocol"] = opts.Protocol
	}
	if opts.Direction == secrules.DIR_OUT {
		params["BoundType"] = "Out"
	}
	if opts.Action == secrules.SecurityRuleAllow {
		params["Policy"] = "ACCEPT"
	}

	start, end := 0, 0
	if len(opts.Ports) > 0 {
		if strings.Contains(opts.Ports, "-") {
			ports := strings.Split(opts.Ports, "-")
			if len(ports) != 2 {
				return errors.Errorf("invalid ports %s", opts.Ports)
			}
			var err error
			_start, _end := ports[0], ports[1]
			start, err = strconv.Atoi(_start)
			if err != nil {
				return errors.Errorf("invalid start port %s", _start)
			}
			end, err = strconv.Atoi(_end)
			if err != nil {
				return errors.Errorf("invalid end port %s", _end)
			}
		} else {
			port, err := strconv.Atoi(opts.Ports)
			if err != nil {
				return errors.Errorf("invalid ports %s", opts.Ports)
			}
			start, end = port, port
		}
	}
	if start > 0 && end > 0 {
		params["FromPort"] = fmt.Sprintf("%d", start)
		params["ToPort"] = fmt.Sprintf("%d", end)
	}

	_, err := self.invoke("AuthorizeSecurityGroupIngress", params)
	if err == nil {
		return errors.Wrapf(err, "AuthorizeSecurityGroupIngress")
	}
	return nil
}

func (self *SRegion) GetSecurityGroups(id, name, nextToken string) ([]SSecurityGroup, string, error) {
	params := map[string]string{}
	params["Filter.1.Name"] = "owner-id"
	params["Filter.1.Value.1"] = self.getAccountUser()

	if len(id) > 0 {
		params["GroupId.1"] = id
	}
	if len(name) > 0 {
		params["Filter.2.Name"] = "group-name"
		params["Filter.2.Value.1"] = name
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

func (self *SRegion) deleteSecurityGroup(id string) error {
	params := map[string]string{}
	params["GroupId"] = id

	_, err := self.invoke("DeleteSecurityGroup", params)
	return err
}

type SecurityGroupCreateOutput struct {
	Return  bool
	GroupId string
}

func (self *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	params := map[string]string{}
	if len(opts.Name) > 0 {
		params["GroupName"] = opts.Name
	}
	resp, err := self.invoke("CreateSecurityGroup", params)
	if err != nil {
		return nil, err
	}

	ret := &SecurityGroupCreateOutput{}
	_ = resp.Unmarshal(&ret)

	if ret.Return {
		return self.GetISecurityGroupById(ret.GroupId)
	}

	return nil, errors.Wrap(cloudprovider.ErrUnknown, "CreateSecurityGroup")
}
