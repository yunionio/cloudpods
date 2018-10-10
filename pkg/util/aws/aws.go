package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"

	sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/jsonutils"
)

const (
	CLOUD_PROVIDER_AWS    = models.CLOUD_PROVIDER_AWS
	CLOUD_PROVIDER_AWS_CN = "AWS"

	AWS_DEFAULT_REGION = "cn-hangzhou"
	AWS_API_VERSION = "2014-05-26"
)

type SAwsClient struct {
	providerId   string
	providerName string
	accessKey    string
	secret       string
	iregions     []cloudprovider.ICloudRegion
}

func (self *SAwsClient) getDefaultSession() (*session.Session, error) {
	return session.NewSession(&sdk.Config{
		Region: sdk.String(AWS_DEFAULT_REGION),
		Credentials: credentials.NewStaticCredentials(self.accessKey, self.secret, ""),
	})
}

func (self *SAwsClient) fetchRegions() error {
	s, err := self.getDefaultSession()
	if err != nil {
		return err
	}
	svc := ec2.New(s)
	result, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return err
	}
	sregions := make([]SRegion, 0)
	regions := jsonutils.ParseString("")
	regions.Unmarshal(&sregions, "Regions")
	return nil
}

func (self *SAwsClient) GetRegions() error {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeRegions
	panic("implement me")
}

func (self *SAwsClient) UpdateAccount(accessKey, secret string) error {
	return nil
}

func (self *SAwsClient) GetRegion(regionId string) *SRegion {
	return nil
}

func (self *SAwsClient) GetIRegions() []cloudprovider.ICloudRegion {
	panic("implement me")
}

func (self *SAwsClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	panic("implement me")
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
