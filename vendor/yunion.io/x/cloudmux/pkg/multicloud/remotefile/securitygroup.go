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

package remotefile

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	SResourceBase

	VpcId string
	Desc  string
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Desc
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	return []cloudprovider.ISecurityGroupRule{}, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SSecurityGroup) Delete() error {
	return cloudprovider.ErrNotSupported
}
