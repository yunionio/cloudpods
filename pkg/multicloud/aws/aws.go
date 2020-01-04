// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_AWS    = api.CLOUD_PROVIDER_AWS
	CLOUD_PROVIDER_AWS_CN = "AWS"

	AWS_INTERNATIONAL_CLOUDENV = "InternationalCloud"
	AWS_CHINA_CLOUDENV         = "ChinaCloud"

	AWS_INTERNATIONAL_DEFAULT_REGION = "us-west-1"
	AWS_CHINA_DEFAULT_REGION         = "cn-north-1"
	AWS_API_VERSION                  = "2018-10-10"
)

var (
	DEBUG = false
)

type SAwsClient struct {
	providerId   string
	providerName string
	accessUrl    string // 服务区域 ChinaCloud | InternationalCloud
	accessKey    string
	secret       string

	ownerId   string
	ownerName string

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket

	debug bool
}

func NewAwsClient(providerId string, providerName string, accessUrl string, accessKey string, secret string, debug bool) (*SAwsClient, error) {
	client := SAwsClient{
		providerId:   providerId,
		providerName: providerName,
		accessUrl:    accessUrl,
		accessKey:    accessKey,
		secret:       secret,
		debug:        debug,
	}
	DEBUG = debug
	err := client.fetchRegions()
	if err != nil {
		return nil, errors.Wrap(err, "fetchRegions")
	}
	err = client.fetchBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "fetchBuckets")
	}
	if debug {
		log.Debugf("ownerId: %s ownerName: %s", client.ownerId, client.ownerName)
	}
	return &client, nil
}

func GetDefaultRegionId(accessUrl string) string {
	defaultRegion := AWS_INTERNATIONAL_DEFAULT_REGION
	switch accessUrl {
	case AWS_INTERNATIONAL_CLOUDENV:
		defaultRegion = AWS_INTERNATIONAL_DEFAULT_REGION
	case AWS_CHINA_CLOUDENV:
		defaultRegion = AWS_CHINA_DEFAULT_REGION
	}

	return defaultRegion
}

func (self *SAwsClient) getDefaultRegionId() string {
	return GetDefaultRegionId(self.accessUrl)
}

func (client *SAwsClient) getDefaultSession() (*session.Session, error) {
	return client.getAwsSession(client.getDefaultRegionId())
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
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (client *SAwsClient) GetAccountId() string {
	return client.ownerId
}

func (self *SAwsClient) UpdateAccount(accessKey, secret string) error {
	if self.accessKey != accessKey || self.secret != secret {
		self.accessKey = accessKey
		self.secret = secret
		self.iregions = nil
		return self.fetchRegions()
	} else {
		return nil
	}
}

// 用于初始化region信息
func (self *SAwsClient) fetchRegions() error {
	if self.iregions != nil {
		return nil
	}
	s, err := self.getDefaultSession()
	if err != nil {
		return err
	}
	svc := ec2.New(s)
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeRegions
	result, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return err
	}

	regions := make([]SRegion, 0)
	// empty iregions
	// if self.iregions != nil {
	// 	self.iregions = self.iregions[:0]
	// }

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

func (client *SAwsClient) getAwsSession(regionId string) (*session.Session, error) {
	disableParamValidation := true
	chainVerboseErrors := true
	return session.NewSession(&sdk.Config{
		Region:                 sdk.String(regionId),
		Credentials:            credentials.NewStaticCredentials(client.accessKey, client.secret, ""),
		DisableParamValidation: &disableParamValidation,

		CredentialsChainVerboseErrors: &chainVerboseErrors,
	})
}

func (self *SAwsClient) invalidateIBuckets() {
	self.iBuckets = nil
}

func (self *SAwsClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if self.iBuckets == nil {
		err := self.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.iBuckets, nil
}

func (client *SAwsClient) fetchBuckets() error {
	s, err := client.getDefaultSession()
	if err != nil {
		return errors.Wrap(err, "getDefaultSession")
	}
	s3cli := s3.New(s)
	output, err := s3cli.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return errors.Wrap(err, "ListBuckets")
	}

	if output.Owner != nil {
		if output.Owner.ID != nil {
			client.ownerId = *output.Owner.ID
		}
		if output.Owner.DisplayName != nil {
			client.ownerName = *output.Owner.DisplayName
		}
	}

	ret := make([]cloudprovider.ICloudBucket, 0)
	for _, bInfo := range output.Buckets {
		if err := FillZero(bInfo); err != nil {
			log.Errorf("s3cli.Binfo.FillZero error %s", err)
			continue
		}

		input := &s3.GetBucketLocationInput{}
		input.Bucket = bInfo.Name
		output, err := s3cli.GetBucketLocation(input)
		if err != nil {
			log.Errorf("s3cli.GetBucketLocation error %s", err)
			continue
		}

		if err := FillZero(output); err != nil {
			log.Errorf("s3cli.GetBucketLocation.FillZero error %s", err)
			continue
		}

		location := *output.LocationConstraint
		region, err := client.getIRegionByRegionId(location)
		if err != nil {
			log.Errorf("client.getIRegionByRegionId %s fail %s", location, err)
			continue
		}
		b := SBucket{
			region:       region.(*SRegion),
			Name:         *bInfo.Name,
			Location:     location,
			CreationDate: *bInfo.CreationDate,
		}
		ret = append(ret, &b)
	}

	client.iBuckets = ret

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
		case AWS_INTERNATIONAL_CLOUDENV:
			regionId = AWS_INTERNATIONAL_DEFAULT_REGION
		case AWS_CHINA_CLOUDENV:
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

func (self *SAwsClient) getIRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, ErrorNotFound()
}

func (self *SAwsClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, ErrorNotFound()
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
	return nil, ErrorNotFound()
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
	return nil, ErrorNotFound()
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
	return nil, ErrorNotFound()
}

type SAccountBalance struct {
	AvailableAmount     float64
	AvailableCashAmount float64
	CreditAmount        float64
	MybankCreditAmount  float64
	Currency            string
}

func (self *SAwsClient) QueryAccountBalance() (*SAccountBalance, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SAwsClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SAwsClient) GetAccessEnv() string {
	switch self.accessUrl {
	case AWS_INTERNATIONAL_CLOUDENV:
		return api.CLOUD_ACCESS_ENV_AWS_GLOBAL
	case AWS_CHINA_CLOUDENV:
		return api.CLOUD_ACCESS_ENV_AWS_CHINA
	default:
		return api.CLOUD_ACCESS_ENV_AWS_GLOBAL
	}
}

func jsonRequest(cli *client.Client, apiName string, params map[string]string, retval interface{}, debug bool) error {
	op := &request.Operation{
		Name:       apiName,
		HTTPMethod: "POST",
		HTTPPath:   "/",
		Paginator: &request.Paginator{
			InputTokens:     []string{"NextToken"},
			OutputTokens:    []string{"NextToken"},
			LimitToken:      "MaxResults",
			TruncationToken: "",
		},
	}

	req := cli.NewRequest(op, params, retval)
	err := req.Send()
	if err != nil {
		return err
	}
	return nil
}

func (self *SAwsClient) GetCapabilities() []string {
	caps := []string{
		// cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
	}
	return caps
}
