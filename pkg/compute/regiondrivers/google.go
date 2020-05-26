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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SGoogleRegionDriver) RequestCreateVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		provider := vpc.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to found vpc %s(%s) cloudprovider", vpc.Name, vpc.Id)
		}
		providerDriver, err := provider.GetProvider()
		if err != nil {
			return nil, errors.Wrap(err, "provider.GetProvider")
		}
		iregion, err := providerDriver.GetIRegionById(region.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetIRegion")
		}
		ivpc, err := iregion.CreateIVpc(vpc.Name, vpc.Description, vpc.CidrBlock)
		if err != nil {
			return nil, errors.Wrap(err, "iregion.CreateIVpc")
		}
		db.SetExternalId(vpc, userCred, ivpc.GetGlobalId())

		regions, err := models.CloudregionManager.GetRegionByExternalIdPrefix(self.GetProvider())
		if err != nil {
			return nil, errors.Wrap(err, "GetRegionByExternalIdPrefix")
		}
		for _, region := range regions {
			iregion, err := providerDriver.GetIRegionById(region.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "providerDrivder.GetIRegionById")
			}
			region.SyncVpcs(ctx, userCred, iregion, provider)
		}

		err = vpc.SyncWithCloudVpc(ctx, userCred, ivpc)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncWithCloudVpc")
		}

		err = vpc.SyncRemoteWires(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncRemoteWires")
		}
		return nil, nil
	})
	return nil
}

func (self *SGoogleRegionDriver) RequestDeleteVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		region, err := vpc.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetIRegion")
		}
		ivpc, err := region.GetIVpcById(vpc.GetExternalId())
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				err = vpc.Purge(ctx, userCred)
				if err != nil {
					return nil, errors.Wrap(err, "vpc.Purge")
				}
				return nil, nil
			}
			return nil, errors.Wrap(err, "region.GetIVpcById")
		}

		globalVpc, err := vpc.GetGlobalVpc()
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetGlobalVpc")
		}

		vpcs, err := globalVpc.GetVpcs()
		if err != nil {
			return nil, errors.Wrap(err, "globalVpc.GetVpcs")
		}

		for i := range vpcs {
			if vpcs[i].Status == api.VPC_STATUS_AVAILABLE && vpcs[i].ManagerId == vpc.ManagerId {
				err = vpc.ValidateDeleteCondition(ctx)
				if err != nil {
					return nil, errors.Wrapf(err, "vpc %s(%s) not empty", vpc.Name, vpc.Id)
				}
			}
		}

		err = ivpc.Delete()
		if err != nil {
			return nil, errors.Wrap(err, "ivpc.Delete")
		}

		for i := range vpcs {
			if vpcs[i].ManagerId == vpc.ManagerId && vpcs[i].Id != vpc.Id {
				err = vpcs[i].Purge(ctx, userCred)
				if err != nil {
					return nil, errors.Wrapf(err, "vpc.Purge %s(%s)", vpc.Name, vpc.Id)
				}
			}
		}

		return nil, nil
	})
	return nil

}
