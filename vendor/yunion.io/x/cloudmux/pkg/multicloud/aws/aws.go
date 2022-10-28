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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	CLOUD_PROVIDER_AWS    = api.CLOUD_PROVIDER_AWS
	CLOUD_PROVIDER_AWS_CN = "AWS"
	CLOUD_PROVIDER_AWS_EN = "AWS"

	AWS_INTERNATIONAL_CLOUDENV = "InternationalCloud"
	AWS_CHINA_CLOUDENV         = "ChinaCloud"

	AWS_INTERNATIONAL_DEFAULT_REGION = "us-west-1"
	AWS_CHINA_DEFAULT_REGION         = "cn-north-1"
	AWS_API_VERSION                  = "2018-10-10"

	AWS_GLOBAL_ARN_PREFIX = "arn:aws:iam::aws:policy/"
	AWS_CHINA_ARN_PREFIX  = "arn:aws-cn:iam::aws:policy/"

	DEFAULT_S3_REGION_ID = "us-east-1"

	DefaultAssumeRoleName = "OrganizationAccountAccessRole"
)

var (
	DEBUG = false
)

type AwsClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	accessUrl    string // 服务区域 ChinaCloud | InternationalCloud
	accessKey    string
	accessSecret string
	accountId    string

	debug bool

	assumeRoleName string
}

func NewAwsClientConfig(accessUrl, accessKey, accessSecret, accountId string) *AwsClientConfig {
	cfg := &AwsClientConfig{
		accessUrl:    accessUrl,
		accessKey:    accessKey,
		accessSecret: accessSecret,
		accountId:    accountId,
	}
	return cfg
}

func (cfg *AwsClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *AwsClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *AwsClientConfig) Debug(debug bool) *AwsClientConfig {
	cfg.debug = debug
	DEBUG = debug
	return cfg
}

func (cfg *AwsClientConfig) SetAssumeRole(roleName string) *AwsClientConfig {
	cfg.assumeRoleName = roleName
	return cfg
}

func (cfg *AwsClientConfig) getAssumeRoleName() string {
	if len(cfg.assumeRoleName) > 0 {
		return cfg.assumeRoleName
	}
	return DefaultAssumeRoleName
}

type SAwsClient struct {
	*AwsClientConfig

	ownerId   string
	ownerName string

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket

	//sessions map[string]map[bool]*session.Session
	sessions sync.Map // map[string]map[bool]*session.Session
}

func NewAwsClient(cfg *AwsClientConfig) (*SAwsClient, error) {
	client := SAwsClient{
		AwsClientConfig: cfg,
	}
	_, err := client.fetchRegions()
	if err != nil {
		return nil, err
	}
	err = client.fetchOwnerId()
	if err != nil {
		return nil, errors.Wrap(err, "fetchOwnerId")
	}
	if client.debug {
		log.Debugf("ownerId: %s ownerName: %s", client.ownerId, client.ownerName)
	}
	return &client, nil
}

func (cli *SAwsClient) getIamArn(arn string) string {
	switch cli.GetAccessEnv() {
	case api.CLOUD_ACCESS_ENV_AWS_GLOBAL:
		return AWS_GLOBAL_ARN_PREFIX + arn
	default:
		return AWS_CHINA_ARN_PREFIX + arn
	}
}

