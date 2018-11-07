package aliyun

import (
	"fmt"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SRegion struct {
	client    *SAliyunClient
	ecsClient *sdk.Client
	ossClient *oss.Client

	RegionId  string
	LocalName string

	izones []cloudprovider.ICloudZone

	ivpcs []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	instanceTypes []SInstanceType

	latitude      float64
	longitude     float64
	fetchLocation bool
}

func (self *SRegion) GetClient() *SAliyunClient {
	return self.client
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) getEcsClient() (*sdk.Client, error) {
	if self.ecsClient == nil {
		cli, err := sdk.NewClientWithAccessKey(self.RegionId, self.client.accessKey, self.client.secret)
		if err != nil {
			return nil, err
		}
		self.ecsClient = cli
	}
	return self.ecsClient, nil
}

// oss endpoint
// https://help.aliyun.com/document_detail/31837.html?spm=a2c4g.11186623.2.6.6E8ZkO
func (self *SRegion) GetOSSExternalDomain() string {
	return fmt.Sprintf("oss-%s.aliyuncs.com", self.RegionId)
}

func (self *SRegion) GetOSSInternalDomain() string {
	return fmt.Sprintf("oss-%s-internal.aliyuncs.com", self.RegionId)
}

func (self *SRegion) GetOssClient() (*oss.Client, error) {
	if self.ossClient == nil {
		// https://help.aliyun.com/document_detail/31837.html?spm=a2c4g.11186623.2.6.XqEgD1
		ep := self.GetOSSExternalDomain()
		cli, err := oss.New(ep, self.client.accessKey, self.client.secret)
		if err != nil {
			return nil, err
		}
		self.ossClient = cli
	}
	return self.ossClient, nil
}

func (self *SRegion) ecsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getEcsClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, apiName, params)
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) GetId() string {
	return self.RegionId
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_ALIYUN_CN, self.LocalName)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_ALIYUN, self.RegionId)
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_ALIYUN
}

func (self *SRegion) GetLatitude() float32 {
	if locationInfo, ok := LatitudeAndLongitude[self.RegionId]; ok {
		return locationInfo["latitude"]
	}
	return 0.0
}

func (self *SRegion) GetLongitude() float32 {
	if locationInfo, ok := LatitudeAndLongitude[self.RegionId]; ok {
		return locationInfo["longitude"]
	}
	return 0.0
}

