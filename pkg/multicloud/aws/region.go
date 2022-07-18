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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/private/protocol/query"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/wafv2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
}

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

	ELASTICACHE_SERVICE_NAME = "elasticache"
)

type SRegion struct {
	multicloud.SRegion

	client                 *SAwsClient
	ec2Client              *ec2.EC2
	iamClient              *iam.IAM
	s3Client               *s3.S3
	elbv2Client            *elbv2.ELBV2
	acmClient              *acm.ACM
	wafClient              *wafv2.WAFV2
	organizationClient     *organizations.Organizations
	resourceGroupTagClient *resourcegroupstaggingapi.ResourceGroupsTaggingAPI

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	RegionEndpoint string
	RegionId       string // 这里为保持一致沿用阿里云RegionId的叫法, 与AWS RegionName字段对应
}

/////////////////////////////////////////////////////////////////////////////
/* 请不要使用这个client(AWS_DEFAULT_REGION)跨region查信息.有可能导致查询返回的信息为空。比如DescribeAvailabilityZones*/
func (self *SRegion) GetClient() *SAwsClient {
	return self.client
}

func (self *SRegion) getAwsSession() (*session.Session, error) {
	return self.client.getAwsSession(self.RegionId, true)
}

func (self *SRegion) getEc2Client() (*ec2.EC2, error) {
	if self.ec2Client == nil {
		s, err := self.getAwsSession()

		if err != nil {
			return nil, errors.Wrap(err, "getAwsSession")
		}

		self.ec2Client = ec2.New(s)
	}

	return self.ec2Client, nil
}

func (self *SRegion) getIamClient() (*iam.IAM, error) {
	if self.iamClient == nil {
		s, err := self.getAwsSession()

		if err != nil {
			return nil, errors.Wrap(err, "getAwsSession")
		}

		self.iamClient = iam.New(s)
	}

	return self.iamClient, nil
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

func (self *SRegion) rdsRequest(apiName string, params map[string]string, retval interface{}) error {
	return self.client.request(self.RegionId, RDS_SERVICE_NAME, RDS_SERVICE_ID, "2014-10-31", apiName, params, retval, true)
}

func (self *SRegion) ec2Request(apiName string, params map[string]string, retval interface{}) error {
	return self.client.request(self.RegionId, EC2_SERVICE_NAME, EC2_SERVICE_ID, "2016-11-15", apiName, params, retval, true)
}

func (self *SRegion) cloudWatchRequest(apiName string, params *cloudwatch.GetMetricStatisticsInput,
	retval interface{}) error {
	lock.Lock()
	session, err := self.getAwsSession()
	if err != nil {
		lock.Unlock()
		return err
	}
	lock.Unlock()
	c := session.ClientConfig(CLOUDWATCH_SERVICE_NAME)
	metadata := metadata.ClientInfo{
		ServiceName:   CLOUDWATCH_SERVICE_NAME,
		ServiceID:     CLOUDWATCH_SERVICE_ID,
		SigningName:   c.SigningName,
		SigningRegion: c.SigningRegion,
		Endpoint:      c.Endpoint,
		APIVersion:    "2010-08-01",
	}

	client := client.New(*c.Config, metadata, c.Handlers)
	client.Handlers.Sign.PushBackNamed(v4.SignRequestHandler)
	client.Handlers.Build.PushBackNamed(query.BuildHandler)
	client.Handlers.Unmarshal.PushBackNamed(query.UnmarshalHandler)
	client.Handlers.UnmarshalMeta.PushBackNamed(query.UnmarshalMetaHandler)
	client.Handlers.UnmarshalError.PushBackNamed(query.UnmarshalErrorHandler)
	return cloudWatchRequest(client, apiName, params, retval, false)
}

func (self *SRegion) GetElbV2Client() (*elbv2.ELBV2, error) {
	if self.elbv2Client == nil {
		s, err := self.getAwsSession()

		if err != nil {
			return nil, errors.Wrap(err, "getAwsSession")
		}

		self.elbv2Client = elbv2.New(s)
	}

	return self.elbv2Client, nil
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) fetchZones() error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	// todo: 这里将过滤出指定region下全部的zones。是否只过滤出可用的zone即可？ The state of the Availability Zone (available | information | impaired | unavailable)
	zones, err := ec2Client.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return errors.Wrap(err, "DescribeAvailabilityZones")
	}
	err = FillZero(zones)
	if err != nil {
		return err
	}

	self.izones = make([]cloudprovider.ICloudZone, 0)
	for _, zone := range zones.AvailabilityZones {
		self.izones = append(self.izones, &SZone{ZoneId: *zone.ZoneName, State: *zone.State, LocalName: *zone.ZoneName, region: self})
	}

	return nil
}

