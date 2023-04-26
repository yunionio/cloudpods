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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	return true
}

func (self *SBingoCloudRegionDriver) GenerateSecurityGroupName(name string) string {
	if strings.ToLower(name) == "default" {
		return "default"
	}
	return name
}

func (self *SBingoCloudRegionDriver) IsSecurityGroupBelongVpc() bool {
	return false
}

func (self *SBingoCloudRegionDriver) IsSupportClassicSecurityGroup() bool {
	return false
}

func (self *SBingoCloudRegionDriver) IsSecurityGroupBelongGlobalVpc() bool {
	return false
}

func (self *SBingoCloudRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:allow any")}
}

func (self *SBingoCloudRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:allow any")}
}

func (self *SBingoCloudRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	return nil
}
