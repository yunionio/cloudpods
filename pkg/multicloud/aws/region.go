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
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/private/protocol/query"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

var RegionLocations = map[string]string{
	"us-east-2":      "美国东部(俄亥俄州)",
	"us-east-1":      "美国东部(弗吉尼亚北部)",
	"us-west-1":      "美国西部(加利福尼亚北部)",
	"us-west-2":      "美国西部(俄勒冈)",
	"ap-south-1":     "亚太地区(孟买)",
	"ap-northeast-2": "亚太区域(首尔)",
	"ap-northeast-3": "亚太区域(大阪)",
	"ap-southeast-1": "亚太区域(新加坡)",
	"ap-southeast-2": "亚太区域(悉尼)",
	"ap-northeast-1": "亚太区域(东京)",
	"ca-central-1":   "加拿大(中部)",
	"cn-north-1":     "中国(北京)",
	"cn-northwest-1": "中国(宁夏)",
	"eu-central-1":   "欧洲(法兰克福)",
	"eu-west-1":      "欧洲(爱尔兰)",
	"eu-west-2":      "欧洲(伦敦)",
	"eu-west-3":      "欧洲(巴黎)",
	"sa-east-1":      "南美洲(圣保罗)",
	"us-gov-west-1":  "AWS GovCloud(美国)",
}

const (
	RDS_SERVICE_NAME = "rds"
	RDS_SERVICE_ID   = "RDS"

	EC2_SERVICE_NAME = "ec2"
	EC2_SERVICE_ID   = "EC2"
)

type SRegion struct {
	multicloud.SRegion

	client      *SAwsClient
	ec2Client   *ec2.EC2
	iamClient   *iam.IAM
	s3Client    *s3.S3
	elbv2Client *elbv2.ELBV2
	acmClient   *acm.ACM

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache  *SStoragecache
	instanceTypes []SInstanceType

	RegionEndpoint string
	RegionId       string // 这里为保持一致沿用阿里云RegionId的叫法, 与AWS RegionName字段对应
}

/////////////////////////////////////////////////////////////////////////////
/* 请不要使用这个client(AWS_DEFAULT_REGION)跨region查信息.有可能导致查询返回的信息为空。比如DescribeAvailabilityZones*/
func (self *SRegion) GetClient() *SAwsClient {
	return self.client
}

func (self *SRegion) getAwsSession() (*session.Session, error) {
	return self.client.getAwsSession(self.RegionId)
}

func (self *SRegion) getEc2Client() (*ec2.EC2, error) {
	if self.ec2Client == nil {
		s, err := self.getAwsSession()

		if err != nil {
			return nil, err
		}

		self.ec2Client = ec2.New(s)
		return self.ec2Client, nil
	}

	return self.ec2Client, nil
}

func (self *SRegion) getIamClient() (*iam.IAM, error) {
	if self.iamClient == nil {
		s, err := self.getAwsSession()

		if err != nil {
			return nil, err
		}

		self.iamClient = iam.New(s)
	}

	return self.iamClient, nil
}

func (self *SRegion) GetS3Client() (*s3.S3, error) {
	if self.s3Client == nil {
		s, err := self.getAwsSession()
		if err != nil {
			return nil, err
		}
		self.s3Client = s3.New(s)
	}
	return self.s3Client, nil
}

var UnmarshalHandler = request.NamedHandler{Name: "yunion.query.Unmarshal", Fn: Unmarshal}

