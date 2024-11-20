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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/sqlchemy"

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

func (self *SUcloudRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_UCLOUD
}

func (self *SUcloudRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	var cidrV = validators.NewIPv4PrefixValidator("cidr_block")
	if err := cidrV.Validate(ctx, jsonutils.Marshal(input).(*jsonutils.JSONDict)); err != nil {
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

func (self *SUcloudRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		if input.Rules[i].Protocol == secrules.PROTO_ANY {
			return nil, httperrors.NewNotSupportedError("protocol %s", input.Rules[i].Protocol)
		}
		if input.Rules[i].Priority == nil {
			return nil, httperrors.NewMissingParameterError("priority")
		}
		if *input.Rules[i].Priority < 1 || *input.Rules[i].Priority > 3 {
			return nil, httperrors.NewInputParameterError("invalid priority %d, range 1-3", *input.Rules[i].Priority)
		}
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateSecurityGroupInput(ctx, userCred, input)
}

func (self *SUcloudRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Priority != nil && (*input.Priority < 1 || *input.Priority > 3) {
		return nil, httperrors.NewInputParameterError("invalid priority %d, range 1-3", *input.Priority)
	}

	if input.Protocol != nil && *input.Protocol == secrules.PROTO_ANY {
		return nil, httperrors.NewNotSupportedError("protocol %s", *input.Protocol)
	}

	if input.Ports != nil && strings.Contains(*input.Ports, ",") {
		return nil, httperrors.NewInputParameterError("invalid ports %s", *input.Ports)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateSecurityGroupRuleInput(ctx, userCred, input)
}

func (self *SUcloudRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("cloudregion_id", vpc.CloudregionId).Equals("manager_id", vpc.ManagerId)
	}, nil
}
