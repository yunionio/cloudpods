package cloudprovider

import (
	"context"
	"fmt"

	"errors"

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
	GetIRegionById(id string) (ICloudRegion, error)

	GetOnPremiseIRegion() (ICloudRegion, error)

	GetBalance() (float64, error)

	GetSubAccounts() ([]SSubAccount, error)
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
		if ! d.IsPublicCloud() {
			providers = append(providers, p)
		}
	}
	return providers
}

func GetOnPremiseProviders() []string {
	providers := make([]string, 0)
	for p, d := range providerTable {
		if ! d.IsPublicCloud() && d.IsOnPremise() {
			providers = append(providers, p)
		}
	}
	return providers
}