func Unmarshal(r *request.Request) {
	defer r.HTTPResponse.Body.Close()
	if r.DataFilled() {
		var decoder *xml.Decoder
		if DEBUG {
			body, err := ioutil.ReadAll(r.HTTPResponse.Body)
			if err != nil {
				r.Error = awserr.NewRequestFailure(
					awserr.New("ioutil.ReadAll", "read response body", err),
					r.HTTPResponse.StatusCode,
					r.RequestID,
				)
				return
			}
			log.Debugf("response: \n%s", string(body))
			decoder = xml.NewDecoder(strings.NewReader(string(body)))
		} else {
			decoder = xml.NewDecoder(r.HTTPResponse.Body)
		}
		if r.ClientInfo.ServiceID == EC2_SERVICE_ID {
			err := decoder.Decode(r.Data)
			if err != nil {
				r.Error = awserr.NewRequestFailure(
					awserr.New("SerializationError", "failed decoding EC2 Query response", err),
					r.HTTPResponse.StatusCode,
					r.RequestID,
				)
			}
			return
		}
		for {
			tok, err := decoder.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				r.Error = awserr.NewRequestFailure(
					awserr.New("decoder.Token()", "get token", err),
					r.HTTPResponse.StatusCode,
					r.RequestID,
				)
				return
			}

			if tok == nil {
				break
			}

			switch typed := tok.(type) {
			case xml.CharData:
				continue
			case xml.StartElement:
				if typed.Name.Local == r.Operation.Name+"Result" {
					err = decoder.DecodeElement(r.Data, &typed)
					if err != nil {
						r.Error = awserr.NewRequestFailure(
							awserr.New("DecodeElement", "failed decoding Query response", err),
							r.HTTPResponse.StatusCode,
							r.RequestID,
						)
					}
					return
				}
			case xml.EndElement:
				break
			}
		}

	}
}

var buildHandler = request.NamedHandler{Name: "yunion.query.Build", Fn: Build}

func Build(r *request.Request) {
	body := url.Values{
		"Action":  {r.Operation.Name},
		"Version": {r.ClientInfo.APIVersion},
	}
	if r.Params != nil {
		if params, ok := r.Params.(map[string]string); ok {
			for k, v := range params {
				body.Add(k, v)
			}
		}
	}

	if DEBUG {
		log.Debugf("params: %s", body.Encode())
	}

	if !r.IsPresigned() {
		r.HTTPRequest.Method = "POST"
		r.HTTPRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		r.SetBufferBody([]byte(body.Encode()))
	} else { // This is a pre-signed request
		r.HTTPRequest.Method = "GET"
		r.HTTPRequest.URL.RawQuery = body.Encode()
	}
}

func (self *SRegion) rdsRequest(apiName string, params map[string]string, retval interface{}) error {
	session, err := self.getAwsSession()
	if err != nil {
		return err
	}
	c := session.ClientConfig(RDS_SERVICE_NAME)
	metadata := metadata.ClientInfo{
		ServiceName:   RDS_SERVICE_NAME,
		ServiceID:     RDS_SERVICE_ID,
		SigningName:   c.SigningName,
		SigningRegion: c.SigningRegion,
		Endpoint:      c.Endpoint,
		APIVersion:    "2014-10-31",
	}

	if self.client.debug {
		logLevel := aws.LogLevelType(uint(aws.LogDebugWithRequestErrors) + uint(aws.LogDebugWithHTTPBody))
		c.Config.LogLevel = &logLevel
	}

	client := client.New(*c.Config, metadata, c.Handlers)
	client.Handlers.Sign.PushBackNamed(v4.SignRequestHandler)
	client.Handlers.Build.PushBackNamed(buildHandler)
	client.Handlers.Unmarshal.PushBackNamed(UnmarshalHandler)
	client.Handlers.UnmarshalMeta.PushBackNamed(query.UnmarshalMetaHandler)
	client.Handlers.UnmarshalError.PushBackNamed(query.UnmarshalErrorHandler)
	return jsonRequest(client, apiName, params, retval, true)
}

