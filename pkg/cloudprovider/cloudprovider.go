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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	ErrNoSuchProvder = errors.Error("no such provider")
)

type SCloudaccountCredential struct {
	ProjectName string //OpenStack
	DomainName  string //OpenStack
	Username    string //OpenStack Esxi ZStack
	Password    string //OpenStack Esxi ZStack
	AuthUrl     string //OpenStack ZStack

	AccessKeyId     string //Huawei Aliyun Ucloud Aws
	AccessKeySecret string //Huawei Aliyun Ucloud Aws
	Environment     string //Huawei Azure Aws

	DirectoryId  string //Azure
	ClientId     string //Azure
	ClientSecret string //Azure

	Host string //Esxi
	Port int    //Esxi

	Endpoint string

	AppId     string //Qcloud
	SecretId  string //Qcloud
	SecretKey string //Qcloud

	ClientEmail  string //Google
	ProjectId    string //Google
	PrivateKeyId string //Google
	PrivateKey   string //Google
}

type SCloudaccount struct {
	Account   string
	Secret    string
	AccessUrl string
}

type ICloudProviderFactory interface {
	GetProvider(providerId, providerName, url, account, secret string) (ICloudProvider, error)

	GetClientRC(url, account, secret string) (map[string]string, error)

	GetId() string
	GetName() string

	ValidateChangeBandwidth(instanceId string, bandwidth int64) error
	ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, input SCloudaccountCredential) (SCloudaccount, error)
	ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, input SCloudaccountCredential, cloudaccount string) (SCloudaccount, error)
	GetSupportedBrands() []string

	IsPublicCloud() bool
	IsOnPremise() bool
	IsSupportPrepaidResources() bool
	NeedSyncSkuFromCloud() bool

	IsSupportObjectStorage() bool
	IsSupportComputeEngine() bool
	IsSupportNetworkManage() bool

	IsCloudeventRegional() bool
	GetMaxCloudEventSyncDays() int
	GetMaxCloudEventKeepDays() int
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
	GetAccountId() string

	// region external id 是以provider 做为前缀.因此可以通过该判断条件过滤出同一个provider的regions列表
	// 但是华为云有点特殊一个provider只对应一个region,因此需要进一步指定region名字，才能找到provider对应的region
	GetCloudRegionExternalIdPrefix() string

	GetStorageClasses(regionId string) []string

	GetCapabilities() []string
}

func IsSupportProject(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_PROJECT, prod.GetCapabilities())
}

func IsSupportCompute(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_COMPUTE, prod.GetCapabilities())
}

func IsSupportLoadbalancer(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_LOADBALANCER, prod.GetCapabilities())
}

func IsSupportObjectstore(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_OBJECTSTORE, prod.GetCapabilities())
}

func IsSupportRds(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_RDS, prod.GetCapabilities())
}

func IsSupportElasticCache(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_CACHE, prod.GetCapabilities())
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
		return nil, errors.Wrap(err, "GetProviderFactory")
	}
	return driver.GetProvider(providerId, providerName, accessUrl, account, secret)
}

func GetClientRC(accessUrl, account, secret, provider string) (map[string]string, error) {
	driver, err := GetProviderFactory(provider)
	if err != nil {
		return nil, errors.Wrap(err, "GetProviderFactory")
	}
	return driver.GetClientRC(accessUrl, account, secret)
}

func IsSupported(provider string) bool {
	_, ok := providerTable[provider]
	return ok
}

func IsValidCloudAccount(accessUrl, account, secret, provider string) (string, error) {
	factory, ok := providerTable[provider]
	if ok {
		provider, err := factory.GetProvider("", "", accessUrl, account, secret)
		if err != nil {
			return "", err
		}
		return provider.GetAccountId(), nil
	} else {
		return "", ErrNoSuchProvder
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

type baseProviderFactory struct {
}

func (factory *baseProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (factory *baseProviderFactory) GetSupportedBrands() []string {
	return []string{}
}

func (factory *baseProviderFactory) GetProvider(providerId, providerName, url, username, password string) (ICloudProvider, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented GetProvider")
}

func (factory *baseProviderFactory) IsOnPremise() bool {
	return false
}

func (factory *baseProviderFactory) IsCloudeventRegional() bool {
	return false
}

func (factory *baseProviderFactory) GetMaxCloudEventSyncDays() int {
	return 7
}

func (factory *baseProviderFactory) GetMaxCloudEventKeepDays() int {
	return 7
}

type SPremiseBaseProviderFactory struct {
	baseProviderFactory
}

func (factory *SPremiseBaseProviderFactory) IsPublicCloud() bool {
	return false
}

func (factory *SPremiseBaseProviderFactory) IsSupportPrepaidResources() bool {
	return false
}

func (factory *SPremiseBaseProviderFactory) IsOnPremise() bool {
	return true
}

func (factory *SPremiseBaseProviderFactory) IsSupportObjectStorage() bool {
	return false
}

func (factory *SPremiseBaseProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

func (factory *SPremiseBaseProviderFactory) IsSupportComputeEngine() bool {
	return true
}

func (factory *SPremiseBaseProviderFactory) IsSupportNetworkManage() bool {
	return false
}

type SPublicCloudBaseProviderFactor struct {
	baseProviderFactory
}

func (factory *SPublicCloudBaseProviderFactor) IsPublicCloud() bool {
	return true
}

func (factory *SPublicCloudBaseProviderFactor) IsSupportPrepaidResources() bool {
	return true
}

func (factory *SPublicCloudBaseProviderFactor) IsSupportObjectStorage() bool {
	return true
}

func (factory *SPublicCloudBaseProviderFactor) NeedSyncSkuFromCloud() bool {
	return false
}

func (factory *SPublicCloudBaseProviderFactor) IsSupportComputeEngine() bool {
	return true
}

func (factory *SPublicCloudBaseProviderFactor) IsSupportNetworkManage() bool {
	return true
}

type SPrivateCloudBaseProviderFactor struct {
	baseProviderFactory
}

func (factory *SPrivateCloudBaseProviderFactor) IsPublicCloud() bool {
	return false
}

func (factory *SPrivateCloudBaseProviderFactor) IsSupportPrepaidResources() bool {
	return false
}

func (factory *SPrivateCloudBaseProviderFactor) IsSupportObjectStorage() bool {
	return false
}

func (factory *SPrivateCloudBaseProviderFactor) NeedSyncSkuFromCloud() bool {
	return true
}

func (factory *SPrivateCloudBaseProviderFactor) IsSupportComputeEngine() bool {
	return true
}

func (factory *SPrivateCloudBaseProviderFactor) IsSupportNetworkManage() bool {
	return true
}
