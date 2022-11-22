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
	"context"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCtyunRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SCtyunRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SCtyunRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return true
}

func (self *SCtyunRegionDriver) GenerateSecurityGroupName(name string) string {
	if strings.ToLower(name) == "default" {
		return "DefaultGroup"
	}
	return name
}

func (self *SCtyunRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SCtyunRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:deny any")}
}

func (self *SCtyunRegionDriver) GetSecurityGroupRuleMaxPriority() int {
	return 0
}

func (self *SCtyunRegionDriver) GetSecurityGroupRuleMinPriority() int {
	return 0
}

func (self *SCtyunRegionDriver) IsOnlySupportAllowRules() bool {
	return true
}

func (self *SCtyunRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CTYUN
}

func (self *SCtyunRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	cidrV := validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
		return input, err
	}

	err := IsInPrivateIpRange(cidrV.Value.ToIPRange())
	if err != nil {
		return input, err
	}

	if cidrV.Value.MaskLen > 24 {
		return input, httperrors.NewInputParameterError("invalid cidr range %s, mask length should less than or equal to 24", cidrV.Value.String())
	}
	return input, nil
}