func (self *SRegion) ec2Request(apiName string, params map[string]string, retval interface{}) error {
	session, err := self.getAwsSession()
	if err != nil {
		return err
	}
	c := session.ClientConfig(EC2_SERVICE_NAME)
	metadata := metadata.ClientInfo{
		ServiceName:   EC2_SERVICE_NAME,
		ServiceID:     EC2_SERVICE_ID,
		SigningName:   c.SigningName,
		SigningRegion: c.SigningRegion,
		Endpoint:      c.Endpoint,
		APIVersion:    "2016-11-15",
	}

	requestErr := aws.LogDebugWithRequestErrors

	c.Config.LogLevel = &requestErr

	client := client.New(*c.Config, metadata, c.Handlers)
	client.Handlers.Sign.PushBackNamed(v4.SignRequestHandler)
	client.Handlers.Build.PushBackNamed(buildHandler)
	client.Handlers.Unmarshal.PushBackNamed(UnmarshalHandler)
	client.Handlers.UnmarshalMeta.PushBackNamed(query.UnmarshalMetaHandler)
	client.Handlers.UnmarshalError.PushBackNamed(query.UnmarshalErrorHandler)
	return jsonRequest(client, apiName, params, retval, true)
}

func (self *SRegion) GetElbV2Client() (*elbv2.ELBV2, error) {
	if self.elbv2Client == nil {
		s, err := self.getAwsSession()

		if err != nil {
			return nil, err
		}

		self.elbv2Client = elbv2.New(s)
	}

	return self.elbv2Client, nil
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) fetchZones() error {
	// todo: 这里将过滤出指定region下全部的zones。是否只过滤出可用的zone即可？ The state of the Availability Zone (available | information | impaired | unavailable)
	zones, err := self.ec2Client.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return err
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
	vpcs, err := self.ec2Client.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return err
	}

	self.ivpcs = make([]cloudprovider.ICloudVpc, 0)
	for _, vpc := range vpcs.Vpcs {
		tags := make(map[string]string, 0)
		for _, tag := range vpc.Tags {
			tags[*tag.Key] = *tag.Value
		}

		self.ivpcs = append(self.ivpcs, &SVpc{region: self,
			CidrBlock:       *vpc.CidrBlock,
			Tags:            tags,
			IsDefault:       *vpc.IsDefault,
			RegionId:        self.RegionId,
			Status:          *vpc.State,
			VpcId:           *vpc.VpcId,
			InstanceTenancy: *vpc.InstanceTenancy,
		})
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

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_AWS, self.RegionId)
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

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
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
			return nil, err
		}
	}
	return self.izones, nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if self.ivpcs == nil {
		err := self.fetchInfrastructure()
		if err != nil {
			return nil, err
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
	_, err := self.getEc2Client()
	if err != nil {
		return nil, err
	}

	eips, total, err := self.GetEips("", "", 0, 0)
	if err != nil {
		return nil, err
	}

	ret := make([]cloudprovider.ICloudEIP, total)
	for i := 0; i < len(eips); i += 1 {
		ret[i] = &eips[i]
	}
	return ret, nil
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, _, err := self.GetSnapshots("", "", "", []string{}, 0, 0)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	for _, zone := range izones {
		if zone.GetGlobalId() == id {
			return zone, nil
		}
	}

	return nil, ErrorNotFound()
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}

	for _, vpc := range ivpcs {
		if vpc.GetGlobalId() == id {
			return vpc, nil
		}
	}

	return nil, ErrorNotFound()
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, ErrorNotFound()
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		istore, err := izones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, ErrorNotFound()
}

func (self *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := make([]cloudprovider.ICloudHost, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneHost, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, iZoneHost...)
	}
	return iHosts, nil
}