func (self *SRegion) fetchIVpcs() error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	vpcs, err := ec2Client.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return err
	}

	self.ivpcs = make([]cloudprovider.ICloudVpc, 0)
	for _, vpc := range vpcs.Vpcs {
		cidrBlockAssociationSet := []string{}
		for i := range vpc.CidrBlockAssociationSet {
			cidr := vpc.CidrBlockAssociationSet[i]
			if cidr.CidrBlockState.State != nil && *cidr.CidrBlockState.State == "associated" {
				cidrBlockAssociationSet = append(cidrBlockAssociationSet, *cidr.CidrBlock)
			}
		}

		tagspec := TagSpec{ResourceType: "vpc"}
		tagspec.LoadingEc2Tags(vpc.Tags)
		ivpc := &SVpc{region: self,
			CidrBlock:               *vpc.CidrBlock,
			CidrBlockAssociationSet: cidrBlockAssociationSet,
			IsDefault:               *vpc.IsDefault,
			RegionId:                self.RegionId,
			Status:                  *vpc.State,
			VpcId:                   *vpc.VpcId,
			VpcName:                 tagspec.GetNameTag(),
			InstanceTenancy:         *vpc.InstanceTenancy,
		}
		jsonutils.Update(&ivpc.AwsTags.TagSet, vpc.Tags)
		self.ivpcs = append(self.ivpcs, ivpc)
	}

	return nil
}

