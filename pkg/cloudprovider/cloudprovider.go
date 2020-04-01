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
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	ErrNoSuchProvder = errors.Error("no such provider")
)

type SCloudaccountCredential struct {
	// 账号所在的项目
	ProjectName string `json:"project_name"`

	// 账号所在的域
	// default: Default
	DomainName string `json:"domain_name"`

	// 用户名
	Username string `json:"username"`

	// 密码
	Password string `json:"password"`

	// 认证地址
	AuthUrl string `json:"auto_url"`

	// 秘钥id
	AccessKeyId string `json:"access_key_id"`

	// 秘钥key
	AccessKeySecret string `json:"access_key_secret"`

	// 环境
	Environment string `json:"environment"`

	// 目录ID
	DirectoryId string `json:"directory_id"`

	// 客户端ID
	ClientId string `json:"client_id"`

	// 客户端秘钥
	ClientSecret string `json:"client_secret"`

	// 主机IP
	Host string `json:"host"`

	// 主机端口
	Port int `json:"port"`

	// 端点
	Endpoint string `json:"endpoint"`

	// app id
	AppId string `json:"app_id"`

	//秘钥ID
	SecretId string `json:"secret_id"`

	//秘钥key
	SecretKey string `json:"secret_key"`

	// Google服务账号email
	ClientEmail string `json:"client_email"`
	// Google服务账号project id
	ProjectId string `json:"project_id"`
	// Google服务账号秘钥id
	PrivateKeyId string `json:"private_key_id"`
	// Google服务账号秘钥
	PrivateKey string `json:"private_key"`
}

type SCloudaccount struct {
	// 账号信息，各个平台字段不尽相同，以下是各个平台账号创建所需要的字段
	//
	//
	//
	// | 云平台		|字段				| 翻译				| 是否必传	| 默认值	| 可否更新	| 获取方式	|
	// | ------ 	|------				| ------			| ---------	| --------	|--------	|--------	|
	// |Aliyun	 	|access_key_id		|秘钥ID				| 是		|			|	是		|			|
	// |Aliyun		|access_key_secret	|秘钥Key			| 是		|			|	是		|			|
	// |Qcloud	 	|app_id				|APP ID				| 是		|			|	否		|			|
	// |Qcloud		|secret_id			|秘钥ID				| 是		|			|	是		|			|
	// |Qcloud		|secret_key			|秘钥Key			| 是		|			|	是		|			|
	// |OpenStack	|project_name		|用户所在项目 		| 是		|			|	是		|			|
	// |OpenStack	|username			|用户名				| 是		|			|	是		|			|
	// |OpenStack	|password			|用户密码			| 是		|			|	是		|			|
	// |OpenStack	|auth_url			|认证地址			| 是		|			|	否		|			|
	// |OpenStack	|domain_name		|用户所在的域		| 否		|Default	|	是		|			|
	// |VMware		|username			|用户名				| 是		|			|	是		|			|
	// |VMware		|password			|密码				| 是		|			|	是		|			|
	// |VMware		|host				|主机IP或域名		| 是		|			|	否		|			|
	// |VMware		|port				|主机端口			| 否		|443		|	否		|			|
	// |Azure		|directory_id		|目录ID				| 是		|			|	否		|			|
	// |Azure		|environment		|区域				| 是		|			|	否		|			|
	// |Azure		|client_id			|客户端ID			| 是		|			|	是		|			|
	// |Azure		|client_secret		|客户端密码			| 是		|			|	是		|			|
	// |Huawei		|access_key_id		|秘钥ID				| 是		|			|	是		|			|
	// |Huawei		|access_key_secret	|秘钥				| 是		|			|	是		|			|
	// |Huawei		|environment		|区域				| 是		|			|	否		|			|
	// |Aws			|access_key_id		|秘钥ID				| 是		|			|	是		|			|
	// |Aws			|access_key_secret	|秘钥				| 是		|			|	是		|			|
	// |Aws			|environment		|区域				| 是		|			|	否		|			|
	// |Ucloud		|access_key_id		|秘钥ID				| 是		|			|	是		|			|
	// |Ucloud		|access_key_secret	|秘钥				| 是		|			|	是		|			|
	// |Google		|project_id			|项目ID				| 是		|			|	否		|			|
	// |Google		|client_email		|客户端email		| 是		|			|	否		|			|
	// |Google		|private_key_id		|秘钥ID				| 是		|			|	是		|			|
	// |Google		|private_key		|秘钥Key			| 是		|			|	是		|			|
	Account string `json:"account"`

	// swagger:ignore
	Secret string

	// 认证地址
	AccessUrl string `json:"access_url"`
}

type ProviderConfig struct {
	// Id, Name are properties of Cloudprovider object
	Id   string
	Name string

	// Vendor are names like Aliyun, OpenStack, etc.
	Vendor  string
	URL     string
	Account string
	Secret  string

	ProxyFunc httputils.TransportProxyFunc
}

type ICloudProviderFactory interface {
	GetProvider(cfg ProviderConfig) (ICloudProvider, error)

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
	GetICloudQuotas() ([]ICloudQuota, error)
	GetICloudPolicyDefinitions() ([]ICloudPolicyDefinition, error)
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

func GetProvider(cfg ProviderConfig) (ICloudProvider, error) {
	driver, err := GetProviderFactory(cfg.Vendor)
	if err != nil {
		return nil, errors.Wrap(err, "GetProviderFactory")
	}
	return driver.GetProvider(cfg)
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

func IsValidCloudAccount(cfg ProviderConfig) (string, error) {
	factory, ok := providerTable[cfg.Vendor]
	if ok {
		provider, err := factory.GetProvider(cfg)
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

func (self *SBaseProvider) GetICloudQuotas() ([]ICloudQuota, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetICloudPolicyDefinitions() ([]ICloudPolicyDefinition, error) {
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

func GetProviderCloudEnv(provider string) string {
	p, err := GetProviderFactory(provider)
	if err != nil {
		return ""
	}
	if p.IsPublicCloud() {
		return CLOUD_ENV_PUBLIC_CLOUD
	}
	if p.IsOnPremise() {
		return CLOUD_ENV_ON_PREMISE
	}
	return CLOUD_ENV_PRIVATE_CLOUD
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

func (factory *SPremiseBaseProviderFactory) NeedSyncSkuFromCloud() bool {
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

func (factory *SPublicCloudBaseProviderFactor) NeedSyncSkuFromCloud() bool {
	return false
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

func (factory *SPrivateCloudBaseProviderFactor) NeedSyncSkuFromCloud() bool {
	return true
}
