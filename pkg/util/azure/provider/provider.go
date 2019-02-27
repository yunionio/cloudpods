package provider

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/azure"
	// "yunion.io/x/log"
)

type SAzureProviderFactory struct {
}

func (self *SAzureProviderFactory) GetId() string {
	return azure.CLOUD_PROVIDER_AZURE
}

func (self *SAzureProviderFactory) GetName() string {
	return azure.CLOUD_PROVIDER_AZURE_CN
}

func (self *SAzureProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", azure.CLOUD_PROVIDER_AZURE)
}

func (self *SAzureProviderFactory) IsPublicCloud() bool {
	return true
}

func (self *SAzureProviderFactory) IsOnPremise() bool {
	return false
}

func (self *SAzureProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (self *SAzureProviderFactory) IsProjectRegional() bool {
	return false
}

func (self *SAzureProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

func (self *SAzureProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	directoryID, _ := data.GetString("directory_id")
	if len(directoryID) == 0 {
		return httperrors.NewMissingParameterError("directory_id")
	}
	clientID, _ := data.GetString("client_id")
	if len(clientID) == 0 {
		return httperrors.NewMissingParameterError("client_id")
	}
	clientSecret, _ := data.GetString("client_secret")
	if len(clientSecret) == 0 {
		return httperrors.NewMissingParameterError("client_secret")
	}
	environment, _ := data.GetString("environment")
	if len(environment) == 0 {
		return httperrors.NewMissingParameterError("environment")
	}
	data.Set("account", jsonutils.NewString(directoryID))
	data.Set("secret", jsonutils.NewString(fmt.Sprintf("%s/%s", clientID, clientSecret)))
	data.Set("access_url", jsonutils.NewString(environment))
	return nil
}

func (self *SAzureProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	clientID, _ := data.GetString("client_id")
	if len(clientID) == 0 {
		return nil, httperrors.NewMissingParameterError("client_id")
	}
	clientSecret, _ := data.GetString("client_secret")
	if len(clientSecret) == 0 {
		return nil, httperrors.NewMissingParameterError("client_secret")
	}
	account := &cloudprovider.SCloudaccount{
		Account: cloudaccount,
		Secret:  fmt.Sprintf("%s/%s", clientID, clientSecret),
	}
	return account, nil
}

func (self *SAzureProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	if client, err := azure.NewAzureClient(providerId, providerName, account, secret, url); err != nil {
		return nil, err
	} else {
		return &SAzureProvider{
			SBaseProvider: cloudprovider.NewBaseProvider(self),
			client:        client,
		}, nil
	}
}

func init() {
	factory := SAzureProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SAzureProvider struct {
	cloudprovider.SBaseProvider
	client *azure.SAzureClient
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
