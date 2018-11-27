package aws

import (
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"

	sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	CLOUD_PROVIDER_AWS    = models.CLOUD_PROVIDER_AWS
	CLOUD_PROVIDER_AWS_CN = "AWS"

	AWS_INTERNATIONAL_DEFAULT_REGION = "us-west-1"
	AWS_CHINA_DEFAULT_REGION         = "cn-north-1"
	AWS_API_VERSION                  = "2018-10-10"
)

type SAwsClient struct {
	providerId   string
	providerName string
	accessUrl    string // 服务区域 ChinaCloud | InternationalCloud
	accessKey    string
	secret       string
	iregions     []cloudprovider.ICloudRegion
}

func NewAwsClient(providerId string, providerName string, accessUrl string, accessKey string, secret string) (*SAwsClient, error) {
	client := SAwsClient{providerId: providerId, providerName: providerName, accessUrl: accessUrl, accessKey: accessKey, secret: secret}
	err := client.fetchRegions()
	if err != nil {
		log.Debugf("NewAwsClient %s", err.Error())
		return nil, err
	}
	return &client, nil
}

func (self *SAwsClient) getDefaultSession() (*session.Session, error) {
	defaultRegion := AWS_INTERNATIONAL_DEFAULT_REGION
	switch self.accessUrl {
	case "InternationalCloud":
		defaultRegion = AWS_INTERNATIONAL_DEFAULT_REGION
	case "ChinaCloud":
		defaultRegion = AWS_CHINA_DEFAULT_REGION
	}
	return session.NewSession(&sdk.Config{
		Region:      sdk.String(defaultRegion),
		Credentials: credentials.NewStaticCredentials(self.accessKey, self.secret, ""),
	})
}

func (self *SAwsClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	// todo: implement me
	err := self.fetchRegions()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = self.providerName
	subAccount.Account = self.accessKey
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SAwsClient) UpdateAccount(accessKey, secret string) error {
	if self.accessKey != accessKey || self.secret != secret {
		self.accessKey = accessKey
		self.secret = secret
		return self.fetchRegions()
	} else {
		return nil
	}
}

// 用于初始化region信息
func (self *SAwsClient) fetchRegions() error {
	s, err := self.getDefaultSession()
	if err != nil {
		return err
	}
	svc := ec2.New(s)
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeRegions
	result, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})
	log.Debugf("remote regions: %s", result)
	if err != nil {
		return err
	}

	regions := make([]SRegion, 0)
	// empty iregions
	if self.iregions != nil {
		self.iregions = self.iregions[:0]
	}

	for _, region := range result.Regions {
		name := *region.RegionName
		endpoint := *region.Endpoint
		sregion := SRegion{client: self, RegionId: name, RegionEndpoint: endpoint}
		// 初始化region client
		sregion.getEc2Client()
		regions = append(regions, sregion)
		self.iregions = append(self.iregions, &sregion)
	}

	return nil
}

// 只是使用fetchRegions初始化好的self.iregions. 本身并不从云服务器厂商拉取region信息
func (self *SAwsClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SAwsClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SAwsClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = AWS_INTERNATIONAL_DEFAULT_REGION
		switch self.accessUrl {
		case "InternationalCloud":
			regionId = AWS_INTERNATIONAL_DEFAULT_REGION
		case "ChinaCloud":
			regionId = AWS_CHINA_DEFAULT_REGION
		}
	}
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SAwsClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAwsClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAwsClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIVpcById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAwsClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

type SAccountBalance struct {
	AvailableAmount     float64
	AvailableCashAmount float64
	CreditAmount        float64
	MybankCreditAmount  float64
	Currency            string
}

func (self *SAwsClient) QueryAccountBalance() (*SAccountBalance, error) {
	// todo: aws 貌似没有余额？
	panic("implement me")
}
