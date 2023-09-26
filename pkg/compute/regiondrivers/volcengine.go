// Copyright 2023 Yunion
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

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SVolcengineRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SVolcengineRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SVolcengineRegionDriver) IsSecurityGroupBelongVpc() bool {
	return true
}

func (self *SVolcengineRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return false
}

func (self *SVolcengineRegionDriver) GenerateSecurityGroupName(name string) string {
	return name
}

func (self *SVolcengineRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_VOLCENGINE
}

func (self *SVolcengineRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
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
