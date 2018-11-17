package aws

import (
	"fmt"

	sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

var RegionLocations map[string]string = map[string]string{
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

type SRegion struct {
	client    *SAwsClient
	ec2Client *ec2.EC2
	iamClient *iam.IAM
	s3Client  *s3.S3

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
	return session.NewSession(&sdk.Config{
		Region:      sdk.String(self.RegionId),
		Credentials: credentials.NewStaticCredentials(self.client.accessKey, self.client.secret, ""),
	})
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

func (self *SRegion) getS3Client() (*s3.S3, error) {
	if self.s3Client == nil {
		s, err := self.getAwsSession()

		if err != nil {
			return nil, err
		}

		self.s3Client = s3.New(s)
	}

	return self.s3Client, nil
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
			CidrBlock: *vpc.CidrBlock,
			Tags:      tags,
			IsDefault: *vpc.IsDefault,
			RegionId:  self.RegionId,
			Status:    *vpc.State,
			VpcId:     *vpc.VpcId,
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
	return models.CLOUD_REGION_STATUS_INSERVER
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

func (self *SRegion) GetLatitude() float32 {
	if data, ok := LatitudeAndLongitude[self.RegionId]; !ok {
		log.Debugf("Region %s not found in LatitudeAndLongitude", self.RegionId)
		return 0.0
	} else if lat, ok := data["latitude"]; !ok {
		log.Debugf("Region %s's latitude not found in LatitudeAndLongitude", self.RegionId)
		return 0.0
	} else {
		return lat
	}
}

func (self *SRegion) GetLongitude() float32 {
	if data, ok := LatitudeAndLongitude[self.RegionId]; !ok {
		log.Debugf("Region %s not found in LatitudeAndLongitude", self.RegionId)
		return 0.0
	} else if lat, ok := data["longitude"]; !ok {
		log.Debugf("Region %s's latitude not found in LatitudeAndLongitude", self.RegionId)
		return 0.0
	} else {
		return lat
	}
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

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	_, err := self.getEc2Client()
	if err != nil {
		return nil, err
	}

	eips, total, err := self.GetEips("", 0, 0)
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

	return nil, cloudprovider.ErrNotFound
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

	return nil, cloudprovider.ErrNotFound
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
	return nil, cloudprovider.ErrNotFound
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
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}

	if self.storageCache.GetGlobalId() == id {
		return self.storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound

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
	eips, total, err := self.GetEips(eipId, 0, 0)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
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
			inst, err := z.getHost().CreateVM(name, imgId, 0, cpu, memGB*1024, networkId, "", "", "", storageType, dataDiskSizesGB, publicKey, "", "")
			if err != nil {
				return nil, err
			}
			return inst.(*SInstance), nil
		}
	}
	return nil, fmt.Errorf("cannot find vswitch %s", networkId)
}
