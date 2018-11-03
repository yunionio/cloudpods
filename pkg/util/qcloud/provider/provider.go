package provider

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/qcloud"
)

type SQcloudProviderFactory struct {
	// providerTable map[string]*SQcloudProvider
}

func (self *SQcloudProviderFactory) GetId() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := qcloud.NewQcloudClient(providerId, providerName, account, secret)
	if err != nil {
		return nil, err
	}
	return &SQcloudProvider{client: client}, nil
}

func init() {
	factory := SQcloudProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SQcloudProvider struct {
	client *qcloud.SQcloudClient
}

func (self *SQcloudProvider) IsPublicCloud() bool {
	return true
}

func (self *SQcloudProvider) GetId() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudProvider) GetName() string {
	return qcloud.CLOUD_PROVIDER_QCLOUD_CN
}

func (self *SQcloudProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(qcloud.QCLOUD_API_VERSION), "api_version")
	return info, nil
}

func (self *SQcloudProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SQcloudProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SQcloudProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SQcloudProvider) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return self.client.GetIHostById(id)
}

func (self *SQcloudProvider) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return self.client.GetIVpcById(id)
}

func (self *SQcloudProvider) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.client.GetIStorageById(id)
}

func (self *SQcloudProvider) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return self.client.GetIStoragecacheById(id)
}

func (self *SQcloudProvider) GetBalance() (float64, error) {
	return 0.0, nil
}
