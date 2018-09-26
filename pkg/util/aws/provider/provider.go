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
	panic("implement me")
}

func init() {
	factory := SAwsProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SAwsProvider struct {
	client *aws.SAwsClient
}

func (self *SAwsProvider) GetId() string {
	panic("implement me")
}

func (self *SAwsProvider) GetName() string {
	panic("implement me")
}

func (self *SAwsProvider) GetIRegions() []cloudprovider.ICloudRegion {
	panic("implement me")
}

func (self *SAwsProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *SAwsProvider) IsPublicCloud() bool {
	panic("implement me")
}

func (self *SAwsProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	panic("implement me")
}

func (self *SAwsProvider) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	panic("implement me")
}

func (self *SAwsProvider) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	panic("implement me")
}

func (self *SAwsProvider) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	panic("implement me")
}

func (self *SAwsProvider) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	panic("implement me")
}

func (self *SAwsProvider) GetBalance() (float64, error) {
	panic("implement me")
}