func (cli *SAwsClient) getIamCommonArn(arn string) string {
	switch cli.GetAccessEnv() {
	case api.CLOUD_ACCESS_ENV_AWS_GLOBAL:
		return strings.TrimPrefix(arn, AWS_GLOBAL_ARN_PREFIX)
	default:
		return strings.TrimPrefix(arn, AWS_CHINA_ARN_PREFIX)
	}
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

func (client *SAwsClient) getDefaultSession(assumeRole bool) (*session.Session, error) {
	return client.getAwsSession(client.getDefaultRegionId(), assumeRole)
}

func (client *SAwsClient) GetAccountId() string {
	return client.ownerId
}

var (
	// cache for describeRegions
	describeRegionResult        sync.Map //map[string]*ec2.DescribeRegionsOutput = map[string]*ec2.DescribeRegionsOutput{}
	describeRegionResultCacheAt sync.Map // map[string]time.Time                  = map[string]time.Time{}
)

const (
	describeRegionExpireHours = 2
)

// 用于初始化region信息
func (self *SAwsClient) fetchRegions() ([]SRegion, error) {
	cacheTime, _ := describeRegionResultCacheAt.Load(self.accessUrl)
	if _, ok := describeRegionResult.Load(self.accessUrl); !ok || cacheTime.(time.Time).IsZero() || time.Now().After(cacheTime.(time.Time).Add(time.Hour*describeRegionExpireHours)) {
		s, err := self.getDefaultSession(false)
		if err != nil {
			return nil, errors.Wrap(err, "getDefaultSession")
		}
		svc := ec2.New(s)
		// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeRegions
		input := &ec2.DescribeRegionsInput{}
		input.SetAllRegions(true)
		result, err := svc.DescribeRegions(input)
		if err != nil {
			if e, ok := err.(awserr.Error); ok && e.Code() == "AuthFailure" {
				return nil, errors.Wrap(httperrors.ErrInvalidAccessKey, err.Error())
			}
			return nil, errors.Wrap(err, "DescribeRegions")
		}
		describeRegionResult.Store(self.accessUrl, result)
		describeRegionResultCacheAt.Store(self.accessUrl, time.Now())
	}

	self.iregions = []cloudprovider.ICloudRegion{}
	regions := make([]SRegion, 0)
	descRegions, _ := describeRegionResult.Load(self.accessUrl)
	for _, region := range descRegions.(*ec2.DescribeRegionsOutput).Regions {
		name := *region.RegionName
		endpoint := *region.Endpoint
		sregion := SRegion{client: self, RegionId: name, RegionEndpoint: endpoint}
		// 初始化region client
		// sregion.getEc2Client()
		regions = append(regions, sregion)
		self.iregions = append(self.iregions, &sregion)
	}

	return regions, nil
}

func (client *SAwsClient) getAwsSession(regionId string, assumeRole bool) (*session.Session, error) {
	client.sessions.LoadOrStore(regionId, map[bool]*session.Session{})
	if sess, ok := client.sessions.Load(regionId); ok {
		if ret, ok := sess.(map[bool]*session.Session)[assumeRole]; ok {
			return ret, nil
		}
	}
	httpClient := client.cpcfg.AdaptiveTimeoutHttpClient()
	transport, _ := httpClient.Transport.(*http.Transport)
	httpClient.Transport = cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response), error) {
		var action string
		if req.ContentLength > 0 {
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return nil, errors.Wrapf(err, "ioutil.ReadAll")
			}
			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			params, err := url.ParseQuery(string(body))
			if err != nil {
				return nil, errors.Wrapf(err, "ParseQuery(%s)", string(body))
			}
			action = params.Get("Action")
		}

		service := strings.Split(req.URL.Host, ".")[0]
		method, path := req.Method, req.URL.Path
		respCheck := func(resp *http.Response) {
			if resp.StatusCode == 403 {
				if client.cpcfg.UpdatePermission != nil {
					if len(action) > 0 {
						client.cpcfg.UpdatePermission(service, action)
					} else { // s3
						client.cpcfg.UpdatePermission(service, fmt.Sprintf("%s %s", method, path))
					}
				}
			}
		}

		if client.cpcfg.ReadOnly {
			if len(action) > 0 {
				for _, prefix := range []string{"Get", "List", "Describe"} {
					if strings.HasPrefix(action, prefix) {
						return respCheck, nil
					}
				}
				return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, action)
			}
			// s3
			if req.Method == "GET" || req.Method == "HEAD" {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return respCheck, nil
	})
	s, err := session.NewSession(&sdk.Config{
		Region: sdk.String(regionId),
		Credentials: credentials.NewStaticCredentials(
			client.accessKey, client.accessSecret, "",
		),
		HTTPClient:                    httpClient,
		DisableParamValidation:        sdk.Bool(true),
		CredentialsChainVerboseErrors: sdk.Bool(true),
	})
	if err != nil {
		return nil, errors.Wrap(err, "getAwsSession.NewSession")
	}
	if assumeRole && len(client.accountId) > 0 {
		// need to assumeRole
		var env string
		switch client.GetAccessEnv() {
		case api.CLOUD_ACCESS_ENV_AWS_GLOBAL:
			env = "aws"
		default:
			env = "aws-cn"
		}
		roleARN := fmt.Sprintf("arn:%s:iam::%s:role/%s", env, client.accountId, client.getAssumeRoleName())
		creds := stscreds.NewCredentials(s, roleARN)
		s = s.Copy(&aws.Config{Credentials: creds})
	}
	if client.debug {
		logLevel := aws.LogLevelType(uint(aws.LogDebugWithRequestErrors) + uint(aws.LogDebugWithHTTPBody) + uint(aws.LogDebugWithSigning))
		s.Config.LogLevel = &logLevel
	}

	client.sessions.Store(regionId, map[bool]*session.Session{assumeRole: s})
	return s, nil
}

func (region *SRegion) getAwsElasticacheClient() (*elasticache.ElastiCache, error) {
	session, err := region.getAwsSession()
	if err != nil {
		return nil, errors.Wrap(err, "client.getDefaultSession")
	}
	session.ClientConfig(ELASTICACHE_SERVICE_NAME)
	return elasticache.New(session), nil
}

