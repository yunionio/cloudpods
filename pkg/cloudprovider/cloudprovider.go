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
	"net/http"

	"yunion.io/x/jsonutils"
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
	// 账号所在的项目 (openstack)
	ProjectName string `json:"project_name"`

	// 账号所在的域 (openstack)
	// default: Default
	DomainName string `json:"domain_name"`

	// 用户名 (openstack, zstack, esxi)
	Username string `json:"username"`

	// 密码 (openstack, zstack, esxi)
	Password string `json:"password"`

	// 认证地址 (openstack,zstack)
	AuthUrl string `json:"auto_url"`

	// 秘钥id (Aliyun, Aws, huawei, ucloud, ctyun, zstack, s3)
	AccessKeyId string `json:"access_key_id"`

	// 秘钥key (Aliyun, Aws, huawei, ucloud, ctyun, zstack, s3)
	AccessKeySecret string `json:"access_key_secret"`

	// 环境 (Azure, Aws, huawei, ctyun, aliyun)
	Environment string `json:"environment"`

	// 目录ID (Azure)
	DirectoryId string `json:"directory_id"`

	// 客户端ID (Azure)
	ClientId string `json:"client_id"`

	// 客户端秘钥 (Azure)
	ClientSecret string `json:"client_secret"`

	// 主机IP (esxi)
	Host string `json:"host"`

	// 主机端口 (esxi)
	Port int `json:"port"`

	// 端点 (s3) 或 Apsara(飞天)
	Endpoint string `json:"endpoint"`

	// app id (Qcloud)
	AppId string `json:"app_id"`

	//秘钥ID (Qcloud)
	SecretId string `json:"secret_id"`

	//秘钥key (Qcloud)
	SecretKey string `json:"secret_key"`

	// Google服务账号email (gcp)
	GCPClientEmail string `json:"gcp_client_email"`
	// Google服务账号project id (gcp)
	GCPProjectId string `json:"gcp_project_id"`
	// Google服务账号秘钥id (gcp)
	GCPPrivateKeyId string `json:"gcp_private_key_id"`
	// Google服务账号秘钥 (gcp)
	GCPPrivateKey string `json:"gcp_private_key"`

	// 阿里云专有云Endpoints
	*SApsaraEndpoints
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

	AccountId string

	SApsaraEndpoints

	ProxyFunc httputils.TransportProxyFunc
}

func (cp *ProviderConfig) AdaptiveTimeoutHttpClient() *http.Client {
	client := httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(client, cp.ProxyFunc)
	return client
}

type SProviderInfo struct {
	Name    string
	Url     string
	Account string
	Secret  string
}

type ICloudProviderFactory interface {
	GetProvider(cfg ProviderConfig) (ICloudProvider, error)

	GetClientRC(SProviderInfo) (map[string]string, error)

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

	IsNeedForceAutoCreateProject() bool

	IsCloudpolicyWithSubscription() bool     // 自定义权限属于订阅级别资源
	IsClouduserpolicyWithSubscription() bool // 绑定用户权限需要指定订阅

	IsSupportCloudIdService() bool
	IsSupportClouduserPolicy() bool
	IsSupportResetClouduserPassword() bool
	GetClouduserMinPolicyCount() int
	IsClouduserNeedInitPolicy() bool
	IsSupportCreateCloudgroup() bool

	IsSystemCloudpolicyUnified() bool // 国内国外权限是否一致

	IsSupportCrossCloudEnvVpcPeering() bool
	IsSupportCrossRegionVpcPeering() bool
	IsSupportVpcPeeringVpcCidrOverlap() bool
	ValidateCrossRegionVpcPeeringBandWidth(bandwidth int) error

	IsSupportModifyRouteTable() bool

	GetSupportedDnsZoneTypes() []TDnsZoneType
	GetSupportedDnsTypes() map[TDnsZoneType][]TDnsType
	GetSupportedDnsPolicyTypes() map[TDnsZoneType][]TDnsPolicyType
	GetSupportedDnsPolicyValues() map[TDnsPolicyType][]TDnsPolicyValue
	GetTTLRange(zoneType TDnsZoneType, productType TDnsProductType) TTlRange

	IsSupportSAMLAuth() bool

	GetAccountIdEqualizer() func(origin, now string) bool
}

