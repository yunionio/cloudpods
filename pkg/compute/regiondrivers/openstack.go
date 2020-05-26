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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SOpenStackRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SOpenStackRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SOpenStackRegionDriver) GetSecurityGroupRuleOrder() cloudprovider.TPriorityOrder {
	return cloudprovider.PriorityOrderByDesc
}

func (self *SOpenStackRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SOpenStackRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:deny any")}
}

func (self *SOpenStackRegionDriver) GetSecurityGroupRuleMaxPriority() int {
	return 0
}

func (self *SOpenStackRegionDriver) GetSecurityGroupRuleMinPriority() int {
	return 0
}

func (self *SOpenStackRegionDriver) IsOnlySupportAllowRules() bool {
	return true
}

func (self *SOpenStackRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackRegionDriver) IsVpcCreateNeedInputCidr() bool {
	return false
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer", self.GetProvider())
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer acl", self.GetProvider())
}

func (self *SOpenStackRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer certificate", self.GetProvider())
}

func (self *SOpenStackRegionDriver) ValidateCreateEipData(ctx context.Context, userCred mcclient.TokenCredential, input *api.SElasticipCreateInput) error {
	if len(input.Network) == 0 {
		return httperrors.NewMissingParameterError("network")
	}
	_network, err := models.NetworkManager.FetchByIdOrName(userCred, input.Network)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperrors.NewResourceNotFoundError("failed to found network %s", input.Network)
		}
		return httperrors.NewGeneralError(err)
	}
	network := _network.(*models.SNetwork)
	input.NetworkId = network.Id

	vpc := network.GetVpc()
	if vpc == nil {
		return httperrors.NewInputParameterError("failed to found vpc for network %s(%s)", network.Name, network.Id)
	}
	region, err := vpc.GetRegion()
	if err != nil {
		return err
	}
	if region.GetDriver().GetProvider() != self.GetProvider() {
		return httperrors.NewUnsupportOperationError("network %s(%s) does not belong to %s", network.Name, network.Id, self.GetProvider())
	}
	return nil
}
