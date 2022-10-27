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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SUcloudRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SUcloudRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SUcloudRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return false
}

func (self *SUcloudRegionDriver) GenerateSecurityGroupName(name string) string {
	return name
}

func (self *SUcloudRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SUcloudRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:allow any")}
}

func (self *SUcloudRegionDriver) GetSecurityGroupRuleMaxPriority() int {
	return 3
}

func (self *SUcloudRegionDriver) GetSecurityGroupRuleMinPriority() int {
	return 1
}

func (self *SUcloudRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_UCLOUD
}

func (self *SUcloudRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer", self.GetProvider())
}

func (self *SUcloudRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer acl", self.GetProvider())
}

func (self *SUcloudRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer certificate", self.GetProvider())
}

func (self *SUcloudRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	var cidrV = validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
		return input, err
	}
	err := IsInPrivateIpRange(cidrV.Value.ToIPRange())
	if err != nil {
		return input, err
	}

	if cidrV.Value.MaskLen > 29 {
		return input, httperrors.NewInputParameterError("%s request the mask range should be less than or equal to 29", self.GetProvider())
	}
	return input, nil
}
