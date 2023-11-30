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
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/wafv2"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var (
	lock sync.RWMutex
)

var RegionLocations = map[string]string{
	"us-east-2":      "美国东部(俄亥俄州)",
	"us-east-1":      "美国东部(弗吉尼亚北部)",
	"us-west-1":      "美国西部(加利福尼亚北部)",
	"us-west-2":      "美国西部(俄勒冈)",
	"ap-east-1":      "亚太区域(香港)",
	"ap-south-1":     "亚太区域(孟买)",
	"ap-northeast-3": "亚太区域(大阪-本地)",
	"ap-northeast-2": "亚太区域(首尔)",
	"ap-southeast-1": "亚太区域(新加坡)",
	"ap-southeast-2": "亚太区域(悉尼)",
	"ap-northeast-1": "亚太区域(东京)",
	"ca-central-1":   "加拿大(中部)",
	"cn-north-1":     "中国(北京)",
	"cn-northwest-1": "中国(宁夏)",
	"eu-central-1":   "欧洲(法兰克福)",
	"eu-west-1":      "欧洲(爱尔兰)",
	"eu-west-2":      "欧洲(伦敦)",
	"eu-south-1":     "欧洲(米兰)",
	"eu-west-3":      "欧洲(巴黎)",
	"eu-north-1":     "欧洲(斯德哥尔摩)",
	"me-south-1":     "中东(巴林)",
	"sa-east-1":      "南美洲(圣保罗)",
	"us-gov-west-1":  "AWS GovCloud(美国西部)",
	"us-gov-east-1":  "AWS GovCloud(美国东部)",

	"af-south-1": "非洲(开普敦)",

	"ap-southeast-4": "亚太地区(墨尔本)",
	"ap-south-2":     "亚太地区(德拉巴)",
	"eu-south-2":     "欧洲(西班牙)",
	"eu-central-2":   "欧洲(苏黎世)",
	"me-central-1":   "中东(阿联酋)",
	"ap-southeast-3": "亚太地区(雅加达)",
	"il-central-1":   "以色列(特拉维夫)",
}

// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Concepts.RegionsAndAvailabilityZones.html
var RegionLocationsEN = map[string]string{
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"af-south-1":     "Africa (Cape Town)",
	"ap-east-1":      "Asia Pacific (Hong Kong)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-northeast-3": "Asia Pacific (Osaka)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ca-central-1":   "Canada (Central)",
	"eu-central-1":   "Europe (Frankfurt)",
	"eu-west-1":      "Europe (Ireland)",
	"eu-west-2":      "Europe (London)",
	"eu-south-1":     "Europe (Milan)",
	"eu-west-3":      "Europe (Paris)",
	"eu-north-1":     "Europe (Stockholm)",
	"me-south-1":     "Middle East (Bahrain)",
	"sa-east-1":      "South America (São Paulo)",
	"cn-north-1":     "China (Beijing)",
	"cn-northwest-1": "China (Ninxia)",
	"us-gov-west-1":  "AWS GovCloud(US West)",
	"us-gov-east-1":  "AWS GovCloud(US East)",

	"ap-southeast-4": "Asia Pacific (Melbourne)",
	"ap-south-2":     "Asia Pacific (Hyderabad)",
	"eu-south-2":     "Europe (Spain)",
	"eu-central-2":   "Europe (Zurich)",
	"me-central-1":   "Middle East (UAE)",
	"ap-southeast-3": "Asia Pacific (Jakarta)",
	"il-central-1":   "Israel (Tel Aviv)",
}