func (self *SRegion) fetchInfrastructure() error {
	if _, err := self.getEc2Client(); err != nil {
		return err
	}

	if err := self.fetchZones(); err != nil {
		return err
	}

	if err := self.fetchIVpcs(); err != nil {
		return err
	}

	for i := 0; i < len(self.ivpcs); i += 1 {
		for j := 0; j < len(self.izones); j += 1 {
			zone := self.izones[j].(*SZone)
			vpc := self.ivpcs[i].(*SVpc)
			wire := SWire{zone: zone, vpc: vpc}
			zone.addWire(&wire)
			vpc.addWire(&wire)
		}
	}
	return nil
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
	if self.izones == nil {
		if err := self.fetchInfrastructure(); err != nil {
			return nil, errors.Wrap(err, "fetchInfrastructure")
		}
	}
	return self.izones, nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if self.ivpcs == nil {
		err := self.fetchInfrastructure()
		if err != nil {
			return nil, errors.Wrap(err, "fetchInfrastructure")
		}
	}
	return self.ivpcs, nil
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
	snapshots, _, err := self.GetSnapshots("", "", "", []string{}, 0, 0)
	if err != nil {
		return nil, errors.Wrap(err, "GetSnapshots")
	}

	ret := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i += 1 {
		ret[i] = &snapshots[i]
	}
	return ret, nil
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}

	for _, zone := range izones {
		if zone.GetGlobalId() == id {
			return zone, nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIZoneById")
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, errors.Wrap(err, "GetIVpcs")
	}

	for _, vpc := range ivpcs {
		if vpc.GetGlobalId() == id {
			return vpc, nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIVpcById")
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			log.Errorf("GetIHostById %s: %s", id, err)
			return nil, errors.Wrap(err, "GetIHostById")
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
		if err == nil {
			return istore, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			log.Errorf("GetIStorageById %s: %s", id, err)
			return nil, errors.Wrap(err, "GetIStorageById")
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

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}

	if self.storageCache.GetGlobalId() == id {
		return self.storageCache, nil
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIStoragecacheById")

}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	tagspec := TagSpec{ResourceType: "vpc"}
	if len(opts.NAME) > 0 {
		tagspec.SetNameTag(opts.NAME)
	}

	if len(opts.Desc) > 0 {
		tagspec.SetDescTag(opts.Desc)
	}

	spec, err := tagspec.GetTagSpecifications()
	if err != nil {
		return nil, errors.Wrap(err, "GetTagSpecifications")
	}

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	// start create vpc
	vpc, err := ec2Client.CreateVpc(&ec2.CreateVpcInput{CidrBlock: &opts.CIDR})
	if err != nil {
		return nil, errors.Wrap(err, "CreateVpc")
	}

	tagsParams := &ec2.CreateTagsInput{Resources: []*string{vpc.Vpc.VpcId}, Tags: spec.Tags}
	_, err = ec2Client.CreateTags(tagsParams)
	if err != nil {
		log.Debugf("CreateIVpc add tag failed %s", err.Error())
	}

	err = self.fetchInfrastructure()
	if err != nil {
		return nil, errors.Wrap(err, "fetchInfrastructure")
	}
	return self.GetIVpcById(*vpc.Vpc.VpcId)
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

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, networkId string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.LocalName)
		net := z.getNetworkById(networkId)
		if net != nil {
			desc := &cloudprovider.SManagedVMCreateConfig{
				Name:              name,
				ExternalImageId:   imgId,
				SysDisk:           cloudprovider.SDiskInfo{SizeGB: 0, StorageType: storageType},
				Cpu:               cpu,
				MemoryMB:          memGB * 1024,
				ExternalNetworkId: networkId,
				DataDisks:         []cloudprovider.SDiskInfo{},
				PublicKey:         publicKey,
			}
			for _, sizeGB := range dataDiskSizesGB {
				desc.DataDisks = append(desc.DataDisks, cloudprovider.SDiskInfo{SizeGB: sizeGB, StorageType: storageType})
			}
			inst, err := z.getHost().CreateVM(desc)
			if err != nil {
				return nil, errors.Wrap(err, "CreateVM")
			}
			return inst.(*SInstance), nil
		}
	}
	return nil, fmt.Errorf("cannot find vswitch %s", networkId)
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetElbV2Client")
	}

	params := &elbv2.DescribeLoadBalancersInput{}
	ret, err := client.DescribeLoadBalancers(params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeLoadBalancers")
	}

	result := make([]SElb, 0)
	err = unmarshalAwsOutput(ret, "LoadBalancers", &result)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput.LoadBalancers")
	}

	ielbs := make([]cloudprovider.ICloudLoadbalancer, len(result))
	for i := range result {
		result[i].region = self
		ielbs[i] = &result[i]
	}

	return ielbs, nil
}

func (self *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetElbV2Client")
	}

	params := &elbv2.DescribeLoadBalancersInput{}
	params.SetLoadBalancerArns([]*string{&loadbalancerId})
	ret, err := client.DescribeLoadBalancers(params)
	if err != nil {
		if strings.Contains(err.Error(), "LoadBalancerNotFound") {
			return nil, cloudprovider.ErrNotFound
		}

		return nil, errors.Wrap(err, "DescribeLoadBalancers")
	}

	elbs := []SElb{}
	err = unmarshalAwsOutput(ret, "LoadBalancers", &elbs)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput.LoadBalancers")
	}

	if len(elbs) == 1 {
		elbs[0].region = self
		return &elbs[0], nil
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetILoadBalancerById")
}

func (self *SRegion) getElbAttributesById(loadbalancerId string) (map[string]string, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetElbV2Client")
	}

	params := &elbv2.DescribeLoadBalancerAttributesInput{}
	params.SetLoadBalancerArn(loadbalancerId)
	output, err := client.DescribeLoadBalancerAttributes(params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeLoadBalancerAttributes")
	}

	attrs := []map[string]string{}
	err = unmarshalAwsOutput(output, "Attributes", &attrs)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput.Attributes")
	}

	ret := map[string]string{}
	for i := range attrs {
		for k, v := range attrs[i] {
			ret[k] = v
		}
	}

	return ret, nil
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

