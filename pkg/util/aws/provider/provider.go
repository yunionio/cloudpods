package provider

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/aws"
)

type SAwsProviderFactory struct {
}

func (self *SAwsProviderFactory) GetId() string {
	return aws.CLOUD_PROVIDER_AWS
}

func (self *SAwsProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := aws.NewAwsClient(providerId, providerName, account, secret)
	if err != nil {
		return nil, err
	}
	return &SAwsProvider{client: client}, nil
}

func init() {
	factory := SAwsProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SAwsProvider struct {
	client *aws.SAwsClient
}

func (self *SAwsProvider) GetId() string {
	return aws.CLOUD_PROVIDER_AWS
}

func (self *SAwsProvider) GetName() string {
	return aws.CLOUD_PROVIDER_AWS_CN
}

func (self *SAwsProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SAwsProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(aws.AWS_API_VERSION), "api_version")
	return info, nil
}

func (self *SAwsProvider) IsPublicCloud() bool {
	return true
}

func (self *SAwsProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SAwsProvider) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return self.client.GetIHostById(id)
}

func (self *SAwsProvider) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return self.client.GetIVpcById(id)
}

func (self *SAwsProvider) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.client.GetIStorageById(id)
}

func (self *SAwsProvider) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return self.client.GetIStoragecacheById(id)
}

func (self *SAwsProvider) GetBalance() (float64, error) {
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return 0.0, err
	}
	return balance.AvailableAmount, nil
}
