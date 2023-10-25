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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBingoCloudRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SBingoCloudRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SBingoCloudRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_BINGO_CLOUD
}

func (self *SBingoCloudRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	return nil
}

func (self *SBingoCloudRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		rule := input.Rules[i]
		if len(rule.Ports) > 0 && strings.Contains(rule.Ports, ",") {
			return nil, httperrors.NewInputParameterError("invalid ports %s", rule.Ports)
		}
	}
	return self.SManagedVirtualizationRegionDriver.ValidateCreateSecurityGroupInput(ctx, userCred, input)
}

func (self *SBingoCloudRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Ports != nil && strings.Contains(*input.Ports, ",") {
		return nil, httperrors.NewInputParameterError("invalid ports %s", *input.Ports)
	}

	return self.SManagedVirtualizationRegionDriver.ValidateUpdateSecurityGroupRuleInput(ctx, userCred, input)
}

func (self *SBingoCloudRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("cloudregion_id", vpc.CloudregionId)
	}, nil
}

func (self *SBingoCloudRegionDriver) CreateDefaultSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	vpc *models.SVpc,
) (*models.SSecurityGroup, error) {
	newGroup := &models.SSecurityGroup{}
	newGroup.SetModelManager(models.SecurityGroupManager, newGroup)
	newGroup.Name = fmt.Sprintf("default-auto-%d", time.Now().Unix())
	newGroup.Description = "auto generage"
	newGroup.ManagerId = vpc.ManagerId
	newGroup.CloudregionId = vpc.CloudregionId
	newGroup.DomainId = ownerId.GetDomainId()
	newGroup.ProjectId = ownerId.GetProjectId()
	err := models.SecurityGroupManager.TableSpec().Insert(ctx, newGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "insert")
	}

	region, err := vpc.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	driver := region.GetDriver()
	err = driver.RequestCreateSecurityGroup(ctx, userCred, newGroup, api.SSecgroupRuleResourceSet{})
	if err != nil {
		return nil, errors.Wrapf(err, "RequestCreateSecurityGroup")
	}
	return newGroup, nil
}
