package qcloud

import (
	"fmt"

	"github.com/nelsonken/cos-go-sdk-v5/cos"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SRegion struct {
	client    *SQcloudClient
	cosClient *cos.Client

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	instanceTypes []SInstanceType

	Region      string
	RegionName  string
	RegionState string

	Latitude      float64
	Longitude     float64
	fetchLocation bool
}

func (self *SRegion) GetId() string {
	return self.Region
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_QCLOUD_CN, self.RegionName)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_QCLOUD, self.Region)
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_QCLOUD
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	params := make(map[string]string)
	if len(cidr) > 0 {
		params["CidrBlock"] = cidr
	}
	if len(name) > 0 {
		params["VpcName"] = name
	}
	body, err := self.vpcRequest("CreateVpc", params)
	if err != nil {
		return nil, err
	}
	vpcId, err := body.GetString("Vpc", "VpcId")
	if err != nil {
		return nil, err
	}
	err = self.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	return self.GetIVpcById(vpcId)
}

func (self *SRegion) GetCosClient() (*cos.Client, error) {
	if self.cosClient == nil {
		self.cosClient = cos.New(&cos.Option{
			AppID:     self.client.AppID,
			SecretID:  self.client.SecretID,
			SecretKey: self.client.SecretKey,
			Region:    self.Region,
		})
	}
	return self.cosClient, nil
}

func (self *SRegion) GetClient() *SQcloudClient {
	return self.client
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	if len(eipId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	eips, total, err := self.GetEips(eipId, "", 0, 1)
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

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, total, err := self.GetEips("", "", 0, 50)
	if err != nil {
		return nil, err
	}
	for len(eips) < total {
		var parts []SEipAddress
		parts, total, err = self.GetEips("", "", len(eips), 50)
		if err != nil {
			return nil, err
		}
		eips = append(eips, parts...)
	}
	ret := make([]cloudprovider.ICloudEIP, len(eips))
	for i := 0; i < len(eips); i++ {
		ret[i] = &eips[i]
	}
	return ret, nil
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
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return self.storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (self *SRegion) updateInstance(instId string, name, desc, passwd, hostname string) error {
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
	_, err := self.cvmRequest("ModifyInstanceAttribute", params)
	return err
}

func (self *SRegion) UpdateInstancePassword(instId string, passwd string) error {
	return self.updateInstance(instId, "", "", passwd, "")
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(ivpcs); i++ {
		if ivpcs[i].GetGlobalId() == id {
			return ivpcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		zone := izones[i].(*SZone)
		if zone.Zone == id {
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

func (self *SRegion) _fetchZones() error {
	params := make(map[string]string)
	zones := make([]SZone, 0)
	body, err := self.cvmRequest("DescribeZones", params)
	if err != nil {
		return err
	}
	err = body.Unmarshal(&zones, "ZoneSet")
	if err != nil {
		return err
	}
	self.izones = make([]cloudprovider.ICloudZone, len(zones))
	for i := 0; i < len(zones); i++ {
		zones[i].region = self
		self.izones[i] = &zones[i]
	}
	return nil
}

func (self *SRegion) fetchInfrastructure() error {
	err := self._fetchZones()
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

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := make(map[string]string)
	params["VpcId"] = vpcId

	_, err := self.vpcRequest("DeleteVpc", params)
	return err
}

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	vpcs, total, err := self.GetVpcs([]string{vpcId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	vpcs[0].region = self
	return &vpcs[0], nil
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

func (self *SRegion) GetVpcs(vpcIds []string, offset int, limit int) ([]SVpc, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	if vpcIds != nil && len(vpcIds) > 0 {
		for index, vpcId := range vpcIds {
			params[fmt.Sprintf("VpcIds.%d", index)] = vpcId
		}
	}
	body, err := self.vpcRequest("DescribeVpcs", params)
	if err != nil {
		return nil, 0, err
	}
	vpcs := make([]SVpc, 0)
	err = body.Unmarshal(&vpcs, "VpcSet")
	if err != nil {
		log.Errorf("Unmarshal vpc fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return vpcs, int(total), nil
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.Region]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) GetStatus() string {
	if self.RegionState == "AVAILABLE" {
		return models.CLOUD_REGION_STATUS_INSERVER
	}
	return models.CLOUD_REGION_STATUS_OUTOFSERVICE
}

func (self *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (self *SRegion) vpcRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.vpcRequest(apiName, params)
}

func (self *SRegion) cvmRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.jsonRequest(apiName, params)
}

func (self *SRegion) cbsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.cbsRequest(apiName, params)
}

func (self *SRegion) GetNetworks(ids []string, vpcId string, offset int, limit int) ([]SNetwork, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	base := 0
	if ids != nil && len(ids) > 0 {
		for index, networkId := range ids {
			params[fmt.Sprintf("SubnetIds.%d", index)] = networkId
		}
		base += len(ids)
	}
	if len(vpcId) > 0 {
		params["Filters.0.Name"] = "vpc-id"
		params["Filters.0.Values.0"] = vpcId
	}

	body, err := self.vpcRequest("DescribeSubnets", params)
	if err != nil {
		log.Errorf("DescribeSubnets fail %s", err)
		return nil, 0, err
	}

	networks := make([]SNetwork, 0)
	err = body.Unmarshal(&networks, "SubnetSet")
	if err != nil {
		log.Errorf("Unmarshal network fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return networks, int(total), nil
}

func (self *SRegion) GetNetwork(networkId string) (*SNetwork, error) {
	networks, total, err := self.GetNetworks([]string{networkId}, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &networks[0], nil
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
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
		available = zone.getAvaliableInstanceTypes()
	}
	ret := make([]SInstanceType, 0)
	for _, t := range self.instanceTypes {
		if t.CPU == cpu && memMB == t.memoryMB() && gpu == t.GPU {
			if available == nil || utils.IsInStringArray(t.InstanceType, available) {
				ret = append(ret, t)
			}
		}
	}
	return ret, nil
}

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, networkId string, passwd string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.Zone)
		net := z.getNetworkById(networkId)
		if net != nil {
			desc := &cloudprovider.SManagedVMCreateConfig{
				Name:              name,
				ExternalImageId:   imgId,
				SysDisk:           cloudprovider.SDiskInfo{SizeGB: 0, StorageType: storageType},
				Cpu:               cpu,
				MemoryMB:          memGB * 1024,
				ExternalNetworkId: networkId,
				Password:          passwd,
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
	return nil, fmt.Errorf("cannot find network %s", networkId)
}

func (self *SRegion) instanceOperation(instanceId string, opname string, extra map[string]string) error {
	params := make(map[string]string)
	params["InstanceIds.0"] = instanceId
	if extra != nil && len(extra) > 0 {
		for k, v := range extra {
			params[k] = v
		}
	}
	_, err := self.cvmRequest(opname, params)
	return err
}

func (self *SRegion) DeleteSecurityGroup(vpcId string, secgroupId string) error {
	return self.deleteSecurityGroup(secgroupId)
}

func (self *SRegion) GetInstanceVNCUrl(instanceId string) (string, error) {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	body, err := self.cvmRequest("DescribeInstanceVncUrl", params)
	if err != nil {
		return "", err
	}
	return body.GetString("InstanceVncUrl")
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.InstanceState, nil
}

func (region *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotImplemented
}
