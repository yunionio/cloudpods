package provider

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/aliyun"
)

type SAliyunProviderFactory struct {
	// providerTable map[string]*SAliyunProvider
}

func (self *SAliyunProviderFactory) GetId() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN
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

	client, err := aliyun.NewAliyunClient(providerId, providerName, account, secret)
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

func (self *SAliyunProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SAliyunProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SAliyunProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SAliyunProvider) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return self.client.GetIHostById(id)
}

func (self *SAliyunProvider) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return self.client.GetIVpcById(id)
}

func (self *SAliyunProvider) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.client.GetIStorageById(id)
}

func (self *SAliyunProvider) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return self.client.GetIStoragecacheById(id)
}

func (self *SAliyunProvider) GetBalance() (float64, error) {
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return 0.0, err
	}
	return balance.AvailableAmount, nil
}
