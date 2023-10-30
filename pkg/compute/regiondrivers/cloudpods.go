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

	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudpodsRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SCloudpodsRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SCloudpodsRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CLOUDPODS
}

func (self *SCloudpodsRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	if !utils.IsInStringArray(input.CidrBlock, []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"}) {
		return input, httperrors.NewInputParameterError("Invalid cidr_block, want 192.168.0.0/16|10.0.0.0/8|172.16.0.0/12, got %s", input.CidrBlock)
	}
	return input, nil
}

func (self *SCloudpodsRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		rule := input.Rules[i]
		if rule.Priority == nil {
			return nil, httperrors.NewMissingParameterError("priority")
		}

		if *rule.Priority < 1 || *rule.Priority > 100 {
			return nil, httperrors.NewInputParameterError("invalid priority %d, range 1-100", *rule.Priority)
		}

	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateSecurityGroupInput(ctx, userCred, input)
}

func (self *SCloudpodsRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Priority != nil && (*input.Priority < 1 || *input.Priority > 100) {
		return nil, httperrors.NewInputParameterError("invalid priority %d, range 1-100", *input.Priority)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateSecurityGroupRuleInput(ctx, userCred, input)
}

func (self *SCloudpodsRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("cloudregion_id", vpc.CloudregionId)
	}, nil
}
