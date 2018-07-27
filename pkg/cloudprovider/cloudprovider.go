package cloudprovider

import (
	"fmt"

	"github.com/yunionio/jsonutils"
)

type ICloudProviderFactory interface {
	GetProvider(providerId, url, account, secret string) (ICloudProvider, error)
	GetId() string
}

type ICloudProvider interface {
	GetId() string
	GetName() string
	GetIRegions() []ICloudRegion
	GetSysInfo() (jsonutils.JSONObject, error)
	IsPublicCloud() bool

	GetIRegionById(id string) (ICloudRegion, error)
}

var providerTable map[string]ICloudProviderFactory

func init() {
	providerTable = make(map[string]ICloudProviderFactory)
}

func RegisterFactory(factory ICloudProviderFactory) {
	providerTable[factory.GetId()] = factory
}

func GetProvider(providerId, accessUrl, account, secret, provider string) (ICloudProvider, error) {
	factory, ok := providerTable[provider]
	if ok {
		return factory.GetProvider(providerId, accessUrl, account, secret)
	}
	return nil, fmt.Errorf("No such provider %s", provider)
}

func IsSupported(provider string) bool {
	_, ok := providerTable[provider]
	return ok
}
