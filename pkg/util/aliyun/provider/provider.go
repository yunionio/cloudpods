package provider

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/aliyun"
)

type SAliyunProviderFactory struct {
	// providerTable map[string]*SAliyunProvider
}

func (self *SAliyunProviderFactory) GetId() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN
}

func (self *SAliyunProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (self *SAliyunProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	accessKeyID, _ := data.GetString("access_key_id")
	if len(accessKeyID) == 0 {
		return httperrors.NewMissingParameterError("access_key_id")
	}
	accessKeySecret, _ := data.GetString("access_key_secret")
	if len(accessKeySecret) == 0 {
		return httperrors.NewMissingParameterError("access_key_secret")
	}
	data.Set("account", jsonutils.NewString(accessKeyID))
	data.Set("secret", jsonutils.NewString(accessKeySecret))
	return nil
}

func (self *SAliyunProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {

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

func (self *SAliyunProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	/*	provider, ok := self.providerTable[providerId]
		if ok {
			err := provider.client.UpdateAccount(account, secret)
			if err != nil {
				return nil, err
			} else {
				return provider, nil
			}
		}
		client, err := aliyun.NewAliyunClient(providerId, providerName, account, secret)
		if err != nil {
			return nil, err
		}
		self.providerTable[providerId] = &SAliyunProvider{client: client}
		return self.providerTable[providerId], nil
	*/

	client, err := aliyun.NewAliyunClient(providerId, providerName, account, secret, false)
	if err != nil {
		return nil, err
	}
	return &SAliyunProvider{client: client}, nil
}

func init() {
	factory := SAliyunProviderFactory{
		// providerTable: make(map[string]*SAliyunProvider),
	}
	cloudprovider.RegisterFactory(&factory)
}

type SAliyunProvider struct {
	client *aliyun.SAliyunClient
}

func (self *SAliyunProvider) IsPublicCloud() bool {
	return true
}

func (self *SAliyunProvider) IsOnPremiseInfrastructure() bool {
	return false
}

func (self *SAliyunProvider) GetId() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN
}

func (self *SAliyunProvider) GetName() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN_CN
}

func (self *SAliyunProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(aliyun.ALIYUN_API_VERSION), "api_version")
	return info, nil
}

func (self *SAliyunProvider) GetVersion() string {
	return aliyun.ALIYUN_API_VERSION
}

func (self *SAliyunProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SAliyunProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SAliyunProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SAliyunProvider) GetBalance() (float64, error) {
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return 0.0, err
	}
	return balance.AvailableAmount, nil
}

func (self *SAliyunProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}
