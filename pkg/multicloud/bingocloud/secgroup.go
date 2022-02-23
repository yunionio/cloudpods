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
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SSecGroup struct {
	ComplexIpPermissions       string `json:"complexIpPermissions"`
	ComplexIpPermissionsEgress string `json:"complexIpPermissionsEgress"`
	DisplayName                string `json:"displayName"`
	GroupDescription           string `json:"groupDescription"`
	GroupId                    string `json:"groupId"`
	GroupName                  string `json:"groupName"`
	IpPermissionType           string `json:"ipPermissionType"`
	IpPermissions              string `json:"ipPermissions"`
	IpPermissionsEgress        struct {
		BoundType   string `json:"boundType"`
		Description string `json:"description"`
		FromPort    string `json:"fromPort"`
		Groups      string `json:"groups"`
		IpProtocol  string `json:"ipProtocol"`
		IpRanges    struct {
			Item struct {
				CidrIp string `json:"cidrIp"`
			}
		}
	} `json:"ipPermissionsEgress"`
	OwnerId string `json:"ownerId"`
}

func (self *SRegion) DescribeSecurityGroups() ([]SSecGroup, error) {
	resp, err := self.invoke("DescribeSecurityGroups", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		SecurityGroupInfo struct {
			Item []SSecGroup
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.SecurityGroupInfo.Item, nil
}

func (self *SRegion) DescribeSecurityGroup(id string) (*SSecGroup, error) {
	secgroup := &SSecGroup{}
	return secgroup, cloudprovider.ErrNotImplemented
}
