package provider

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/qcloud"
)

type STencentProviderFactory struct {
	// providerTable map[string]*SAliyunProvider
}

func (self *STencentProviderFactory) GetId() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD
}

func (self *STencentProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := qcloud.NewQcloudClient(providerId, providerName, account, secret)
	if err != nil {
		return nil, err
	}
	return &STencentProvider{client: client}, nil
}

func init() {
	factory := STencentProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type STencentProvider struct {
	client *qcloud.SQcloudClient
}

func (self *STencentProvider) IsPublicCloud() bool {
	return true
}

func (self *STencentProvider) GetId() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD
}

func (self *STencentProvider) GetName() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD_CN
}

func (self *STencentProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(qcloud.QCLOUD_API_VERSION), "api_version")
	return info, nil
}

func (self *STencentProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *STencentProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *STencentProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *STencentProvider) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return self.client.GetIHostById(id)
}

func (self *STencentProvider) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return self.client.GetIVpcById(id)
}

func (self *STencentProvider) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.client.GetIStorageById(id)
}

func (self *STencentProvider) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return self.client.GetIStoragecacheById(id)
}

func (self *STencentProvider) GetBalance() (float64, error) {
	return 0.0, nil
}
