package provider

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/azure"
	// "yunion.io/x/log"
)

type SAzureProviderFactory struct {
}

func (self *SAzureProviderFactory) GetId() string {
	return azure.CLOUD_PROVIDER_AZURE
}

func (self *SAzureProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", azure.CLOUD_PROVIDER_AZURE)
}

func (self *SAzureProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	if client, err := azure.NewAzureClient(providerId, providerName, account, secret, url); err != nil {
		return nil, err
	} else {
		return &SAzureProvider{client: client}, nil
	}
}

func init() {
	factory := SAzureProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SAzureProvider struct {
	client *azure.SAzureClient
}

func (self *SAzureProvider) IsPublicCloud() bool {
	return true
}

func (self *SAzureProvider) IsOnPremiseInfrastructure() bool {
	return false
}

func (self *SAzureProvider) GetId() string {
	return azure.CLOUD_PROVIDER_AZURE
}

func (self *SAzureProvider) GetName() string {
	return azure.CLOUD_PROVIDER_AZURE_CN
}

func (self *SAzureProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(azure.AZURE_API_VERSION), "api_version")
	return info, nil
}

func (self *SAzureProvider) GetVersion() string {
	return azure.AZURE_API_VERSION
}

func (self *SAzureProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SAzureProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SAzureProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SAzureProvider) GetBalance() (float64, error) {
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return 0.0, err
	}
	return balance.AvailableAmount, nil
}

func (self *SAzureProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SAzureProvider) SupportPrepaidResources() bool {
	return true
}