const (
	RDS_SERVICE_NAME = "rds"
	RDS_SERVICE_ID   = "RDS"

	EC2_SERVICE_NAME = "ec2"
	EC2_SERVICE_ID   = "EC2"

	IAM_SERVICE_NAME = "iam"
	IAM_SERVICE_ID   = "IAM"

	STS_SERVICE_NAME = "sts"
	STS_SERVICE_ID   = "STS"

	CLOUDWATCH_SERVICE_NAME = "monitoring"
	CLOUDWATCH_SERVICE_ID   = "CloudWatch"

	CLOUD_TRAIL_SERVICE_NAME = "CloudTrail"
	CLOUD_TRAIL_SERVICE_ID   = "cloudtrail"

	ROUTE53_SERVICE_NAME = "route53"
	ROUTE53_SERVICE_ID   = "Route 53"

	ELASTICACHE_SERVICE_NAME = "elasticache"
	ELASTICACHE_SERVICE_ID   = "ElastiCache"

	ELB_SERVICE_NAME = "elasticloadbalancing"
	ELB_SERVICE_ID   = "Elastic Load Balancing v2"

	EKS_SERVICE_NAME = "eks"
	EKS_SERVICE_ID   = "EKS"

	PRICING_SERVICE_NAME = "api.pricing"
	PRICING_SERVICE_ID   = "Pricing"

	CDN_SERVICE_NAME = "cloudfront"
	CDN_SERVICE_ID   = "CloudFront"

	ECS_SERVICE_NAME = "ecs"
	ECS_SERVICE_ID   = "ECS"

	LAMBDA_SERVICE_NAME = "lambda"
	LAMBDA_SERVICE_ID   = "Lambda"

	KINESIS_SERVICE_NAME = "kinesis"
	KINESIS_SERVICE_ID   = "Kinesis"

	DYNAMODB_SERVICE_NAME = "dynamodb"
	DYNAMODB_SERVICE_ID   = "DynamoDB"
)

type SRegion struct {
	multicloud.SRegion

	client                 *SAwsClient
	s3Client               *s3.S3
	wafClient              *wafv2.WAFV2
	organizationClient     *organizations.Organizations
	resourceGroupTagClient *resourcegroupstaggingapi.ResourceGroupsTaggingAPI

	RegionEndpoint string `xml:"regionEndpoint"`
	RegionId       string `xml:"regionName"`
	OptInStatus    string `xml:"optInStatus"`
}

/////////////////////////////////////////////////////////////////////////////
/* 请不要使用这个client(AWS_DEFAULT_REGION)跨region查信息.有可能导致查询返回的信息为空。比如DescribeAvailabilityZones*/
func (self *SRegion) GetClient() *SAwsClient {
	return self.client
}

func (self *SRegion) getAwsSession() (*session.Session, error) {
	return self.client.getAwsSession(self.RegionId, true)
}

func (self *SRegion) getWafClient() (*wafv2.WAFV2, error) {
	if self.wafClient == nil {
		s, err := self.getAwsSession()
		if err != nil {
			return nil, errors.Wrapf(err, "getAwsSession")
		}
		self.wafClient = wafv2.New(s)
	}
	return self.wafClient, nil
}

func (self *SRegion) GetS3Client() (*s3.S3, error) {
	if self.s3Client == nil {
		s, err := self.getAwsSession()
		if err != nil {
			return nil, errors.Wrap(err, "getAwsSession")
		}
		self.s3Client = s3.New(s,
			&aws.Config{
				DisableRestProtocolURICleaning: aws.Bool(true),
			})
	}
	return self.s3Client, nil
}

func (r *SRegion) getOrganizationClient() (*organizations.Organizations, error) {
	if r.organizationClient == nil {
		s, err := r.getAwsSession()
		if err != nil {
			return nil, errors.Wrap(err, "getAwsSession")
		}
		r.organizationClient = organizations.New(s)
	}
	return r.organizationClient, nil
}

func (self *SRegion) getResourceGroupTagClient() (*resourcegroupstaggingapi.ResourceGroupsTaggingAPI, error) {
	if self.resourceGroupTagClient == nil {
		s, err := self.getAwsSession()
		if err != nil {
			return nil, errors.Wrap(err, "getAwsSession")
		}
		self.resourceGroupTagClient = resourcegroupstaggingapi.New(s)
	}
	return self.resourceGroupTagClient, nil
}

func (self *SRegion) elbRequest(apiName string, params map[string]string, retval interface{}) error {
	return self.client.request(self.RegionId, ELB_SERVICE_NAME, ELB_SERVICE_ID, "2015-12-01", apiName, params, retval, true)
}

func (self *SRegion) rdsRequest(apiName string, params map[string]string, retval interface{}) error {
	return self.client.request(self.RegionId, RDS_SERVICE_NAME, RDS_SERVICE_ID, "2014-10-31", apiName, params, retval, true)
}

