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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/oracle"
)

type SOracleProviderFactory struct {
	cloudprovider.SPublicCloudBaseProviderFactory
}

func (self *SOracleProviderFactory) GetId() string {
	return api.CLOUD_PROVIDER_ORACLE
}

func (self *SOracleProviderFactory) GetName() string {
	return oracle.CLOUD_PROVIDER_ORACLE_CN
}

func (self *SOracleProviderFactory) IsReadOnly() bool {
	return true
}

func (self *SOracleProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, input cloudprovider.SCloudaccountCredential) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.OracleTenancyOCID) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "oracle_tenancy_ocid")
	}
	if len(input.OracleUserOCID) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "oracle_user_ocid")
	}
	if len(input.OraclePrivateKey) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "oracle_private_key")
	}
	output.AccessUrl = input.OracleTenancyOCID
	output.Account = input.OracleUserOCID
	output.Secret = input.OraclePrivateKey
	return output, nil
}

func (self *SOracleProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, input cloudprovider.SCloudaccountCredential, cloudaccount string) (cloudprovider.SCloudaccount, error) {
	output := cloudprovider.SCloudaccount{}
	if len(input.OracleTenancyOCID) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "oracle_tenancy_ocid")
	}
	if len(input.OracleUserOCID) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "oracle_user_ocid")
	}
	if len(input.OraclePrivateKey) == 0 {
		return output, errors.Wrap(cloudprovider.ErrMissingParameter, "oracle_private_key")
	}
	output = cloudprovider.SCloudaccount{
		AccessUrl: input.OracleTenancyOCID,
		Account:   input.OracleUserOCID,
		Secret:    input.OraclePrivateKey,
	}
	return output, nil
}

func parseCompartment(key string) (string, string) {
	userOCID, compartment := key, ""
	if strings.Contains(key, "/") {
		info := strings.Split(key, "/")
		if len(info) == 2 {
			userOCID, compartment = info[0], info[1]
		}
	}
	return userOCID, compartment
}

func (self *SOracleProviderFactory) GetProvider(cfg cloudprovider.ProviderConfig) (cloudprovider.ICloudProvider, error) {
	userOCID, compartment := parseCompartment(cfg.Account)
	opts, err := oracle.NewOracleClientConfig(
		cfg.URL,
		userOCID,
		compartment,
		cfg.Secret,
	)
	if err != nil {
		return nil, err
	}
	client, err := oracle.NewOracleClient(
		opts.CloudproviderConfig(cfg),
	)
	if err != nil {
		return nil, err
	}

	return &SOracleProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func (self *SOracleProviderFactory) GetClientRC(info cloudprovider.SProviderInfo) (map[string]string, error) {
	userOCID, compartment := parseCompartment(info.Account)
	region := info.Region
	if len(region) == 0 {
		region = oracle.ORACLE_DEFAULT_REGION
	}
	return map[string]string{
		"ORACLE_TENANCY_OCID":   info.Url,
		"ORACLE_USER_OCID":      userOCID,
		"ORACLE_COMPARTMENT_ID": compartment,
		"ORACLE_PRIVATE_KEY":    info.Secret,
		"ORACLE_REGION_ID":      region,
	}, nil
}

func init() {
	factory := SOracleProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SOracleProvider struct {
	cloudprovider.SBaseProvider
	client *oracle.SOracleClient
}

func (self *SOracleProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions, err := self.client.GetRegions()
	if err != nil {
		return nil, err
	}
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	return info, nil
}

func (self *SOracleProvider) GetVersion() string {
	return ""
}

func (self *SOracleProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SOracleProvider) GetAccountId() string {
	return self.client.GetAccountId()
}

func (self *SOracleProvider) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	regions, err := self.client.GetRegions()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudRegion{}
	for i := range regions {
		ret = append(ret, &regions[i])
	}
	return ret, nil
}

func (self *SOracleProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	region, err := self.client.GetRegion(extId)
	if err != nil {
		return nil, err
	}
	return region, nil
}

func (self *SOracleProvider) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	ret := &cloudprovider.SBalanceInfo{Currency: "CNY", Status: api.CLOUD_PROVIDER_HEALTH_UNKNOWN}
	return ret, cloudprovider.ErrNotSupported
}

func (self *SOracleProvider) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SOracleProvider) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SOracleProvider) GetStorageClasses(regionId string) []string {
	return []string{}
}

func (self *SOracleProvider) GetBucketCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SOracleProvider) GetObjectCannedAcls(regionId string) []string {
	return []string{
		string(cloudprovider.ACLPrivate),
		string(cloudprovider.ACLPublicRead),
		string(cloudprovider.ACLPublicReadWrite),
	}
}

func (self *SOracleProvider) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SOracleProvider) GetIamLoginUrl() string {
	return ""
}

func (self *SOracleProvider) GetCloudRegionExternalIdPrefix() string {
	return api.CLOUD_PROVIDER_ORACLE + "/"
}

func (self *SOracleProvider) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	return self.client.GetMetrics(opts)
}
