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
	"yunion.io/x/pkg/util/pinyinutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAzureRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SAzureRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SAzureRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_AZURE
}

func (self *SAzureRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return false
}

func (self *SAzureRegionDriver) GenerateSecurityGroupName(name string) string {
	return pinyinutils.Text2Pinyin(name)
}

func (self *SAzureRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	cidrV := validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
		return input, err
	}
	if cidrV.Value.MaskLen < 8 || cidrV.Value.MaskLen > 29 {
		return input, httperrors.NewInputParameterError("%s request the mask range should be between 8 and 29", self.GetProvider())
	}
	return input, nil
}

func (self *SAzureRegionDriver) ValidateCreateWafInstanceData(ctx context.Context, userCred mcclient.TokenCredential, input api.WafInstanceCreateInput) (api.WafInstanceCreateInput, error) {
	if len(input.Type) == 0 {
		input.Type = cloudprovider.WafTypeAppGateway
	}
	switch input.Type {
	case cloudprovider.WafTypeAppGateway:
	default:
		return input, httperrors.NewInputParameterError("Invalid azure waf type %s", input.Type)
	}
	if input.DefaultAction == nil {
		input.DefaultAction = &cloudprovider.DefaultAction{}
	}
	if len(input.DefaultAction.Action) == 0 {
		input.DefaultAction.Action = cloudprovider.WafActionDetection
	}
	switch input.DefaultAction.Action {
	case cloudprovider.WafActionPrevention:
	case cloudprovider.WafActionDetection:
	default:
		return input, httperrors.NewInputParameterError("invalid default action %s", input.DefaultAction.Action)
	}
	return input, nil
}

func (self *SAzureRegionDriver) ValidateCreateWafRuleData(ctx context.Context, userCred mcclient.TokenCredential, waf *models.SWafInstance, input api.WafRuleCreateInput) (api.WafRuleCreateInput, error) {
	return input, nil
}
