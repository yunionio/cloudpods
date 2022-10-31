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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

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

func (self *SBingoCloudRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return false
}

func (self *SBingoCloudRegionDriver) GenerateSecurityGroupName(name string) string {
	if strings.ToLower(name) == "default" {
		return "DefaultGroup"
	}
	return name
}

func (self *SBingoCloudRegionDriver) IsSecurityGroupBelongVpc() bool {
	return false
}

func (self *SBingoCloudRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SBingoCloudRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:deny any")}
}

func (self *SBingoCloudRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not support creating loadbalancer acl", self.GetProvider())
}

func (self *SBingoCloudRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not support creating loadbalancer certificate", self.GetProvider())
}

func (self *SBingoCloudRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	return fmt.Errorf("%s does not support creating snapshot", self.GetProvider())
}