func (self *SRegion) ecRequest(apiName string, params map[string]string, retval interface{}) error {
	return self.client.request(self.RegionId, ELASTICACHE_SERVICE_NAME, ELASTICACHE_SERVICE_ID, "2015-02-02", apiName, params, retval, true)
}

func (self *SRegion) ec2Request(apiName string, params map[string]string, retval interface{}) error {
	return self.client.ec2Request(self.RegionId, apiName, params, retval, true)
}

func (self *SAwsClient) monitorRequest(regionId, apiName string, params map[string]string, retval interface{}) error {
	return self.request(regionId, CLOUDWATCH_SERVICE_NAME, CLOUDWATCH_SERVICE_ID, "2010-08-01", apiName, params, retval, true)
}

// Amazon Elastic Container Service
func (self *SRegion) ecsRequest(apiName string, params map[string]interface{}, retval interface{}) error {
	return self.client.ecsRequest(self.RegionId, apiName, params, retval, true)
}

func (self *SRegion) lambdaRequest(apiName, path string, params map[string]interface{}, retval interface{}) error {
	return self.client.lambdaRequest(self.RegionId, apiName, path, params, retval, true)
}

func (self *SRegion) kinesisRequest(apiName, path string, params map[string]interface{}, retval interface{}) error {
	return self.client.kinesisRequest(self.RegionId, apiName, path, params, retval, true)
}

func (self *SRegion) dynamodbRequest(apiName string, params map[string]interface{}, retval interface{}) error {
	return self.client.invoke(self.RegionId, DYNAMODB_SERVICE_NAME, DYNAMODB_SERVICE_ID, "2012-08-10", apiName, "", params, retval, true)
}

func (self *SRegion) eksRequest(apiName, path string, params map[string]interface{}, retval interface{}) error {
	return self.client.invoke(self.RegionId, EKS_SERVICE_NAME, EKS_SERVICE_ID, "2017-11-01", apiName, path, params, retval, true)
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) GetZones(id string) ([]SZone, error) {
	params := map[string]string{
		"Filter.1.Name":    "region-name",
		"Filter.1.Value.1": self.RegionId,
	}
	if len(id) > 0 {
		params["Filter.2.Name"] = "zone-id"
		params["Filter.2.Value.1"] = id
	}
	ret := struct {
		AvailabilityZoneInfo []SZone `xml:"availabilityZoneInfo>item"`
	}{}
	err := self.ec2Request("DescribeAvailabilityZones", params, &ret)
	if err != nil {
		return nil, err
	}
	return ret.AvailabilityZoneInfo, nil
}

func (self *SRegion) GetId() string {
	return self.RegionId
}