type ICloudProvider interface {
	GetFactory() ICloudProviderFactory

	GetSysInfo() (jsonutils.JSONObject, error)
	GetVersion() string
	GetIamLoginUrl() string

	GetIRegions() []ICloudRegion
	GetIProjects() ([]ICloudProject, error)
	CreateIProject(name string) (ICloudProject, error)
	GetIRegionById(id string) (ICloudRegion, error)

	GetOnPremiseIRegion() (ICloudRegion, error)

	GetBalance() (float64, string, error)

	GetSubAccounts() ([]SSubAccount, error)
	GetAccountId() string

	// region external id 是以provider 做为前缀.因此可以通过该判断条件过滤出同一个provider的regions列表
	// 但是华为云有点特殊一个provider只对应一个region,因此需要进一步指定region名字，才能找到provider对应的region
	GetCloudRegionExternalIdPrefix() string

	GetStorageClasses(regionId string) []string
	GetBucketCannedAcls(regionId string) []string
	GetObjectCannedAcls(regionId string) []string

	GetCapabilities() []string
	GetICloudQuotas() ([]ICloudQuota, error)

	IsClouduserSupportPassword() bool
	GetICloudusers() ([]IClouduser, error)
	GetISystemCloudpolicies() ([]ICloudpolicy, error)
	GetICustomCloudpolicies() ([]ICloudpolicy, error)
	GetICloudgroups() ([]ICloudgroup, error)
	GetICloudgroupByName(name string) (ICloudgroup, error)
	CreateICloudgroup(name, desc string) (ICloudgroup, error)
	GetIClouduserByName(name string) (IClouduser, error)
	CreateIClouduser(conf *SClouduserCreateConfig) (IClouduser, error)
	CreateICloudSAMLProvider(opts *SAMLProviderCreateOptions) (ICloudSAMLProvider, error)
	GetICloudSAMLProviders() ([]ICloudSAMLProvider, error)
	GetICloudroles() ([]ICloudrole, error)
	GetICloudroleById(id string) (ICloudrole, error)
	GetICloudroleByName(name string) (ICloudrole, error)
	CreateICloudrole(opts *SRoleCreateOptions) (ICloudrole, error)

	CreateICloudpolicy(opts *SCloudpolicyCreateOptions) (ICloudpolicy, error)

	GetEnrollmentAccounts() ([]SEnrollmentAccount, error)
	CreateSubscription(SubscriptionCreateInput) error

	GetSamlEntityId() string

	GetICloudDnsZones() ([]ICloudDnsZone, error)
	GetICloudDnsZoneById(id string) (ICloudDnsZone, error)
	CreateICloudDnsZone(opts *SDnsZoneCreateOptions) (ICloudDnsZone, error)

	GetICloudInterVpcNetworks() ([]ICloudInterVpcNetwork, error)
	GetICloudInterVpcNetworkById(id string) (ICloudInterVpcNetwork, error)
	CreateICloudInterVpcNetwork(opts *SInterVpcNetworkCreateOptions) (ICloudInterVpcNetwork, error)
}

func IsSupportProject(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_PROJECT, prod.GetCapabilities())
}

func IsSupportDnsZone(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_DNSZONE, prod.GetCapabilities())
}

