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

package cloudprovider

import (
	"context"
	"errors"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	ErrNoSuchProvder = errors.New("no such provider")
)

type SCloudaccount struct {
	Account string
	Secret  string
}

type ICloudProviderFactory interface {
	GetProvider(providerId, providerName, url, account, secret string) (ICloudProvider, error)

	GetId() string
	GetName() string

	ValidateChangeBandwidth(instanceId string, bandwidth int64) error
	ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error
	ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*SCloudaccount, error)

	IsPublicCloud() bool
	IsOnPremise() bool
	IsSupportPrepaidResources() bool
	NeedSyncSkuFromCloud() bool
}

type ICloudProvider interface {
	GetFactory() ICloudProviderFactory

	GetSysInfo() (jsonutils.JSONObject, error)
	GetVersion() string

	GetIRegions() []ICloudRegion
	GetIProjects() ([]ICloudProject, error)
	GetIRegionById(id string) (ICloudRegion, error)

	GetOnPremiseIRegion() (ICloudRegion, error)

	GetBalance() (float64, string, error)

	GetSubAccounts() ([]SSubAccount, error)

	// region external id 是以provider 做为前缀.因此可以通过该判断条件过滤出同一个provider的regions列表
	// 但是华为云有点特殊一个provider只对应一个region,因此需要进一步指定region名字，才能找到provider对应的region
	GetCloudRegionExternalIdPrefix() string
}

var providerTable map[string]ICloudProviderFactory

func init() {
	providerTable = make(map[string]ICloudProviderFactory)
}

func RegisterFactory(factory ICloudProviderFactory) {
	providerTable[factory.GetId()] = factory
}

func GetProviderFactory(provider string) (ICloudProviderFactory, error) {
	factory, ok := providerTable[provider]
	if ok {
		return factory, nil
	}
	log.Errorf("Provider %s not registerd", provider)
	return nil, fmt.Errorf("No such provider %s", provider)
}

func GetRegistedProviderIds() []string {
	providers := []string{}
	for id := range providerTable {
		providers = append(providers, id)
	}
	return providers
}

func GetProvider(providerId, providerName, accessUrl, account, secret, provider string) (ICloudProvider, error) {
	driver, err := GetProviderFactory(provider)
	if err != nil {
		return nil, err
	}
	return driver.GetProvider(providerId, providerName, accessUrl, account, secret)
}

func IsSupported(provider string) bool {
	_, ok := providerTable[provider]
	return ok
}

func IsValidCloudAccount(accessUrl, account, secret, provider string) error {
	factory, ok := providerTable[provider]
	if ok {
		_, err := factory.GetProvider("", "", accessUrl, account, secret)
		return err
	} else {
		return ErrNoSuchProvder
	}
}

type SBaseProvider struct {
	factory ICloudProviderFactory
}

func (provider *SBaseProvider) GetFactory() ICloudProviderFactory {
	return provider.factory
}

func (self *SBaseProvider) GetOnPremiseIRegion() (ICloudRegion, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetCloudRegionExternalIdPrefix() string {
	return self.factory.GetId()
}

func NewBaseProvider(factory ICloudProviderFactory) SBaseProvider {
	return SBaseProvider{factory: factory}
}

func GetPublicProviders() []string {
	providers := make([]string, 0)
	for p, d := range providerTable {
		if d.IsPublicCloud() {
			providers = append(providers, p)
		}
	}
	return providers
}

func GetPrivateProviders() []string {
	providers := make([]string, 0)
	for p, d := range providerTable {
		if !d.IsPublicCloud() && !d.IsOnPremise() {
			providers = append(providers, p)
		}
	}
	return providers
}

func GetOnPremiseProviders() []string {
	providers := make([]string, 0)
	for p, d := range providerTable {
		if !d.IsPublicCloud() && d.IsOnPremise() {
			providers = append(providers, p)
		}
	}
	return providers
}
