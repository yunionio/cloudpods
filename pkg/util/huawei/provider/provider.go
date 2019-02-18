package provider

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/huawei"
)

type SHuaweiProviderFactory struct {
}

func (self *SHuaweiProviderFactory) GetId() string {
	return huawei.CLOUD_PROVIDER_HUAWEI
}

func (self *SHuaweiProviderFactory) GetName() string {
	return huawei.CLOUD_PROVIDER_HUAWEI_CN
}

func (self *SHuaweiProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (self *SHuaweiProviderFactory) IsPublicCloud() bool {
	return true
}

func (self *SHuaweiProviderFactory) IsOnPremise() bool {
	return false
}

func (self *SHuaweiProviderFactory) IsSupportPrepaidResources() bool {
	return true
}

func (self *SHuaweiProviderFactory) NeedSyncSkuFromCloud() bool {
	return false
}

func (self *SHuaweiProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	accessKeyID, _ := data.GetString("access_key_id")
	if len(accessKeyID) == 0 {
		return httperrors.NewMissingParameterError("access_key_id")
	}
	accessKeySecret, _ := data.GetString("access_key_secret")
	if len(accessKeySecret) == 0 {
		return httperrors.NewMissingParameterError("access_key_secret")
	}
	environment, _ := data.GetString("environment")
	if len(environment) == 0 {
		return httperrors.NewMissingParameterError("environment")
	}
	data.Set("account", jsonutils.NewString(accessKeyID))
	data.Set("secret", jsonutils.NewString(accessKeySecret))
	data.Set("access_url", jsonutils.NewString(environment))
	return nil
}

func (self *SHuaweiProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	accessKeyID, _ := data.GetString("access_key_id")
	if len(accessKeyID) == 0 {
		return nil, httperrors.NewMissingParameterError("access_key_id")
	}
	accessKeySecret, _ := data.GetString("access_key_secret")
	if len(accessKeySecret) == 0 {
		return nil, httperrors.NewMissingParameterError("access_key_secret")
	}
	account := &cloudprovider.SCloudaccount{
		Account: accessKeyID,
		Secret:  accessKeySecret,
	}
	return account, nil
}

func (self *SHuaweiProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := huawei.NewHuaweiClient(providerId, providerName, url, account, secret, false)
	if err != nil {
		return nil, err
	}
	return &SHuaweiProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func init() {
	factory := SHuaweiProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SHuaweiProvider struct {
	cloudprovider.SBaseProvider
	client *huawei.SHuaweiClient
}

func (self *SHuaweiProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (self *SHuaweiProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(huawei.HUAWEI_API_VERSION), "api_version")
	return info, nil
}

func (self *SHuaweiProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SHuaweiProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SHuaweiProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHuaweiProvider) GetBalance() (float64, error) {
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return 0.0, err
	}
	return balance.AvailableAmount, nil
}

func (self *SHuaweiProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}