func (self *SRegion) GetStatus() string {
	return models.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if self.izones == nil {
		var err error
		err = self.fetchInfrastructure()
		if err != nil {
			return nil, err
		}
	}
	return self.izones, nil
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) _fetchZones(chargeType InstanceChargeType, spotStrategy SpotStrategyType) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	if len(chargeType) > 0 {
		params["InstanceChargeType"] = string(chargeType)
	}
	if len(spotStrategy) > 0 {
		params["SpotStrategy"] = string(spotStrategy)
	}
	body, err := self.ecsRequest("DescribeZones", params)
	if err != nil {
		return err
	}

	zones := make([]SZone, 0)
	err = body.Unmarshal(&zones, "Zones", "Zone")
	if err != nil {
		return err
	}

	self.izones = make([]cloudprovider.ICloudZone, len(zones))

	for i := 0; i < len(zones); i += 1 {
		zones[i].region = self
		self.izones[i] = &zones[i]
	}

	return nil
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		zone := izones[i].(*SZone)
		if zone.ZoneId == id {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
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

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(ivpcs); i += 1 {
		if ivpcs[i].GetGlobalId() == id {
			return ivpcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) fetchIVpcs() error {
	vpcs := make([]SVpc, 0)
	for {
		part, total, err := self.GetVpcs(nil, len(vpcs), 50)
		if err != nil {
			return err
		}
		vpcs = append(vpcs, part...)
		if len(vpcs) >= total {
			break
		}
	}
	self.ivpcs = make([]cloudprovider.ICloudVpc, len(vpcs))
	for i := 0; i < len(vpcs); i += 1 {
		vpcs[i].region = self
		self.ivpcs[i] = &vpcs[i]
	}
	return nil
}

func (self *SRegion) fetchInfrastructure() error {
	err := self._fetchZones(PostPaidInstanceChargeType, NoSpotStrategy)
	if err != nil {
		return err
	}
	err = self.fetchIVpcs()
	if err != nil {
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

func (self *SRegion) GetVpcs(vpcId []string, offset int, limit int) ([]SVpc, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	if vpcId != nil && len(vpcId) > 0 {
		params["VpcId"] = strings.Join(vpcId, ",")
	}

	body, err := self.ecsRequest("DescribeVpcs", params)
	if err != nil {
		log.Errorf("GetVpcs fail %s", err)
		return nil, 0, err
	}

	vpcs := make([]SVpc, 0)
	err = body.Unmarshal(&vpcs, "Vpcs", "Vpc")
	if err != nil {
		log.Errorf("Unmarshal vpc fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return vpcs, int(total), nil
}

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	vpcs, total, err := self.GetVpcs([]string{vpcId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	vpcs[0].region = self
	return &vpcs[0], nil
}

func (self *SRegion) GetVRouters(offset int, limit int) ([]SVRouter, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	body, err := self.ecsRequest("DescribeVRouters", params)
	if err != nil {
		log.Errorf("GetVRouters fail %s", err)
		return nil, 0, err
	}

	vrouters := make([]SVRouter, 0)
	err = body.Unmarshal(&vrouters, "VRouters", "VRouter")
	if err != nil {
		log.Errorf("Unmarshal vrouter fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return vrouters, int(total), nil
}

func (self *SRegion) GetRouteTables(ids []string, offset int, limit int) ([]SRouteTable, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	if ids != nil && len(ids) > 0 {
		params["RouteTableId"] = strings.Join(ids, ",")
	}

	body, err := self.ecsRequest("DescribeRouteTables", params)
	if err != nil {
		log.Errorf("GetRouteTables fail %s", err)
		return nil, 0, err
	}

	routetables := make([]SRouteTable, 0)
	err = body.Unmarshal(&routetables, "RouteTables", "RouteTable")
	if err != nil {
		log.Errorf("Unmarshal routetables fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return routetables, int(total), nil
}

func (self *SRegion) GetVSwitches(ids []string, vpcId string, offset int, limit int) ([]SVSwitch, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	if ids != nil && len(ids) > 0 {
		params["VSwitchId"] = strings.Join(ids, ",")
	}
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}

	body, err := self.ecsRequest("DescribeVSwitches", params)
	if err != nil {
		log.Errorf("GetVSwitches fail %s", err)
		return nil, 0, err
	}

	switches := make([]SVSwitch, 0)
	err = body.Unmarshal(&switches, "VSwitches", "VSwitch")
	if err != nil {
		log.Errorf("Unmarshal vswitches fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return switches, int(total), nil
}

func (self *SRegion) GetMatchInstanceTypes(cpu int, memMB int, gpu int, zoneId string) ([]SInstanceType, error) {
	if self.instanceTypes == nil {
		types, err := self.GetInstanceTypes()
		if err != nil {
			log.Errorf("GetInstanceTypes %s", err)
			return nil, err
		}
		self.instanceTypes = types
	}
	var available []string
	if len(zoneId) > 0 {
		zone, err := self.getZoneById(zoneId)
		if err != nil {
			return nil, err
		}
		available = zone.AvailableInstanceTypes.InstanceTypes
	}
	ret := make([]SInstanceType, 0)
	for _, t := range self.instanceTypes {
		if t.CpuCoreCount == cpu && memMB == t.memoryMB() && gpu == t.GPUAmount {
			if available == nil || utils.IsInStringArray(t.InstanceTypeId, available) {
				ret = append(ret, t)
			}
		}
	}
	return ret, nil
}

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, vswitchId string, passwd string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.LocalName)
		net := z.getNetworkById(vswitchId)
		if net != nil {
			inst, err := z.getHost().CreateVM(name, imgId, 0, cpu, memGB*1024, vswitchId, "", "", passwd, storageType, dataDiskSizesGB, publicKey, "", "")
			if err != nil {
				return nil, err
			}
			return inst.(*SInstance), nil
		}
	}
	return nil, fmt.Errorf("cannot find vswitch %s", vswitchId)
}

func (self *SRegion) instanceOperation(instanceId string, opname string, extra map[string]string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	if extra != nil && len(extra) > 0 {
		for k, v := range extra {
			params[k] = v
		}
	}
	_, err := self.ecsRequest(opname, params)
	return err
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.Status, nil
}

func (self *SRegion) GetInstanceVNCUrl(instanceId string) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	body, err := self.ecsRequest("DescribeInstanceVncUrl", params)
	if err != nil {
		return "", err
	}
	return body.GetString("VncUrl")
}

func (self *SRegion) ModifyInstanceVNCUrlPassword(instanceId string, passwd string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	params["VncPassword"] = passwd // must be 6 digital + alphabet
	_, err := self.ecsRequest("ModifyInstanceVncPasswd", params)
	return err
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	params := make(map[string]string)
	if len(cidr) > 0 {
		params["CidrBlock"] = cidr
	}
	if len(name) > 0 {
		params["VpcName"] = name
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	params["ClientToken"] = utils.GenRequestId(20)
	body, err := self.ecsRequest("CreateVpc", params)
	if err != nil {
		return nil, err
	}
	vpcId, err := body.GetString("VpcId")
	if err != nil {
		return nil, err
	}
	err = self.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	return self.GetIVpcById(vpcId)
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := make(map[string]string)
	params["VpcId"] = vpcId

	_, err := self.ecsRequest("DeleteVpc", params)
	return err
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
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return self.storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) updateInstance(instId string, name, desc, passwd, hostname, userData string) error {
	params := make(map[string]string)
	params["InstanceId"] = instId
	if len(name) > 0 {
		params["InstanceName"] = name
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	if len(passwd) > 0 {
		params["Password"] = passwd
	}
	if len(hostname) > 0 {
		params["HostName"] = hostname
	}
	if len(userData) > 0 {
		params["UserData"] = userData
	}
	_, err := self.ecsRequest("ModifyInstanceAttribute", params)
	return err
}

func (self *SRegion) UpdateInstancePassword(instId string, passwd string) error {
	return self.updateInstance(instId, "", "", passwd, "", "")
}

// func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
// 	eips, total, err := self.GetSnapshots("", 0, 50)
// 	if err != nil {
// 		return nil, err
// 	}
// 	for len(eips) < total {
// 		var parts []SEipAddress
// 		parts, total, err = self.GetEips("", len(eips), 50)
// 		if err != nil {
// 			return nil, err
// 		}
// 		eips = append(eips, parts...)
// 	}
// 	ret := make([]cloudprovider.ICloudEIP, len(eips))
// 	for i := 0; i < len(eips); i += 1 {
// 		ret[i] = &eips[i]
// 	}
// 	return ret, nil
// }

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, total, err := self.GetEips("", 0, 50)
	if err != nil {
		return nil, err
	}
	for len(eips) < total {
		var parts []SEipAddress
		parts, total, err = self.GetEips("", len(eips), 50)
		if err != nil {
			return nil, err
		}
		eips = append(eips, parts...)
	}
	ret := make([]cloudprovider.ICloudEIP, len(eips))
	for i := 0; i < len(eips); i += 1 {
		ret[i] = &eips[i]
	}
	return ret, nil
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	eips, total, err := self.GetEips(eipId, 0, 1)
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
