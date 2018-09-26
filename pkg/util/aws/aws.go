package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

const (
	CLOUD_PROVIDER_AWS    = models.CLOUD_PROVIDER_AWS
	CLOUD_PROVIDER_AWS_CN = "AWS"
)

type SAwsClient struct {
	providerId   string
	providerName string
	accessKey    string
	secret       string
	iregions     []cloudprovider.ICloudRegion
}

func (self *SAwsClient) fetchRegions() error {
	panic("implement me")
}

func (self *SAwsClient) GetRegions() error {
	panic("implement me")
}

func (self *SAwsClient) UpdateAccount(tenantId, secret, envName string) error {
	return nil
}

func (self *SAwsClient) GetIRegions() []cloudprovider.ICloudRegion {
	panic("implement me")
}

func (self *SAwsClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	panic("implement me")
}

func (self *SAwsClient) GetRegion(regionId string) *SRegion {
	return nil
}

func (self *SAwsClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	panic("implement me")
}

func (self *SAwsClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	panic("implement me")
}

func (self *SAwsClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	panic("implement me")
}

func (self *SAwsClient) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	panic("implement me")
}

type SAccountBalance struct {
	AvailableAmount     float64
	AvailableCashAmount float64
	CreditAmount        float64
	MybankCreditAmount  float64
	Currency            string
}

func (self *SAwsClient) QueryAccountBalance() (*SAccountBalance, error) {
	panic("implement me")
}

func NewAwsClient(providerId string, providerName string, accessKey string, secret string) (*SAwsClient, error) {
	client := SAwsClient{providerId: providerId, providerName: providerName, accessKey: accessKey, secret: secret}
	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}
	return &client, nil
}
