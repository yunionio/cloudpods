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

	"github.com/aws/aws-sdk-go/aws"
	sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
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

	regions  []SRegion
	iBuckets []cloudprovider.ICloudBucket
}

func NewAwsClient(cfg *AwsClientConfig) (*SAwsClient, error) {
	client := SAwsClient{
		AwsClientConfig: cfg,
	}
	var err error
	client.regions, err = client.GetRegions()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegions")
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
	err := client.fetchOwnerId()
	if err != nil {
		return ""
	}
	return client.ownerId
}

func (self *SAwsClient) ec2Request(regionId string, apiName string, params map[string]string, retval interface{}, assumeRole bool) error {
	return self.request(regionId, EC2_SERVICE_NAME, EC2_SERVICE_ID, "2016-11-15", apiName, params, retval, assumeRole)
}

// Amazon Elastic Container Service
func (self *SAwsClient) ecsRequest(regionId string, apiName string, params map[string]interface{}, retval interface{}, assumeRole bool) error {
	return self.invoke(regionId, ECS_SERVICE_NAME, ECS_SERVICE_ID, "2014-11-13", apiName, "", params, retval, assumeRole)
}

func (self *SAwsClient) lambdaRequest(regionId string, apiName, path string, params map[string]interface{}, retval interface{}, assumeRole bool) error {
	return self.invoke(regionId, LAMBDA_SERVICE_NAME, LAMBDA_SERVICE_ID, "2015-03-31", apiName, path, params, retval, assumeRole)
}

func (self *SAwsClient) kinesisRequest(regionId string, apiName, path string, params map[string]interface{}, retval interface{}, assumeRole bool) error {
	return self.invoke(regionId, KINESIS_SERVICE_NAME, KINESIS_SERVICE_ID, "2013-12-02", apiName, path, params, retval, assumeRole)
}

func (self *SAwsClient) cfRequest(apiName string, params map[string]string, retval interface{}, assumeRole bool) error {
	return self.request("", CDN_SERVICE_NAME, CDN_SERVICE_ID, "2020-05-31", apiName, params, retval, assumeRole)
}

func (client *SAwsClient) getAwsSession(regionId string, assumeRole bool) (*session.Session, error) {
	httpClient := client.cpcfg.AdaptiveTimeoutHttpClient()
	transport, _ := httpClient.Transport.(*http.Transport)
	httpClient.Transport = cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response) error, error) {
		var action string
		if req.ContentLength > 0 && !strings.Contains(req.URL.Host, ".s3.") {
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
		respCheck := func(resp *http.Response) error {
			if resp.StatusCode == 403 {
				if client.cpcfg.UpdatePermission != nil {
					if len(action) > 0 {
						client.cpcfg.UpdatePermission(service, action)
					} else { // s3
						client.cpcfg.UpdatePermission(service, fmt.Sprintf("%s %s", method, path))
					}
				}
			}
			return nil
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
			// organization
			if service == "organizations" {
				return respCheck, nil
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

	return s, nil
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
		if e, ok := err.(awserr.Error); ok && e.Code() == "AccessDenied" {
			return errors.Wrapf(cloudprovider.ErrForbidden, e.Message())
		}
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
		region, err := client.GetIRegionById(location)
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
	params := map[string]string{
		"AllRegions": "true",
	}
	ret := struct {
		RegionInfo []SRegion `xml:"regionInfo>item"`
	}{}
	err := self.ec2Request("", "DescribeRegions", params, &ret, false)
	if err != nil {
		if e, ok := err.(*sAwsError); ok && e.Errors.Code == "AuthFailure" {
			return nil, errors.Wrap(cloudprovider.ErrInvalidAccessKey, err.Error())
		}
		return nil, errors.Wrapf(err, "DescribeRegions")
	}
	return ret.RegionInfo, nil
}

func (self *SAwsClient) GetIRegions() []cloudprovider.ICloudRegion {
	ret := []cloudprovider.ICloudRegion{}
	for i := range self.regions {
		self.regions[i].client = self
		ret = append(ret, &self.regions[i])
	}
	return ret
}

func (self *SAwsClient) GetRegion(regionId string) (*SRegion, error) {
	regions, err := self.GetRegions()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegions")
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
			regions[i].client = self
			return &regions[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, regionId)
}

func (self *SAwsClient) getDefaultRegion() (*SRegion, error) {
	return self.GetRegion("")
}

func (self *SAwsClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := range self.regions {
		self.regions[i].client = self
		if self.regions[i].GetId() == id || self.regions[i].GetGlobalId() == id {
			return &self.regions[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetIRegionById(%s)", id)
}

func (self *SAwsClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(self.regions); i += 1 {
		ihost, err := self.regions[i].GetIHostById(id)
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
	for i := 0; i < len(self.regions); i += 1 {
		ihost, err := self.regions[i].GetIVpcById(id)
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
	for i := 0; i < len(self.regions); i += 1 {
		ihost, err := self.regions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			log.Errorf("GetIStorageById %s: %s", id, err)
			return nil, errors.Wrap(err, "GetIStorageById")
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIStorageById")
}

func (self *SAwsClient) cdnList(marker string, pageSize int64) ([]SCdnDomain, string, error) {
	input := map[string]string{
		"Marker":   marker,
		"MaxItems": fmt.Sprintf("%d", pageSize),
	}
	ret := &struct {
		Items      []SCdnDomain `xml:"Items>DistributionSummary"`
		NextMarker string       `xml:"NextMarker,omitempty"`
	}{}
	err := self.cfRequest("ListDistributions2020_05_31", input, ret, true)
	if err != nil {
		return nil, "", errors.Wrap(err, "cdnList")
	}
	return ret.Items, ret.NextMarker, err
}

func (self *SAwsClient) cdnGet(id string) (*SCdnDomain, error) {
	input := map[string]string{
		"Id": id,
	}
	var ret SCdnDomain
	err := self.cfRequest("GetDistribution2020_05_31", input, &ret, true)
	if err != nil {
		return nil, errors.Wrap(err, "cdnGet")
	}

	return &ret, err
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

func (self *SAwsClient) dnsRequest(apiName string, params map[string]string, retval interface{}) error {
	return self.request("", ROUTE53_SERVICE_NAME, ROUTE53_SERVICE_ID, "2013-04-01", apiName, params, retval, true)
}

func (self *SAwsClient) stsRequest(apiName string, params map[string]string, retval interface{}) error {
	return self.request("", STS_SERVICE_NAME, STS_SERVICE_ID, "2011-06-15", apiName, params, retval, false)
}

func (self *SAwsClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_NAT + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_DNSZONE,
		cloudprovider.CLOUD_CAPABILITY_SAML_AUTH,
		cloudprovider.CLOUD_CAPABILITY_WAF,
		cloudprovider.CLOUD_CAPABILITY_VPC_PEER,
		cloudprovider.CLOUD_CAPABILITY_CONTAINER,
		cloudprovider.CLOUD_CAPABILITY_CDN + cloudprovider.READ_ONLY_SUFFIX,
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
