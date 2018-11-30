package cloudprovider

import (
	"fmt"

	"errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

var (
	ErrNoSuchProvder = errors.New("no such provider")
)

type ICloudProviderFactory interface {
	GetProvider(providerId, providerName, url, account, secret string) (ICloudProvider, error)
	GetId() string
}

type ICloudProvider interface {
	GetId() string
	GetName() string
	GetSysInfo() (jsonutils.JSONObject, error)
	IsPublicCloud() bool
	IsOnPremiseInfrastructure() bool

	GetIRegions() []ICloudRegion
	GetIRegionById(id string) (ICloudRegion, error)

	GetOnPremiseIRegion() (ICloudRegion, error)

	// GetIHostById(id string) (ICloudHost, error)
	// GetIVpcById(id string) (ICloudVpc, error)
	// GetIStorageById(id string) (ICloudStorage, error)
	// GetIStoragecacheById(id string) (ICloudStoragecache, error)

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

func GetProvider(providerId, providerName, accessUrl, account, secret, provider string) (ICloudProvider, error) {
	factory, ok := providerTable[provider]
	if ok {
		return factory.GetProvider(providerId, providerName, accessUrl, account, secret)
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