func IsSupportInterVpcNetwork(prod ICloudProvider) bool {
	return utils.IsInStringArray(CLOUD_CAPABILITY_INTERVPCNETWORK, prod.GetCapabilities())
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

func GetClientRC(name, accessUrl, account, secret, provider string) (map[string]string, error) {
	driver, err := GetProviderFactory(provider)
	if err != nil {
		return nil, errors.Wrap(err, "GetProviderFactory")
	}
	info := SProviderInfo{
		Name:    name,
		Url:     accessUrl,
		Account: account,
		Secret:  secret,
	}
	return driver.GetClientRC(info)
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

func (self *SBaseProvider) GetIamLoginUrl() string {
	return ""
}

func (self *SBaseProvider) IsClouduserSupportPassword() bool {
	return true
}

func (self *SBaseProvider) GetICloudusers() ([]IClouduser, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetICloudgroups() ([]ICloudgroup, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetICloudgroupByName(name string) (ICloudgroup, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) CreateICloudgroup(name, desc string) (ICloudgroup, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetISystemCloudpolicies() ([]ICloudpolicy, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetICustomCloudpolicies() ([]ICloudpolicy, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetIClouduserByName(name string) (IClouduser, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) CreateIClouduser(conf *SClouduserCreateConfig) (IClouduser, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetICloudSAMLProviders() ([]ICloudSAMLProvider, error) {
	return nil, errors.Wrapf(ErrNotImplemented, "GetICloudSAMLProviders")
}

func (self *SBaseProvider) GetICloudroles() ([]ICloudrole, error) {
	return nil, errors.Wrapf(ErrNotImplemented, "GetICloudroles")
}

func (self *SBaseProvider) GetICloudroleById(id string) (ICloudrole, error) {
	return nil, errors.Wrapf(ErrNotImplemented, "GetICloudroleById")
}

func (self *SBaseProvider) GetICloudroleByName(name string) (ICloudrole, error) {
	return nil, errors.Wrapf(ErrNotImplemented, "GetICloudroleByName")
}

func (self *SBaseProvider) CreateICloudrole(opts *SRoleCreateOptions) (ICloudrole, error) {
	return nil, errors.Wrapf(ErrNotImplemented, "CreateICloudrole")
}

func (self *SBaseProvider) CreateICloudSAMLProvider(opts *SAMLProviderCreateOptions) (ICloudSAMLProvider, error) {
	return nil, errors.Wrapf(ErrNotImplemented, "CreateICloudSAMLProvider")
}

func (self *SBaseProvider) CreateICloudpolicy(opts *SCloudpolicyCreateOptions) (ICloudpolicy, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetEnrollmentAccounts() ([]SEnrollmentAccount, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) CreateSubscription(SubscriptionCreateInput) error {
	return ErrNotImplemented
}

func (self *SBaseProvider) GetICloudDnsZones() ([]ICloudDnsZone, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetICloudDnsZoneById(id string) (ICloudDnsZone, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) CreateICloudDnsZone(opts *SDnsZoneCreateOptions) (ICloudDnsZone, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetCloudRegionExternalIdPrefix() string {
	return self.factory.GetId()
}

func (self *SBaseProvider) CreateIProject(name string) (ICloudProject, error) {
	return nil, ErrNotImplemented
}

func (self *SBaseProvider) GetSamlEntityId() string {
	return ""
}

func (self *SBaseProvider) GetSamlSpInitiatedLoginUrl(idpName string) string {
	return ""
}

func (self *SBaseProvider) GetICloudInterVpcNetworks() ([]ICloudInterVpcNetwork, error) {
	return nil, ErrNotImplemented
}
func (self *SBaseProvider) GetICloudInterVpcNetworkById(id string) (ICloudInterVpcNetwork, error) {
	return nil, ErrNotImplemented
}
func (self *SBaseProvider) CreateICloudInterVpcNetwork(opts *SInterVpcNetworkCreateOptions) (ICloudInterVpcNetwork, error) {
	return nil, ErrNotImplemented
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

func GetSupportCloudgroupProviders() []string {
	providers := []string{}
	for p, d := range providerTable {
		if d.IsSupportCreateCloudgroup() {
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

func GetSupportCloudIdProvider() []string {
	providers := []string{}
	for p, d := range providerTable {
		if d.IsSupportCloudIdService() {
			providers = append(providers, p)
		}
	}
	return providers
}

func GetClouduserpolicyWithSubscriptionProviders() []string {
	providers := []string{}
	for p, d := range providerTable {
		if d.IsClouduserpolicyWithSubscription() {
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

func (factory *baseProviderFactory) IsSupportSAMLAuth() bool {
	return false
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

func (factory *baseProviderFactory) IsNeedForceAutoCreateProject() bool {
	return false
}

func (factory *baseProviderFactory) IsCloudpolicyWithSubscription() bool {
	return false
}

func (factory *baseProviderFactory) IsClouduserpolicyWithSubscription() bool {
	return false
}

func (factory *baseProviderFactory) IsSupportCloudIdService() bool {
	return false
}

func (factory *baseProviderFactory) IsSupportClouduserPolicy() bool {
	return true
}

func (factory *baseProviderFactory) IsSupportResetClouduserPassword() bool {
	return true
}

func (factory *baseProviderFactory) IsClouduserNeedInitPolicy() bool {
	return false
}

func (factory *baseProviderFactory) GetClouduserMinPolicyCount() int {
	// unlimited
	return -1
}

func (factory *baseProviderFactory) IsSupportCreateCloudgroup() bool {
	return false
}

func (factory *baseProviderFactory) IsSystemCloudpolicyUnified() bool {
	return true
}

func (factory *baseProviderFactory) IsSupportCrossCloudEnvVpcPeering() bool {
	return false
}

func (factory *baseProviderFactory) IsSupportCrossRegionVpcPeering() bool {
	return false
}

func (factory *baseProviderFactory) IsSupportVpcPeeringVpcCidrOverlap() bool {
	return false
}

func (factory *baseProviderFactory) ValidateCrossRegionVpcPeeringBandWidth(bandwidth int) error {
	return nil
}

func (factory *baseProviderFactory) IsSupportModifyRouteTable() bool {
	return false
}

func (factory *baseProviderFactory) GetSupportedDnsZoneTypes() []TDnsZoneType {
	return []TDnsZoneType{}
}

func (factory *baseProviderFactory) GetSupportedDnsTypes() map[TDnsZoneType][]TDnsType {
	return map[TDnsZoneType][]TDnsType{}
}

func (factory *baseProviderFactory) GetSupportedDnsPolicyTypes() map[TDnsZoneType][]TDnsPolicyType {
	return map[TDnsZoneType][]TDnsPolicyType{}
}

func (factory *baseProviderFactory) GetSupportedDnsPolicyValues() map[TDnsPolicyType][]TDnsPolicyValue {
	return map[TDnsPolicyType][]TDnsPolicyValue{}
}

func (factory *baseProviderFactory) GetTTLRange(zoneType TDnsZoneType, productType TDnsProductType) TTlRange {
	return TTlRange{}
}

func (factory *baseProviderFactory) GetAccountIdEqualizer() func(origin, now string) bool {
	return func(origin, now string) bool {
		if len(now) > 0 && now != origin {
			return false
		}
		return true
	}
}

type SDnsCapability struct {
	ZoneTypes    []TDnsZoneType
	DnsTypes     map[TDnsZoneType][]TDnsType
	PolicyTypes  map[TDnsZoneType][]TDnsPolicyType
	PolicyValues map[TDnsPolicyType][]TDnsPolicyValue
}

func GetDnsCapabilities() map[string]SDnsCapability {
	capabilities := map[string]SDnsCapability{}
	for provider, driver := range providerTable {
		capabilities[provider] = SDnsCapability{
			ZoneTypes:    driver.GetSupportedDnsZoneTypes(),
			DnsTypes:     driver.GetSupportedDnsTypes(),
			PolicyTypes:  driver.GetSupportedDnsPolicyTypes(),
			PolicyValues: driver.GetSupportedDnsPolicyValues(),
		}
	}
	return capabilities
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

type SPublicCloudBaseProviderFactory struct {
	baseProviderFactory
}

func (factory *SPublicCloudBaseProviderFactory) IsPublicCloud() bool {
	return true
}

func (factory *SPublicCloudBaseProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (factory *SPublicCloudBaseProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

type SPrivateCloudBaseProviderFactory struct {
	baseProviderFactory
}

func (factory *SPrivateCloudBaseProviderFactory) IsPublicCloud() bool {
	return false
}

func (factory *SPrivateCloudBaseProviderFactory) IsSupportPrepaidResources() bool {
	return false
}

func (factory *SPrivateCloudBaseProviderFactory) NeedSyncSkuFromCloud() bool {
	return true
}