func (self *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneStores, err := izones[i].GetIStorages()
		if err != nil {
			return nil, err
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
	return nil, ErrorNotFound()

}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	tagspec := TagSpec{ResourceType: "vpc"}
	if len(name) > 0 {
		tagspec.SetNameTag(name)
	}

	if len(desc) > 0 {
		tagspec.SetDescTag(desc)
	}

	spec, err := tagspec.GetTagSpecifications()
	if err != nil {
		return nil, err
	}

	// start create vpc
	vpc, err := self.ec2Client.CreateVpc(&ec2.CreateVpcInput{CidrBlock: &cidr})
	if err != nil {
		return nil, err
	}

	tagsParams := &ec2.CreateTagsInput{Resources: []*string{vpc.Vpc.VpcId}, Tags: spec.Tags}
	_, err = self.ec2Client.CreateTags(tagsParams)
	if err != nil {
		log.Debugf("CreateIVpc add tag failed %s", err.Error())
	}

	err = self.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	return self.GetIVpcById(*vpc.Vpc.VpcId)
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	eips, total, err := self.GetEips(eipId, "", 0, 0)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, ErrorNotFound()
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &eips[0], nil
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_AWS
}

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, networkId string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
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
				return nil, err
			}
			return inst.(*SInstance), nil
		}
	}
	return nil, fmt.Errorf("cannot find vswitch %s", networkId)
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeLoadBalancersInput{}
	ret, err := client.DescribeLoadBalancers(params)
	if err != nil {
		return nil, err
	}

	result := make([]SElb, 0)
	err = unmarshalAwsOutput(ret, "LoadBalancers", &result)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	params := &elbv2.DescribeLoadBalancersInput{}
	params.SetLoadBalancerArns([]*string{&loadbalancerId})
	ret, err := client.DescribeLoadBalancers(params)
	if err != nil {
		if strings.Contains(err.Error(), "LoadBalancerNotFound") {
			return nil, cloudprovider.ErrNotFound
		}

		return nil, err
	}

	elbs := []SElb{}
	err = unmarshalAwsOutput(ret, "LoadBalancers", &elbs)
	if err != nil {
		return nil, err
	}

	if len(elbs) == 1 {
		elbs[0].region = self
		return &elbs[0], nil
	}

	return nil, ErrorNotFound()
}

func (self *SRegion) getElbAttributesById(loadbalancerId string) (map[string]string, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeLoadBalancerAttributesInput{}
	params.SetLoadBalancerArn(loadbalancerId)
	output, err := client.DescribeLoadBalancerAttributes(params)
	if err != nil {
		return nil, err
	}

	attrs := []map[string]string{}
	err = unmarshalAwsOutput(output, "Attributes", &attrs)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	for i := range certs {
		if certs[i].GetId() == certId {
			return certs[i], nil
		}
	}

	return nil, ErrorNotFound()
}

func (self *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	client, err := self.getIamClient()
	if err != nil {
		return nil, err
	}

	params := &iam.UploadServerCertificateInput{}
	params.SetServerCertificateName(cert.Name)
	params.SetPrivateKey(cert.PrivateKey)
	params.SetCertificateBody(cert.Certificate)
	ret, err := client.UploadServerCertificate(params)
	if err != nil {
		return nil, err
	}
	// wait 3 second after upload cert
	time.Sleep(3 * time.Second)
	return self.GetILoadBalancerCertificateById(*ret.ServerCertificateMetadata.Arn)
}

func (self *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	client, err := self.getIamClient()
	if err != nil {
		return nil, err
	}

	params := &iam.ListServerCertificatesInput{}
	ret, err := client.ListServerCertificates(params)
	if err != nil {
		return nil, err
	}

	certs := []SElbCertificate{}
	err = unmarshalAwsOutput(ret, "ServerCertificateMetadataList", &certs)
	if err != nil {
		return nil, err
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
		return nil, err
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
	ret, err := client.CreateLoadBalancer(params)
	if err != nil {
		return nil, err
	}

	elbs := []SElb{}
	err = unmarshalAwsOutput(ret, "LoadBalancers", &elbs)
	if err != nil {
		return nil, err
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
	input.CreateBucketConfiguration = &s3.CreateBucketConfiguration{}
	input.CreateBucketConfiguration.SetLocationConstraint(region.GetId())
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
		return nil, err
	}

	ret := make([]cloudprovider.ICloudLoadbalancerBackendGroup, len(backendgroups))
	for i := range backendgroups {
		ret[i] = &backendgroups[i]
	}

	return ret, nil
}