func (client *SAwsClient) getAwsRoute53Session() (*session.Session, error) {
	session, err := client.getDefaultSession(true)
	if err != nil {
		return nil, errors.Wrap(err, "client.getDefaultSession()")
	}
	session.ClientConfig(ROUTE53_SERVICE_NAME)
	return session, nil
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

func (client *SAwsClient) fetchOwnerId() error {
	ident, err := client.GetCallerIdentity()
	if err != nil {
		return errors.Wrap(err, "GetCallerIdentity")
	}
	client.ownerId = ident.Account
	return nil
}

func (client *SAwsClient) fetchBuckets() error {
	s, err := client.getDefaultSession(true)
	if err != nil {
		return errors.Wrap(err, "getDefaultSession")
	}
	s3cli := s3.New(s)
	output, err := s3cli.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return errors.Wrap(err, "ListBuckets")
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
		if len(location) == 0 {
			// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLocation.html
			// Buckets in Region us-east-1 have a LocationConstraint of null.
			location = DEFAULT_S3_REGION_ID
		}
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
func (self *SAwsClient) GetRegions() ([]SRegion, error) {
	return self.fetchRegions()
}

func (self *SAwsClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SAwsClient) GetRegion(regionId string) (*SRegion, error) {
	regions, err := self.fetchRegions()
	if err != nil {
		return nil, errors.Wrapf(err, "fetchRegions")
	}

	if len(regionId) == 0 {
		regionId = AWS_INTERNATIONAL_DEFAULT_REGION
		switch self.accessUrl {
		case AWS_INTERNATIONAL_CLOUDENV:
			regionId = AWS_INTERNATIONAL_DEFAULT_REGION
		case AWS_CHINA_CLOUDENV:
			regionId = AWS_CHINA_DEFAULT_REGION
		}
	}
	for i := 0; i < len(regions); i += 1 {
		if regions[i].GetId() == regionId {
			return &regions[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, regionId)
}

func (self *SAwsClient) getDefaultRegion() (*SRegion, error) {
	return self.GetRegion("")
}

func (self *SAwsClient) getIRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "getIRegionByRegionId")
}

func (self *SAwsClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIRegionById")
}

func (self *SAwsClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			log.Errorf("GetIHostById %s: %s", id, err)
			return nil, errors.Wrap(err, "GetIHostById")
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIHostById")
}

func (self *SAwsClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIVpcById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			log.Errorf("GetIVpcById %s: %s", id, err)
			return nil, errors.Wrap(err, "GetIVpcById")
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIVpcById")
}

func (self *SAwsClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			log.Errorf("GetIStorageById %s: %s", id, err)
			return nil, errors.Wrap(err, "GetIStorageById")
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIStorageById")
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

func (self *SAwsClient) iamRequest(apiName string, params map[string]string, retval interface{}) error {
	return self.request("", IAM_SERVICE_NAME, IAM_SERVICE_ID, "2010-05-08", apiName, params, retval, true)
}

func (self *SAwsClient) stsRequest(apiName string, params map[string]string, retval interface{}) error {
	return self.request("", STS_SERVICE_NAME, STS_SERVICE_ID, "2011-06-15", apiName, params, retval, false)
}

func (self *SAwsClient) GetCapabilities() []string {
	caps := []string{
		// cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_NAT + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_DNSZONE,
		cloudprovider.CLOUD_CAPABILITY_SAML_AUTH,
		cloudprovider.CLOUD_CAPABILITY_WAF,
	}
	return caps
}

func (client *SAwsClient) GetIamLoginUrl() string {
	identity, err := client.GetCallerIdentity()
	if err != nil {
		log.Errorf("failed to get caller identity error: %v", err)
		return ""
	}

	switch client.GetAccessEnv() {
	case api.CLOUD_ACCESS_ENV_AWS_CHINA:
		return fmt.Sprintf("https://%s.signin.amazonaws.cn/console/", identity.Account)
	default:
		return fmt.Sprintf("https://%s.signin.aws.amazon.com/console/", identity.Account)
	}
}

func (client *SAwsClient) GetBucketCannedAcls() []string {
	switch client.GetAccessEnv() {
	case api.CLOUD_ACCESS_ENV_AWS_CHINA:
		return []string{
			string(cloudprovider.ACLPrivate),
		}
	default:
		return []string{
			string(cloudprovider.ACLPrivate),
			string(cloudprovider.ACLAuthRead),
			string(cloudprovider.ACLPublicRead),
			string(cloudprovider.ACLPublicReadWrite),
		}
	}
}

func (client *SAwsClient) GetObjectCannedAcls() []string {
	switch client.GetAccessEnv() {
	case api.CLOUD_ACCESS_ENV_AWS_CHINA:
		return []string{
			string(cloudprovider.ACLPrivate),
		}
	default:
		return []string{
			string(cloudprovider.ACLPrivate),
			string(cloudprovider.ACLAuthRead),
			string(cloudprovider.ACLPublicRead),
			string(cloudprovider.ACLPublicReadWrite),
		}
	}
}

func (client *SAwsClient) GetSamlEntityId() string {
	switch client.accessUrl {
	case AWS_CHINA_CLOUDENV:
		return cloudprovider.SAML_ENTITY_ID_AWS_CN
	default:
		return cloudprovider.SAML_ENTITY_ID_AWS
	}
}