func (self *SRegion) GetName() string {
	if localName, ok := RegionLocations[self.RegionId]; ok {
		return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_CN, localName)
	}

	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_CN, self.RegionId)
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	var en string
	if localName, ok := RegionLocationsEN[self.RegionId]; ok {
		en = fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_EN, localName)
	} else {
		en = fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_EN, self.RegionId)
	}

	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.client.GetAccessEnv(), self.RegionId)
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) Refresh() error {
	return nil
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.RegionId]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := self.GetZones("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetZones")
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		zones[i].region = self
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetVpcs(nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		vpcs[i].region = self
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return self.GetInstance(id)
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := self.GetEips("", "", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetEips")
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eips[i].region = self
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("", "", []string{})
	if err != nil {
		return nil, errors.Wrap(err, "GetSnapshots")
	}

	ret := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i += 1 {
		snapshots[i].region = self
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := self.GetZones("")
	if err != nil {
		return nil, errors.Wrap(err, "GetZones")
	}

	for i := range zones {
		zones[i].region = self
		if zones[i].GetId() == id || zones[i].GetGlobalId() == id {
			return &zones[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetVpcs([]string{id})
	if err != nil {
		return nil, errors.Wrap(err, "GetIVpcs")
	}

	for i := range vpcs {
		vpcs[i].region = self
		if vpcs[i].GetGlobalId() == id {
			return &vpcs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetIVpcById %s", id)
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
		if !gotypes.IsNil(ihost) {
			return ihost, nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIHostById")
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}
	for i := 0; i < len(izones); i += 1 {
		istore, err := izones[i].GetIStorageById(id)
		if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
		if !gotypes.IsNil(istore) {
			return istore, nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIStorageById")
}

func (self *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := make([]cloudprovider.ICloudHost, 0)
	izones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneHost, err := izones[i].GetIHosts()
		if err != nil {
			return nil, errors.Wrap(err, "GetIHosts")
		}
		iHosts = append(iHosts, iZoneHost...)
	}
	return iHosts, nil
}

func (self *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneStores, err := izones[i].GetIStorages()
		if err != nil {
			return nil, errors.Wrap(err, "GetIStorages")
		}
		iStores = append(iStores, iZoneStores...)
	}
	return iStores, nil
}

func (self *SRegion) getStorageCache() *SStoragecache {
	return &SStoragecache{region: self}
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := self.GetIStoragecaches()
	if err != nil {
		return nil, err
	}
	for i := range caches {
		if caches[i].GetGlobalId() == id {
			return caches[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIStoragecacheById")

}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	vpc, err := self.CreateVpc(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateVpc")
	}
	return vpc, nil
}

func (self *SRegion) CreateVpc(opts *cloudprovider.VpcCreateOptions) (*SVpc, error) {
	params := map[string]string{
		"CidrBlock": opts.CIDR,
	}
	tagIdx := 1
	if len(opts.NAME) > 0 {
		params["TagSpecification.1.ResourceType"] = "vpc"
		params[fmt.Sprintf("TagSpecification.%d.Tag.1.Key", tagIdx)] = "Name"
		params[fmt.Sprintf("TagSpecification.%d.Tag.1.Value", tagIdx)] = opts.NAME
		if len(opts.Desc) > 0 {
			params[fmt.Sprintf("TagSpecification.%d.Tag.2.Key", tagIdx)] = "Description"
			params[fmt.Sprintf("TagSpecification.%d.Tag.2.Value", tagIdx)] = opts.Desc
		}
		tagIdx++
	}

	ret := struct {
		Vpc SVpc `xml:"vpc"`
	}{}
	err := self.ec2Request("CreateVpc", params, &ret)
	if err != nil {
		return nil, err
	}
	ret.Vpc.region = self
	return &ret.Vpc, nil
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEip(id)
	if err != nil {
		return nil, errors.Wrap(err, "GetEip")
	}
	return eip, nil
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_AWS
}

func (self *SRegion) GetCloudEnv() string {
	return self.client.accessUrl
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	ret := []cloudprovider.ICloudLoadbalancer{}
	marker := ""
	for {
		part, marker, err := self.GetLoadbalancers("", marker)
		if err != nil {
			return nil, errors.Wrapf(err, "GetLoadbalancers")
		}
		for i := range part {
			part[i].region = self
			ret = append(ret, &part[i])
		}
		if len(marker) == 0 || len(part) == 0 {
			break
		}
	}
	return ret, nil
}

func (self *SRegion) GetLoadBalancer(id string) (*SElb, error) {
	part, _, err := self.GetLoadbalancers(id, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancers")
	}
	for i := range part {
		if part[i].GetGlobalId() == id {
			part[i].region = self
			return &part[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetILoadBalancerById(id string) (cloudprovider.ICloudLoadbalancer, error) {
	lb, err := self.GetLoadBalancer(id)
	if err != nil {
		return nil, err
	}
	return lb, nil
}

func (self *SRegion) GetElbAttributes(id string) (map[string]string, error) {
	ret := struct {
		Attributes []struct {
			Key   string
			Value string
		} `xml:"Attributes>member"`
	}{}
	params := map[string]string{"LoadBalancerArn": id}
	err := self.elbRequest("DescribeLoadBalancerAttributes", params, &ret)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, attr := range ret.Attributes {
		result[attr.Key] = attr.Value
	}
	return result, nil
}

func (self *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	certs, err := self.GetILoadBalancerCertificates()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadBalancerCertificates")
	}

	for i := range certs {
		if certs[i].GetId() == certId {
			return certs[i], nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetILoadBalancerCertificateById")
}

func (self *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	certs, err := self.ListServerCertificates()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudLoadbalancerCertificate{}
	for i := range certs {
		certs[i].region = self
		ret = append(ret, &certs[i])
	}

	return ret, nil
}

func (self *SRegion) CreateILoadBalancer(opts *cloudprovider.SLoadbalancerCreateOptions) (cloudprovider.ICloudLoadbalancer, error) {
	lb, err := self.CreateLoadbalancer(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateLoadbalancer")
	}
	return lb, nil
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "getIBuckets")
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range iBuckets {
		if iBuckets[i].GetLocation() != region.GetId() {
			continue
		}
		ret = append(ret, iBuckets[i])
	}
	return ret, nil
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, acl string) error {
	s3cli, err := region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.CreateBucketInput{}
	input.SetBucket(name)
	if region.GetId() != DEFAULT_S3_REGION_ID {
		location := region.GetId()
		input.CreateBucketConfiguration = &s3.CreateBucketConfiguration{}
		input.CreateBucketConfiguration.SetLocationConstraint(location)
	}
	_, err = s3cli.CreateBucket(input)
	if err != nil {
		return errors.Wrap(err, "CreateBucket")
	}
	region.client.invalidateIBuckets()
	// if *output.Location != region.GetId() {
	// 	log.Warningf("Request location %s != got locaiton %s", region.GetId(), *output.Location)
	// }
	return nil
}

func (region *SRegion) DeleteIBucket(name string) error {
	s3cli, err := region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.DeleteBucketInput{}
	input.Bucket = &name
	_, err = s3cli.DeleteBucket(input)
	if err != nil {
		if region.client.debug {
			log.Debugf("%#v %s", err, err)
		}
		if strings.Index(err.Error(), "NoSuchBucket:") >= 0 {
			return nil
		}
		return errors.Wrap(err, "DeleteBucket")
	}
	region.client.invalidateIBuckets()
	return nil
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	s3cli, err := region.GetS3Client()
	if err != nil {
		return false, errors.Wrap(err, "GetS3Client")
	}
	input := &s3.HeadBucketInput{}
	input.Bucket = &name
	_, err = s3cli.HeadBucket(input)
	if err != nil {
		if region.client.debug {
			log.Debugf("%#v %s", err, err)
		}
		if strings.Index(err.Error(), "NotFound:") >= 0 {
			return false, nil
		}
		return false, errors.Wrap(err, "IsBucketExist")
	}
	return true, nil
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(region, name)
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}

func (region *SRegion) getBaseEndpoint() string {
	if len(region.RegionEndpoint) > 4 {
		return region.RegionEndpoint[4:]
	}
	return ""
}

func (region *SRegion) getS3Endpoint() string {
	base := region.getBaseEndpoint()
	if len(base) > 0 {
		return "s3." + base
	}
	return ""
}

func (region *SRegion) getS3WebsiteEndpoint() string {
	base := region.getBaseEndpoint()
	if len(base) > 0 {
		return "s3-website." + base
	}
	return ""
}

func (region *SRegion) getEc2Endpoint() string {
	return region.RegionEndpoint
}

func (self *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	ret, err := self.GetSecurityGroup(id)
	if err != nil {
		return nil, errors.Wrap(err, "GetSecurityGroups")
	}
	return ret, nil
}

func (self *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	groupId, err := self.CreateSecurityGroup(opts)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSecurityGroup")
	}
	return self.GetISecurityGroupById(groupId)
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (self *SRegion) CreateIgw() (*SInternetGateway, error) {
	params := map[string]string{}
	ret := struct {
		InternetGateway SInternetGateway `xml:"internetGateway"`
	}{}
	err := self.ec2Request("CreateInternetGateway", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateInternetGateway")
	}
	ret.InternetGateway.region = self
	return &ret.InternetGateway, nil
}

func (region *SRegion) CreateInternetGateway() (cloudprovider.ICloudInternetGateway, error) {
	igw, err := region.CreateIgw()
	if err != nil {
		return nil, err
	}
	return igw, nil
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := region.GetInstances("", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}

	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		ivms[i] = &vms[i]
	}
	return ivms, nil
}
