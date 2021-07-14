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

type SJDcloudRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SJDcloudRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SJDcloudRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_JDCLOUD
}

// https://docs.jdcloud.com/cn/ssl-certificate/api/overview
func (self *SJDcloudRegionDriver) IsCertificateBelongToRegion() bool {
	return false
}

// https://docs.jdcloud.com/cn/virtual-private-cloud/api/modifynetworksecuritygrouprules
func (self *SJDcloudRegionDriver) IsOnlySupportAllowRules() bool {
	return true
}

func (self *SJDcloudRegionDriver) IsSecurityGroupBelongVpc() bool {
	return true
}

func (self *SJDcloudRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SJDcloudRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:allow any")}
}