func (self *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	client, err := self.getIamClient()
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateILoadBalancerCertificate.getIamClient")
	}

	params := &iam.UploadServerCertificateInput{}
	params.SetServerCertificateName(cert.Name)
	params.SetPrivateKey(cert.PrivateKey)
	params.SetCertificateBody(cert.Certificate)
	ret, err := client.UploadServerCertificate(params)
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateILoadBalancerCertificate.UploadServerCertificate")
	}

	// wait upload cert success
	err = cloudprovider.Wait(5*time.Second, 30*time.Second, func() (bool, error) {
		_, err := self.GetILoadBalancerCertificateById(*ret.ServerCertificateMetadata.Arn)
		if err == nil {
			return true, nil
		}

		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		} else {
			return false, err
		}
	})
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateILoadBalancerCertificate.Wait")
	}

	return self.GetILoadBalancerCertificateById(*ret.ServerCertificateMetadata.Arn)
}

func (self *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	client, err := self.getIamClient()
	if err != nil {
		return nil, errors.Wrap(err, "getIamClient")
	}

	params := &iam.ListServerCertificatesInput{}
	ret, err := client.ListServerCertificates(params)
	if err != nil {
		return nil, errors.Wrap(err, "ListServerCertificates")
	}

	certs := []SElbCertificate{}
	err = unmarshalAwsOutput(ret, "ServerCertificateMetadataList", &certs)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput.ServerCertificateMetadataList")
	}

	icerts := make([]cloudprovider.ICloudLoadbalancerCertificate, len(certs))
	for i := range certs {
		certs[i].region = self
		icerts[i] = &certs[i]
	}

	return icerts, nil
}

func (self *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetElbV2Client")
	}

	params := &elbv2.CreateLoadBalancerInput{}
	params.SetName(loadbalancer.Name)
	params.SetType(loadbalancer.LoadbalancerSpec)
	params.SetIpAddressType("ipv4")
	if loadbalancer.AddressType == api.LB_ADDR_TYPE_INTERNET {
		params.SetScheme("internet-facing")
	} else {
		params.SetScheme("internal")
	}

	// params.SetSecurityGroups()
	params.SetSubnets(ConvertedList(loadbalancer.NetworkIDs))

	tagInput := []*elbv2.Tag{}
	keys := []string{}
	values := []string{}
	for k, v := range loadbalancer.Tags {
		keys = append(keys, k)
		values = append(values, v)
	}
	for i := range keys {
		tagInput = append(tagInput, &elbv2.Tag{
			Key:   &keys[i],
			Value: &values[i],
		})
	}
	if len(loadbalancer.Tags) > 0 {
		params.SetTags(tagInput)
	}

	ret, err := client.CreateLoadBalancer(params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateLoadBalancer")
	}

	elbs := []SElb{}
	err = unmarshalAwsOutput(ret, "LoadBalancers", &elbs)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput.LoadBalancers")
	}

	if len(elbs) == 1 {
		elbs[0].region = self
		return &elbs[0], nil
	}

	return nil, fmt.Errorf("CreateILoadBalancer error %#v", elbs)
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

func (self *SRegion) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	backendgroups, err := self.GetElbBackendgroups("", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetElbBackendgroups")
	}

	ret := make([]cloudprovider.ICloudLoadbalancerBackendGroup, len(backendgroups))
	for i := range backendgroups {
		ret[i] = &backendgroups[i]
	}

	return ret, nil
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	secgroups, total, err := self.GetSecurityGroups("", "", secgroupId, 0, 1)
	if err != nil {
		return nil, errors.Wrap(err, "GetSecurityGroups")
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &secgroups[0], nil
}

func (self *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	secgroups, total, err := self.GetSecurityGroups(opts.VpcId, opts.Name, "", 0, 1)
	if err != nil {
		return nil, errors.Wrap(err, "GetSecurityGroups")
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &secgroups[0], nil
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	groupId, err := self.CreateSecurityGroup(conf.VpcId, conf.Name, "", conf.Desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSecurityGroup")
	}
	return self.GetISecurityGroupById(groupId)
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) CreateInternetGateway() (cloudprovider.ICloudInternetGateway, error) {
	ec2Client, err := region.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}
	input := ec2.CreateInternetGatewayInput{}
	output, err := ec2Client.CreateInternetGateway(&input)
	if err != nil {
		return nil, errors.Wrap(err, "CreateInternetGateway")
	}

	ret := &SInternetGateway{}
	ret.region = region
	err = unmarshalAwsOutput(output, "InternetGateway", ret)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput")
	}

	return ret, nil
}
