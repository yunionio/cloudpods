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
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SKsyunRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SKsyunRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SKsyunRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_KSYUN
}

func (self *SKsyunRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	return nil
}

func (self *SKsyunRegionDriver) CreateDefaultSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	vpc *models.SVpc,
) (*models.SSecurityGroup, error) {
	region, err := vpc.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	driver := region.GetDriver()
	newGroup := &models.SSecurityGroup{}
	newGroup.SetModelManager(models.SecurityGroupManager, newGroup)
	newGroup.Name = fmt.Sprintf("default-auto-%d", time.Now().Unix())
	// 部分云可能不需要vpcId, 创建完安全组后会自动置空
	newGroup.VpcId = vpc.Id
	newGroup.ManagerId = vpc.ManagerId
	newGroup.CloudregionId = vpc.CloudregionId
	newGroup.DomainId = ownerId.GetProjectDomainId()
	newGroup.ProjectId = ownerId.GetProjectId()
	newGroup.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
	err = models.SecurityGroupManager.TableSpec().Insert(ctx, newGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "insert")
	}

	err = driver.RequestCreateSecurityGroup(ctx, userCred, newGroup, api.SSecgroupRuleResourceSet{})
	if err != nil {
		return nil, errors.Wrapf(err, "RequestCreateSecurityGroup")
	}
	return newGroup, nil
}
