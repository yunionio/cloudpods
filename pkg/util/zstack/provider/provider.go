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

package provider

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

type SZStackProviderFactory struct {
	// providerTable map[string]*SZStackProvider
}

func (self *SZStackProviderFactory) GetId() string {
	return zstack.CLOUD_PROVIDER_ZSTACK
}

func (self *SZStackProviderFactory) GetName() string {
	return zstack.CLOUD_PROVIDER_ZSTACK
}

func (self *SZStackProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (self *SZStackProviderFactory) IsPublicCloud() bool {
	return false
}

func (self *SZStackProviderFactory) IsOnPremise() bool {
	return false
}

func (self *SZStackProviderFactory) IsSupportPrepaidResources() bool {
	return false
}

func (self *SZStackProviderFactory) NeedSyncSkuFromCloud() bool {
	return true
}

func (self *SZStackProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	return nil
}

func (self *SZStackProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	account := &cloudprovider.SCloudaccount{}
	return account, nil
}

func (self *SZStackProviderFactory) GetProvider(providerId, providerName, url, account, password string) (cloudprovider.ICloudProvider, error) {
	accountInfo := strings.Split(account, "/")
	if len(accountInfo) < 2 {
		return nil, fmt.Errorf("Missing username or project name %s", account)
	}
	project, username, endpointType := accountInfo[0], accountInfo[1], "internal"
	if len(accountInfo) == 3 {
		endpointType = accountInfo[2]
	}
	client, err := zstack.NewZStackClient(providerId, providerName, url, username, password, false)
	if err != nil {
		return nil, err
	}
	return &SZStackProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func init() {
	factory := SZStackProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SZStackProvider struct {
	cloudprovider.SBaseProvider
	client *zstack.SZStackClient
}

func (self *SZStackProvider) GetVersion() string {
	return ""
}

func (self *SZStackProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (self *SZStackProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SZStackProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SZStackProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SZStackProvider) GetBalance() (float64, string, error) {
	return 0.0, api.CLOUD_PROVIDER_HEALTH_UNKNOWN, cloudprovider.ErrNotSupported
}

func (self *SZStackProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return self.client.GetIProjects()
}
