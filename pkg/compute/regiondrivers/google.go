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

package regiondrivers

import (
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SGoogleRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SGoogleRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SGoogleRegionDriver) GetSecurityGroupRuleOrder() cloudprovider.TPriorityOrder {
	return cloudprovider.PriorityOrderByAsc
}

func (self *SGoogleRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SGoogleRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:allow any")}
}

func (self *SGoogleRegionDriver) GetSecurityGroupRuleMaxPriority() int {
	return 0
}

func (self *SGoogleRegionDriver) GetSecurityGroupRuleMinPriority() int {
	return 65535
}

func (self *SGoogleRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_GOOGLE
}

func (self *SGoogleRegionDriver) IsSecurityGroupBelongGlobalVpc() bool {
	return true
}

func (self *SGoogleRegionDriver) IsVpcBelongGlobalVpc() bool {
	return true
}
